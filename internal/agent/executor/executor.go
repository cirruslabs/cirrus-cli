package executor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/cirrus-cli/internal/agent/cirrusenv"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/contextops"
	"github.com/cirruslabs/cirrus-cli/internal/agent/environment"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/terminalwrapper"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/updatebatcher"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/vaultunboxer"
	"github.com/cirruslabs/cirrus-cli/internal/agent/http_cache"
	agentstorage "github.com/cirruslabs/cirrus-cli/internal/agent/storage"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/samber/lo"
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

	slog.Info("Getting initial commands...")

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
			slog.Warn("Failed to get initial commands", "err", err)
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
		slog.Error("Server token is incorrect!")
		panic("Server token is incorrect!")
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

	slog.Info("Unboxing VAULT[...] environment variables, if any")

	var vaultUnboxer *vaultunboxer.VaultUnboxer

	for key, value := range vaultUnboxerEnv.Items() {
		boxedValue, err := vaultunboxer.NewBoxedValue(value)
		if err != nil {
			if errors.Is(err, vaultunboxer.ErrNotABoxedValue) {
				continue
			}

			message := fmt.Sprintf("failed to parse a Vault-boxed value %s: %v", value, err)
			slog.Error(message)
			executor.reportError(message)

			return
		}

		if vaultUnboxer == nil {
			slog.Info("Found at least one VAULT[...] environment variable, initializing Vault client")

			vaultUnboxer, err = vaultunboxer.NewFromEnvironment(ctx, vaultUnboxerEnv)
			if err != nil {
				message := fmt.Sprintf("failed to initialize a Vault client: %v", err)
				slog.Error(message)
				executor.reportError(message)

				return
			}

			slog.Info("Vault client successfully initialized")
		}

		unboxedValue, err := vaultUnboxer.Unbox(ctx, boxedValue)
		if err != nil {
			message := fmt.Sprintf("failed to unbox a Vault-boxed value %s: %v", value, err)
			slog.Error(message)
			executor.reportError(message)

			return
		}

		scriptEnvironment[key] = unboxedValue
		executor.env.AddSensitiveValues(unboxedValue)
	}

	executor.env.Merge(scriptEnvironment, false)

	workingDir, ok := executor.env.Lookup("CIRRUS_WORKING_DIR")
	if ok {
		slog.Info("Changing current working directory", "path", workingDir)

		EnsureFolderExists(workingDir)

		if err := os.Chdir(workingDir); err != nil {
			message := fmt.Sprintf("Failed to change current working directory to '%s': %v", workingDir, err)
			slog.Error(message)
			executor.reportError(message)

			return
		}
	} else {
		slog.Info("Not changing current working directory because CIRRUS_WORKING_DIR is not set")
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
		transport := http_cache.DefaultTransport()

		backend := agentstorage.NewCirrusStoreBackend(client.CirrusClient, client.CirrusTaskIdentification)
		httpCacheHost := http_cache.Start(ctx, transport, backend)

		executor.env.Set("CIRRUS_HTTP_CACHE_HOST", httpCacheHost)
	}

	executor.httpCacheHost = executor.env.Get("CIRRUS_HTTP_CACHE_HOST")

	// Normal timeout-bounded context
	timeout := time.Duration(response.TimeoutInSeconds) * time.Second

	timeoutCtx, timeoutCtxCancel := context.WithTimeoutCause(ctx, timeout, ErrTimedOut)
	defer timeoutCtxCancel()

	// Like timeout-bounded context, but extended by 5 minutes
	// to allow for "on_timeout:" user-defined instructions to succeed
	var extendedTimeoutCtx context.Context
	var extendedTimeoutCtxCancel context.CancelFunc

	executor.env.AddSensitiveValues(response.SecretsToMask...)

	if len(commands) == 0 {
		slog.Info("No commands to run, exiting!")

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

		slog.Info("Executing command", "name", command.Name)

		var stepCtx context.Context

		if command.ExecutionBehaviour == api.Command_ON_TIMEOUT || command.ExecutionBehaviour == api.Command_ALWAYS {
			if extendedTimeoutCtx == nil {
				extendedTimeoutCtx, extendedTimeoutCtxCancel = context.WithTimeout(context.Background(), 5*time.Minute)
				defer extendedTimeoutCtxCancel()
			}

			stepCtx = contextops.All(timeoutCtx, extendedTimeoutCtx)
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

		slog.Info("Command finished", "name", command.Name, "status", strings.ToLower(currentCommandStatus.String()))

		ub.Queue(&api.CommandResult{
			Name:            command.Name,
			Status:          currentCommandStatus,
			DurationInNanos: stepResult.Duration.Nanoseconds(),
			SignaledToExit:  stepResult.SignaledToExit,
		})
	}

	ub.Flush(ctx, executor.taskIdentification())

	slog.Info("Background commands to clean up after", "count", len(executor.backgroundCommands))
	for i := 0; i < len(executor.backgroundCommands); i++ {
		backgroundCommand := executor.backgroundCommands[i]
		slog.Info("Cleaning up after background command", "name", backgroundCommand.Name)
		err := backgroundCommand.Cmd.Process.Kill()
		if err != nil {
			backgroundCommand.Logs.Write([]byte(fmt.Sprintf("\nFailed to stop background script %s: %s!", backgroundCommand.Name, err)))
		}
		backgroundCommand.Logs.Finalize()
	}

	// Retrieve resource utilization metrics
	slog.Info("Retrieving resource utilization metrics...")

	metricsCancel()

	var resourceUtilization *api.ResourceUtilization

	select {
	case metricsResult := <-metricsResultChan:
		if resourceUtilization := metricsResult.ResourceUtilization; resourceUtilization != nil {
			slog.Info("Received metrics",
				"cpu_points", len(metricsResult.ResourceUtilization.CpuChart),
				"memory_points", len(metricsResult.ResourceUtilization.MemoryChart),
				"errors", len(metricsResult.Errors()))
		} else {
			slog.Info("Received no metrics (this OS/architecture likely doesn't support metric gathering)")
		}
		for _, err := range metricsResult.Errors() {
			message := fmt.Sprintf("Encountered an error while gathering resource utilization metrics: %v", err)
			slog.Warn(message)
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
		slog.Warn(message)
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

	slog.Info("Reporting that the agent has finished...")

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
			slog.Warn("Failed to report that the agent has finished, retrying...", "err", err)
		}),
		retry.Delay(10*time.Second),
		retry.Attempts(2),
		retry.Context(context.WithoutCancel(ctx)),
	); err != nil {
		slog.Error("Failed to report that the agent has finished", "err", err)
	}
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
		slog.Error(message)
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
			slog.Info("Started execution of background command",
				"index", len(executor.backgroundCommands),
				"name", currentStep.Name)
			success = true
		} else {
			slog.Error("Failed to create command line for background command",
				"name", currentStep.Name, "err", err)
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
				slog.Info(operation.Message)
				_, _ = fmt.Fprintln(logUploader, operation.Message)
			case *terminalwrapper.ExitOperation:
				success = operation.Success
				break WaitForTerminalInstructionFor
			}
		}
	default:
		slog.Warn("Unsupported instruction", "type", fmt.Sprintf("%T", instruction))
		success = false
	}

	cirrusEnvVariables, err := cirrusEnv.Consume()
	if err != nil {
		message := fmt.Sprintf("Failed collect CIRRUS_ENV subsystem results: %v", err)
		slog.Error(message)
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
		slog.Warn("Unsupported source", "type", fmt.Sprintf("%T", source))

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
