package executor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/cirrus-cli/internal/agent/cirrusenv"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/terminalwrapper"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/updatebatcher"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/vaultunboxer"
	"github.com/cirruslabs/cirrus-cli/internal/agent/fallbackroundtripper"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/samber/lo"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type CommandAndLogs struct {
	Name string
	Cmd  *exec.Cmd
	Logs *LogUploader
}

type Executor struct {
	taskId               string
	clientToken          string
	serverToken          string
	backgroundCommands   []CommandAndLogs
	httpCacheHost        string
	commandFrom          string
	commandTo            string
	preCreatedWorkingDir string
	cacheAttempts        *CacheAttempts
	env                  *environment.Environment
	terminalWrapper      *terminalwrapper.Wrapper
}

type StepResult struct {
	Success        bool
	SignaledToExit bool
	Duration       time.Duration
}

var (
	ErrStepExit = errors.New("executor step requested to terminate execution")
	ErrTimedOut = errors.New("timed out")
)

func NewExecutor(
	taskId string,
	clientToken string,
	serverToken string,
	commandFrom string,
	commandTo string,
	preCreatedWorkingDir string,
) *Executor {
	return &Executor{
		taskId:               taskId,
		clientToken:          clientToken,
		serverToken:          serverToken,
		backgroundCommands:   make([]CommandAndLogs, 0),
		httpCacheHost:        "",
		commandFrom:          commandFrom,
		commandTo:            commandTo,
		preCreatedWorkingDir: preCreatedWorkingDir,
		cacheAttempts:        NewCacheAttempts(),
		env:                  environment.NewEmpty(),
	}
}

func (executor *Executor) taskIdentification() *api.TaskIdentification {
	return api.OldTaskIdentification(executor.taskId, executor.clientToken)
}

func (executor *Executor) RunBuild(ctx context.Context) {
	// Start collecting metrics
	metricsCtx, metricsCancel := context.WithCancel(ctx)
	defer metricsCancel()
	metricsResultChan := metrics.Run(metricsCtx, nil)

	log.Println("Getting initial commands...")

	var response *api.CommandsResponse
	var err error
	var numRetries uint

	err = retry.Do(
		func() error {
			response, err = client.CirrusClient.InitialCommands(ctx, &api.InitialCommandsRequest{
				TaskIdentification:  executor.taskIdentification(),
				LocalTimestamp:      time.Now().Unix(),
				ContinueFromCommand: executor.commandFrom,
				Retry:               numRetries != 0,
			})
			return err
		}, retry.OnRetry(func(n uint, err error) {
			numRetries++
			log.Printf("Failed to get initial commands: %v", err)
		}),
		retry.Delay(5*time.Second),
		retry.Attempts(0), retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
	if err != nil {
		// Context was cancelled before we had a chance to get initial commands
		return
	}

	if response.ServerToken != executor.serverToken {
		log.Panic("Server token is incorrect!")
		return
	}

	// Retrieve the script/commands environment, but do not merge it into the
	// executor.env yet. We'll unbox the VAULT[...] environment variables first,
	// and overwrite the corresponding scriptEnvironment variables directly.
	//
	// This allows us to defer the expansion of variables pointing to Vault-boxed
	// variables, e.g.:
	//
	// env:
	//   HOSTNAME: VAULT[...]
	//   SERVER_FQDN: "${HOSTNAME}.local"
	scriptEnvironment := getScriptEnvironment(executor, response.Environment)

	// However, expand the environment just for the Vault-unboxer, so that
	// things like "CIRRUS_VAULT_URL: ${CIRRUS_VAULT_URL_GLOBAL}" and
	// "PASSWORD: VAULT[$PATH $ARGS]" would work.
	vaultUnboxerEnv := environment.New(scriptEnvironment)

	log.Println("Unboxing VAULT[...] environment variables, if any")

	var vaultUnboxer *vaultunboxer.VaultUnboxer

	for key, value := range vaultUnboxerEnv.Items() {
		boxedValue, err := vaultunboxer.NewBoxedValue(value)
		if err != nil {
			if errors.Is(err, vaultunboxer.ErrNotABoxedValue) {
				continue
			}

			message := fmt.Sprintf("failed to parse a Vault-boxed value %s: %v", value, err)
			log.Println(message)
			executor.reportError(message)

			return
		}

		if vaultUnboxer == nil {
			log.Println("Found at least one VAULT[...] environment variable, initializing Vault client")

			vaultUnboxer, err = vaultunboxer.NewFromEnvironment(ctx, vaultUnboxerEnv)
			if err != nil {
				message := fmt.Sprintf("failed to initialize a Vault client: %v", err)
				log.Println(message)
				executor.reportError(message)

				return
			}

			log.Println("Vault client successfully initialized")
		}

		unboxedValue, err := vaultUnboxer.Unbox(ctx, boxedValue)
		if err != nil {
			message := fmt.Sprintf("failed to unbox a Vault-boxed value %s: %v", value, err)
			log.Println(message)
			executor.reportError(message)

			return
		}

		scriptEnvironment[key] = unboxedValue
		executor.env.AddSensitiveValues(unboxedValue)
	}

	executor.env.Merge(scriptEnvironment, false)

	workingDir, ok := executor.env.Lookup("CIRRUS_WORKING_DIR")
	if ok {
		log.Printf("Changing current working directory to %s", workingDir)

		EnsureFolderExists(workingDir)

		if err := os.Chdir(workingDir); err != nil {
			message := fmt.Sprintf("Failed to change current working directory to '%s': %v", workingDir, err)
			log.Println(message)
			executor.reportError(message)

			return
		}
	} else {
		log.Printf("Not changing current working directory because CIRRUS_WORKING_DIR is not set")
	}

	commands := response.Commands

	// Prefer the HTTP cache host passed through the OS environment variables
	if cacheHost, ok := os.LookupEnv("CIRRUS_HTTP_CACHE_HOST"); ok {
		executor.env.Set("CIRRUS_HTTP_CACHE_HOST", cacheHost)
	}

	// Otherwise, if the HTTP cache host is not passed either through
	// the OS environment nor through the task's environment,
	// run our built-in cache server
	if _, ok := executor.env.Lookup("CIRRUS_HTTP_CACHE_HOST"); !ok {
		// Default HTTP cache
		transport := http_cache.DefaultTransport()

		httpCacheHost := http_cache.Start(ctx, http_cache.DefaultTransport(), false)

		executor.env.Set("CIRRUS_HTTP_CACHE_HOST", httpCacheHost)

		// Thunder HTTP cache
		chachaTransport, err := executor.tryChachaTransport()
		if err != nil {
			log.Printf("%v", err)
		}
		if chachaTransport != nil {
			// Propagate agent's context using W3C Trace Context
			// when sending requests using the Chacha transport
			chachaTransport = otelhttp.NewTransport(chachaTransport,
				otelhttp.WithPropagators(propagation.TraceContext{}),
			)

			// Always Chacha transport with a fallback to the default transport, just in case
			transport := fallbackroundtripper.New(chachaTransport, transport)

			httpCacheHostThunder := http_cache.Start(ctx, transport, true)

			executor.env.Set("CIRRUS_HTTP_CACHE_HOST_THUNDER", httpCacheHostThunder)
		}
	}

	executor.httpCacheHost = executor.env.Get("CIRRUS_HTTP_CACHE_HOST")

	// Normal timeout-bounded context
	timeout := time.Duration(response.TimeoutInSeconds) * time.Second

	timeoutCtx, timeoutCtxCancel := context.WithTimeoutCause(ctx, timeout, ErrTimedOut)
	defer timeoutCtxCancel()

	// Like timeout-bounded context, but extended by 5 minutes
	// to allow for "on_timeout:" user-defined instructions to succeed
	extendedTimeoutCtx, extendedTimeoutCtxCancel := context.WithTimeout(ctx, timeout+(5*time.Minute))
	defer extendedTimeoutCtxCancel()

	executor.env.AddSensitiveValues(response.SecretsToMask...)

	if len(commands) == 0 {
		log.Printf("No commands to run, exiting!")

		return
	}

	// Launch terminal session for remote access (in case requested by the user)
	var hasWaitForTerminalInstruction bool
	var terminalServerAddress string

	for _, command := range commands {
		if instruction, ok := command.Instruction.(*api.Command_WaitForTerminalInstruction); ok {
			hasWaitForTerminalInstruction = true
			if instruction.WaitForTerminalInstruction != nil {
				terminalServerAddress = instruction.WaitForTerminalInstruction.TerminalServerAddress
			}
			break
		}
	}

	if hasWaitForTerminalInstruction {
		expireIn := 15 * time.Minute

		expireInString, ok := executor.env.Lookup("CIRRUS_TERMINAL_EXPIRATION_WINDOW")
		if ok {
			expireInInt, err := strconv.Atoi(expireInString)
			if err == nil {
				expireIn = time.Duration(expireInInt) * time.Second
			}
		}

		shellEnv := append(os.Environ(), EnvMapAsSlice(executor.env.Items())...)

		executor.terminalWrapper = terminalwrapper.New(timeoutCtx, executor.taskIdentification(), terminalServerAddress,
			expireIn, shellEnv)
	}

	failedAtLeastOnce := response.FailedAtLeastOnce

	ub := updatebatcher.New()

	for _, command := range BoundedCommands(commands, executor.commandFrom, executor.commandTo) {
		shouldRun := (command.ExecutionBehaviour == api.Command_ON_SUCCESS && !failedAtLeastOnce) ||
			(command.ExecutionBehaviour == api.Command_ON_FAILURE && failedAtLeastOnce) ||
			command.ExecutionBehaviour == api.Command_ALWAYS ||
			(command.ExecutionBehaviour == api.Command_ON_TIMEOUT && errors.Is(timeoutCtx.Err(), context.DeadlineExceeded))
		if !shouldRun {
			ub.Queue(&api.CommandResult{
				Name:   command.Name,
				Status: api.Status_SKIPPED,
			})
			continue
		}

		ub.Queue(&api.CommandResult{
			Name:   command.Name,
			Status: api.Status_EXECUTING,
		})
		ub.Flush(ctx, executor.taskIdentification())

		log.Printf("Executing %s...", command.Name)

		var stepCtx context.Context

		if command.ExecutionBehaviour == api.Command_ON_TIMEOUT || command.ExecutionBehaviour == api.Command_ALWAYS {
			stepCtx = extendedTimeoutCtx
		} else {
			stepCtx = timeoutCtx
		}

		stepResult, err := executor.performStep(stepCtx, command)
		if err != nil {
			return
		}

		if !stepResult.Success {
			failedAtLeastOnce = true
		}

		var currentCommandStatus api.Status
		if stepResult.Success {
			currentCommandStatus = api.Status_COMPLETED
		} else {
			currentCommandStatus = api.Status_FAILED
		}

		log.Printf("%s %s!", command.Name, strings.ToLower(currentCommandStatus.String()))

		ub.Queue(&api.CommandResult{
			Name:            command.Name,
			Status:          currentCommandStatus,
			DurationInNanos: stepResult.Duration.Nanoseconds(),
			SignaledToExit:  stepResult.SignaledToExit,
		})
	}

	ub.Flush(ctx, executor.taskIdentification())

	log.Printf("Background commands to clean up after: %d!\n", len(executor.backgroundCommands))
	for i := 0; i < len(executor.backgroundCommands); i++ {
		backgroundCommand := executor.backgroundCommands[i]
		log.Printf("Cleaning up after background command %s...\n", backgroundCommand.Name)
		err := backgroundCommand.Cmd.Process.Kill()
		if err != nil {
			backgroundCommand.Logs.Write([]byte(fmt.Sprintf("\nFailed to stop background script %s: %s!", backgroundCommand.Name, err)))
		}
		backgroundCommand.Logs.Finalize()
	}

	// Retrieve resource utilization metrics
	log.Println("Retrieving resource utilization metrics...")

	metricsCancel()

	var resourceUtilization *api.ResourceUtilization

	select {
	case metricsResult := <-metricsResultChan:
		if resourceUtilization := metricsResult.ResourceUtilization; resourceUtilization != nil {
			log.Printf("Received metrics: %d CPU points, %d memory points and %d errors\n",
				len(metricsResult.ResourceUtilization.CpuChart), len(metricsResult.ResourceUtilization.MemoryChart),
				len(metricsResult.Errors()))
		} else {
			log.Println("Received no metrics (this OS/architecture likely doesn't support metric gathering)")
		}
		for _, err := range metricsResult.Errors() {
			message := fmt.Sprintf("Encountered an error while gathering resource utilization metrics: %v", err)
			log.Println(message)
			_, _ = client.CirrusClient.ReportAgentWarning(ctx, &api.ReportAgentProblemRequest{
				TaskIdentification: executor.taskIdentification(),
				Message:            message,
			})
		}
		resourceUtilization = metricsResult.ResourceUtilization
	case <-time.After(3 * time.Second):
		// Yes, we already use context.Context, but it seems that gopsutil is somewhat lacking it's support[1],
		// so we err on the side of caution here.
		//
		// [1]: https://github.com/shirou/gopsutil/issues/724
		message := "Failed to retrieve resource utilization metrics in time"
		log.Println(message)
		_, _ = client.CirrusClient.ReportAgentWarning(ctx, &api.ReportAgentProblemRequest{
			TaskIdentification: executor.taskIdentification(),
			Message:            message,
		})
	}

	// Emit a warning if multi-line secrets were used[1]
	//
	// [1]: https://github.com/cirruslabs/cirrus-cli/issues/729
	hasMultiLineSecretValues := lo.ContainsBy(executor.env.SensitiveValues(), func(value string) bool {
		return strings.Contains(value, "\n")
	})
	if hasMultiLineSecretValues {
		_, _ = client.CirrusClient.ReportAgentWarning(ctx, &api.ReportAgentProblemRequest{
			TaskIdentification: executor.taskIdentification(),
			Message:            "Found multi-line secret values, masking them would not work",
		})
	}

	log.Println("Reporting that the agent has finished...")

	if err = retry.Do(
		func() error {
			_, err = client.CirrusClient.ReportAgentFinished(context.WithoutCancel(ctx),
				&api.ReportAgentFinishedRequest{
					TaskIdentification:     executor.taskIdentification(),
					CacheRetrievalAttempts: executor.cacheAttempts.ToProto(),
					ResourceUtilization:    resourceUtilization,
					CommandResults:         ub.History(),
				})
			return err
		}, retry.OnRetry(func(n uint, err error) {
			log.Printf("Failed to report that the agent has finished: %v\nRetrying...\n", err)
		}),
		retry.Delay(10*time.Second),
		retry.Attempts(2),
		retry.Context(context.WithoutCancel(ctx)),
	); err != nil {
		log.Printf("Failed to report that the agent has finished: %v\n", err)
	}
}

func (executor *Executor) tryChachaTransport() (http.RoundTripper, error) {
	if _, ok := executor.env.Lookup("CIRRUS_CHACHA_ENABLED"); !ok {
		log.Printf("not initializing Chacha transport because CIRRUS_CHACHA_ENABLED is not set")

		return nil, nil
	}

	chachaAddr, ok := os.LookupEnv("CIRRUS_CHACHA_ADDR")
	if !ok {
		return nil, fmt.Errorf("failed to initialize Chacha transport: " +
			"Chacha is enabled, but no CIRRUS_CHACHA_ADDR is set")
	}

	chachaURL, err := url.Parse(chachaAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Chacha transport: "+
			"failed to parse Chacha address %q: %w", chachaAddr, err)
	}

	chachaTransport := &http.Transport{
		Proxy: func(request *http.Request) (*url.URL, error) {
			// Chacha might issue a redirect and ask the client to connect
			// directly to the storage server (without using a proxy)
			if request.Response != nil && request.Response.Header.Get("X-Chacha-Direct-Connect") == "1" {
				return nil, nil
			}

			return chachaURL, nil
		},

		// Disable compression when using Chacha, as the latter
		// does not support Vary header yet, which means that
		// any presence of this header causes the responses
		// to be non-cacheable.
		//
		// Currently on Chacha we get an "Accept-Encoding: gzip"
		// request and a "Vary: Accept-Encoding" response.
		//
		// When compression is disabled, we should not get
		// "Vary" in responses because no "Accept-Encoding"
		// will be set.
		DisableCompression: true,
	}

	if chachaCert, ok := os.LookupEnv("CIRRUS_CHACHA_CERT"); ok {
		certPool := x509.NewCertPool()

		if certPool.AppendCertsFromPEM([]byte(chachaCert)) {
			chachaTransport.TLSClientConfig = &tls.Config{
				RootCAs: certPool,
			}
		} else {
			return nil, fmt.Errorf("failed to initialize Chacha transport: " +
				"failed to add Chacha PEM certificate to root CA pool")
		}
	}

	log.Printf("Chacha transport initialized")

	return chachaTransport, nil
}

// BoundedCommands bounds a slice of commands with unique names to a half-open range [fromName, toName).
func BoundedCommands(commands []*api.Command, fromName, toName string) []*api.Command {
	left, right := 0, len(commands)

	for i, command := range commands {
		if fromName != "" && command.Name == fromName {
			left = i
		}

		if toName != "" && command.Name == toName {
			right = i
		}
	}

	return commands[left:right]
}

func getScriptEnvironment(executor *Executor, responseEnvironment map[string]string) map[string]string {
	if responseEnvironment == nil {
		responseEnvironment = make(map[string]string)
	}

	if _, ok := responseEnvironment["OS"]; !ok {
		if _, ok := os.LookupEnv("OS"); !ok {
			responseEnvironment["OS"] = runtime.GOOS
		}
	}
	responseEnvironment["CIRRUS_OS"] = runtime.GOOS
	responseEnvironment["CIRRUS_ARCH"] = runtime.GOARCH

	// Use directory created by the persistent worker if CIRRUS_WORKING_DIR
	// was not overridden in the task specification by the user
	_, hasWorkingDir := responseEnvironment["CIRRUS_WORKING_DIR"]
	if !hasWorkingDir && executor.preCreatedWorkingDir != "" {
		responseEnvironment["CIRRUS_WORKING_DIR"] = executor.preCreatedWorkingDir
	}

	if _, ok := responseEnvironment["CIRRUS_WORKING_DIR"]; !ok {
		defaultTempDirPath := filepath.Join(os.TempDir(), "cirrus-ci-build")
		if _, err := os.Stat(defaultTempDirPath); os.IsNotExist(err) {
			responseEnvironment["CIRRUS_WORKING_DIR"] = filepath.ToSlash(defaultTempDirPath)
		} else if executor.commandFrom != "" {
			// Default folder exists and we continue execution. Therefore we need to use it.
			responseEnvironment["CIRRUS_WORKING_DIR"] = filepath.ToSlash(defaultTempDirPath)
		} else {
			uniqueTempDirPath, _ := os.MkdirTemp(os.TempDir(), fmt.Sprintf("cirrus-task-%s", executor.taskId))
			responseEnvironment["CIRRUS_WORKING_DIR"] = filepath.ToSlash(uniqueTempDirPath)
		}
	}

	return responseEnvironment
}

func (executor *Executor) performStep(ctx context.Context, currentStep *api.Command) (*StepResult, error) {
	success := false
	signaledToExit := false
	start := time.Now()

	logUploader, err := NewLogUploader(ctx, executor, currentStep.Name)
	if err != nil {
		message := fmt.Sprintf("Failed to initialize command %s log upload: %v", currentStep.Name, err)

		_, _ = client.CirrusClient.ReportAgentWarning(ctx, &api.ReportAgentProblemRequest{
			TaskIdentification: executor.taskIdentification(),
			Message:            message,
		})

		return &StepResult{
			Success:  false,
			Duration: time.Since(start),
		}, nil
	}

	if _, ok := currentStep.Instruction.(*api.Command_BackgroundScriptInstruction); !ok {
		defer logUploader.Finalize()
	}

	cirrusEnv, err := cirrusenv.New(executor.taskId)
	if err != nil {
		message := fmt.Sprintf("Failed initialize CIRRUS_ENV subsystem: %v", err)
		log.Print(message)
		fmt.Fprintln(logUploader, message)
		return &StepResult{
			Success:  false,
			Duration: time.Since(start),
		}, nil
	}
	defer cirrusEnv.Close()
	executor.env.Set("CIRRUS_ENV", cirrusEnv.Path())

	switch instruction := currentStep.Instruction.(type) {
	case *api.Command_ExitInstruction:
		return nil, ErrStepExit
	case *api.Command_CloneInstruction:
		success = CloneRepository(ctx, logUploader, executor.env)
	case *api.Command_FileInstruction:
		success = executor.CreateFile(ctx, logUploader, instruction.FileInstruction, executor.env)
	case *api.Command_ScriptInstruction:
		cmd, err := executor.ExecuteScriptsStreamLogsAndWait(ctx, logUploader, currentStep.Name,
			instruction.ScriptInstruction.Scripts, executor.env)
		success = err == nil && cmd.ProcessState.Success()
		if err == nil {
			if ws, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok {
				signaledToExit = ws.Signaled()
			}
		}
		if errors.Is(err, ErrTimedOut) {
			signaledToExit = false
		}
	case *api.Command_BackgroundScriptInstruction:
		cmd, err := executor.ExecuteScriptsAndStreamLogs(ctx, logUploader,
			instruction.BackgroundScriptInstruction.Scripts, executor.env)
		if err == nil {
			executor.backgroundCommands = append(executor.backgroundCommands, CommandAndLogs{
				Name: currentStep.Name,
				Cmd:  cmd,
				Logs: logUploader,
			})
			log.Printf("Started execution of #%d background command %s\n", len(executor.backgroundCommands), currentStep.Name)
			success = true
		} else {
			log.Printf("Failed to create command line for background command %s: %s\n", currentStep.Name, err)
			_, _ = logUploader.Write([]byte(fmt.Sprintf("Failed to create command line: %s", err)))
			logUploader.Finalize()
			success = false
		}
	case *api.Command_CacheInstruction:
		success = executor.DownloadCache(ctx, logUploader, currentStep.Name, executor.httpCacheHost,
			instruction.CacheInstruction, executor.env)
	case *api.Command_UploadCacheInstruction:
		success = executor.UploadCache(ctx, logUploader, currentStep.Name, executor.httpCacheHost,
			instruction.UploadCacheInstruction)
	case *api.Command_ArtifactsInstruction:
		success = executor.UploadArtifacts(ctx, logUploader, currentStep.Name,
			instruction.ArtifactsInstruction, executor.env)
	case *api.Command_WaitForTerminalInstruction:
		operationChan := executor.terminalWrapper.Wait()

	WaitForTerminalInstructionFor:
		for {
			switch operation := (<-operationChan).(type) {
			case *terminalwrapper.LogOperation:
				log.Println(operation.Message)
				_, _ = fmt.Fprintln(logUploader, operation.Message)
			case *terminalwrapper.ExitOperation:
				success = operation.Success
				break WaitForTerminalInstructionFor
			}
		}
	default:
		log.Printf("Unsupported instruction %T", instruction)
		success = false
	}

	cirrusEnvVariables, err := cirrusEnv.Consume()
	if err != nil {
		message := fmt.Sprintf("Failed collect CIRRUS_ENV subsystem results: %v", err)
		log.Print(message)
		fmt.Fprintln(logUploader, message)
	}

	// Pick up new CIRRUS_ENV variables
	_, isSensitive := executor.env.Lookup("CIRRUS_ENV_SENSITIVE")
	executor.env.Merge(cirrusEnvVariables, isSensitive)

	return &StepResult{
		Success:        success,
		SignaledToExit: signaledToExit,
		Duration:       time.Since(start),
	}, nil
}

func (executor *Executor) ExecuteScriptsStreamLogsAndWait(
	ctx context.Context,
	logUploader *LogUploader,
	commandName string,
	scripts []string,
	env *environment.Environment) (*exec.Cmd, error) {
	cmd, err := ShellCommandsAndWait(ctx, scripts, env, func(bytes []byte) (int, error) {
		return logUploader.Write(bytes)
	}, executor.shouldKillProcesses())
	return cmd, err
}

func (executor *Executor) ExecuteScriptsAndStreamLogs(
	ctx context.Context,
	logUploader *LogUploader,
	scripts []string,
	env *environment.Environment,
) (*exec.Cmd, error) {
	sc, err := NewShellCommands(ctx, scripts, env, func(bytes []byte) (int, error) {
		return logUploader.Write(bytes)
	})
	var cmd *exec.Cmd
	if sc != nil {
		cmd = sc.cmd
	}
	return cmd, err
}

func (executor *Executor) CreateFile(
	ctx context.Context,
	logUploader *LogUploader,
	instruction *api.FileInstruction,
	env *environment.Environment,
) bool {
	var content string

	switch source := instruction.GetSource().(type) {
	case *api.FileInstruction_FromEnvironmentVariable:
		var isProvided bool

		content, isProvided = env.Lookup(source.FromEnvironmentVariable)
		if !isProvided {
			logUploader.Write([]byte(fmt.Sprintf("Environment variable %s is not set! Skipping file creation...",
				source.FromEnvironmentVariable)))

			return true
		}

		if strings.HasPrefix(content, "ENCRYPTED") {
			logUploader.Write([]byte(fmt.Sprintf("Environment variable %s wasn't decrypted! Skipping file creation...",
				source.FromEnvironmentVariable)))

			return true
		}
	case *api.FileInstruction_FromContents:
		content = source.FromContents
	default:
		log.Printf("Unsupported source %T", source)

		return false
	}

	filePath := env.ExpandText(instruction.DestinationPath)
	EnsureFolderExists(filepath.Dir(filePath))
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		logUploader.Write([]byte(fmt.Sprintf("Failed to write file %s: %s!", filePath, err)))
		return false
	}

	logUploader.Write([]byte(fmt.Sprintf("Created file %s!", filePath)))

	return true
}

func (executor *Executor) shouldKillProcesses() bool {
	_, shouldNotKillProcesses := executor.env.Lookup("CIRRUS_ESCAPING_PROCESSES")

	return !shouldNotKillProcesses
}

func retryableCloneError(err error) bool {
	if err == nil {
		return false
	}
	errorMessage := strings.ToLower(err.Error())
	if strings.Contains(errorMessage, "timeout") {
		return true
	}
	if strings.Contains(errorMessage, "timed out") {
		return true
	}
	if strings.Contains(errorMessage, "tls") {
		return true
	}
	if strings.Contains(errorMessage, "connection") {
		return true
	}
	if strings.Contains(errorMessage, "authentication") {
		return true
	}
	if strings.Contains(errorMessage, "not found") {
		return true
	}
	if strings.Contains(errorMessage, "short write") {
		return true
	}
	return false
}

func (executor *Executor) reportError(message string) {
	request := api.ReportAgentProblemRequest{
		TaskIdentification: executor.taskIdentification(),
		Message:            message,
	}
	_, _ = client.CirrusClient.ReportAgentError(context.Background(), &request)
}
