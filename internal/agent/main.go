package agent

import (
	"context"
	"flag"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor"
	"github.com/cirruslabs/cirrus-cli/internal/agent/network"
	"github.com/cirruslabs/cirrus-cli/internal/agent/signalfilter"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/grpchelper"
	"github.com/getsentry/sentry-go"
	"github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func Run(args []string) {
	apiEndpointPtr := flag.String("api-endpoint", "https://grpc.cirrus-ci.com:443", "GRPC endpoint URL")
	taskIdPtr := flag.String("task-id", "0", "Task ID")
	clientTokenPtr := flag.String("client-token", "", "Secret token")
	serverTokenPtr := flag.String("server-token", "", "Secret token")
	versionFlag := flag.Bool("version", false, "display the version and exit")
	help := flag.Bool("help", false, "help flag")
	stopHook := flag.Bool("stop-hook", false, "pre stop flag")
	commandFromPtr := flag.String("command-from", "", "Command to star execution from (inclusive)")
	commandToPtr := flag.String("command-to", "", "Command to stop execution at (exclusive)")
	preCreatedWorkingDir := flag.String("pre-created-working-dir", "",
		"working directory to use when spawned via Persistent Worker")
	_ = flag.CommandLine.Parse(args)

	// Enrich future events with Cirrus CI-specific tags
	if tags, ok := os.LookupEnv("CIRRUS_SENTRY_TAGS"); ok {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			for _, tag := range strings.Split(tags, ",") {
				splits := strings.SplitN(tag, "=", 2)
				if len(splits) != 2 {
					continue
				}

				scope.SetTag(splits[0], splits[1])
			}
		})
	}

	// Propagate W3C Trace Context from the environment variables[1]
	//
	// [1]: https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/context/env-carriers.md
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), propagation.MapCarrier{
		"traceparent": os.Getenv("TRACEPARENT"),
		"tracestate":  os.Getenv("TRACESTATE"),
	})

	// Initialize logger
	logFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("cirrus-agent-%s.log", *taskIdPtr))
	if *stopHook {
		// In case of a failure the log file will be persisted on the machine for debugging purposes.
		// But unfortunately stop hook invocation will override it so let's use a different name.
		logFilePath = filepath.Join(os.TempDir(), fmt.Sprintf("cirrus-agent-%s-hook.log", *taskIdPtr))
	}
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
	} else {
		defer func() {
			logFilePos, err := logFile.Seek(0, io.SeekCurrent)
			if err != nil {
				log.Printf("Failed to determine the final log file size: %v", err)
			}

			log.Printf("Finalizing log file, %d bytes written", logFilePos)

			_ = logFile.Close()
			uploadAgentLogs(context.Background(), logFilePath, *taskIdPtr, *clientTokenPtr)
		}()
	}
	multiWriter := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(multiWriter)
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(multiWriter, multiWriter, multiWriter))

	// Handle panics
	defer func() {
		err := recover()
		if err == nil {
			return
		}

		// Report exception to Sentry
		hub := sentry.CurrentHub()
		hub.Recover(err)

		// Report exception to log file
		log.Printf("Recovered an error: %v", err)
		stack := string(debug.Stack())
		log.Println(stack)

		// Report exception to Cirrus CI
		if client.CirrusClient == nil {
			return
		}

		request := &api.ReportAgentProblemRequest{
			TaskIdentification: api.OldTaskIdentification(*taskIdPtr, *clientTokenPtr),
			Message:            fmt.Sprint(err),
			Stack:              stack,
		}
		_, err = client.CirrusClient.ReportAgentError(context.Background(), request)
		if err != nil {
			log.Printf("Failed to report agent error: %v\n", err)
		}
	}()

	if *versionFlag {
		fmt.Println(version.FullVersion)
		os.Exit(0)
	}

	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	var conn *grpc.ClientConn

	log.Printf("Running agent version %s", version.FullVersion)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel)
	go func() {
		limiter := rate.NewLimiter(1, 1)

		for {
			sig := <-signalChannel

			if sig == os.Interrupt || sig == syscall.SIGTERM {
				cancel()
			}

			if signalfilter.IsNoisy(sig) || !limiter.Allow() {
				continue
			}

			log.Printf("Captured %v...", sig)

			reportSignal(context.Background(), sig, *taskIdPtr, *clientTokenPtr)
		}
	}()

	// Prevent SENTRY_DSN propagation to scripts
	if err := os.Unsetenv("SENTRY_DSN"); err != nil {
		log.Printf("Failed to unset SENTRY_DSN: %v", err)
	}

	// Connect to the RPC server
	md := metadata.New(map[string]string{
		"org.cirruslabs.task-id":       *taskIdPtr,
		"org.cirruslabs.client-secret": *clientTokenPtr,
	})

	err = retry.Do(
		func() error {
			conn, err = dialWithTimeout(ctx, *apiEndpointPtr, md)
			return err
		}, retry.OnRetry(func(n uint, err error) {
			log.Printf("Failed to open a connection: %v\n", err)
		}),
		retry.Delay(1*time.Second), retry.MaxDelay(1*time.Second),
		retry.Attempts(0), retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
	if err != nil {
		// Context was cancelled before we had a chance to connect
		return
	}

	log.Printf("Connected!\n")

	client.InitClient(conn, *taskIdPtr, *clientTokenPtr)

	if *stopHook {
		log.Printf("Stop hook!\n")

		request := api.ReportStopHookRequest{
			TaskIdentification: client.CirrusTaskIdentification,
		}
		_, err = client.CirrusClient.ReportStopHook(ctx, &request)
		if err != nil {
			log.Printf("Failed to report stop hook for task %s: %v\n", *taskIdPtr, err)
		} else {
			logFile.Close()
			os.Remove(logFilePath)
		}
		os.Exit(0)
	}

	if portsToWait, ok := os.LookupEnv("CIRRUS_PORTS_WAIT_FOR"); ok {
		ports := strings.Split(portsToWait, ",")

		for _, port := range ports {
			portNumber, err := strconv.Atoi(port)
			if err != nil {
				continue
			}

			log.Printf("Waiting on port %v...\n", port)

			subCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			network.WaitForLocalPort(subCtx, portNumber)
			cancel()
		}
	}

	go runHeartbeat(*taskIdPtr, *clientTokenPtr, conn)

	buildExecutor := executor.NewExecutor(*taskIdPtr, *clientTokenPtr, *serverTokenPtr, *commandFromPtr, *commandToPtr,
		*preCreatedWorkingDir)
	buildExecutor.RunBuild(ctx)
}

func uploadAgentLogs(ctx context.Context, logFilePath string, taskId string, clientToken string) {
	if client.CirrusClient == nil {
		return
	}

	logContents, readErr := os.ReadFile(logFilePath)
	if readErr != nil {
		return
	}
	request := api.ReportAgentLogsRequest{
		TaskIdentification: api.OldTaskIdentification(taskId, clientToken),
		Logs:               string(logContents),
	}
	_, err := client.CirrusClient.ReportAgentLogs(ctx, &request)
	if err == nil {
		os.Remove(logFilePath)
	}
}

func reportSignal(ctx context.Context, sig os.Signal, taskId string, clientToken string) {
	if client.CirrusClient == nil {
		return
	}

	request := api.ReportAgentSignalRequest{
		TaskIdentification: api.OldTaskIdentification(taskId, clientToken),
		Signal:             sig.String(),
	}
	_, _ = client.CirrusClient.ReportAgentSignal(ctx, &request)
}

func dialWithTimeout(ctx context.Context, apiEndpoint string, md metadata.MD) (*grpc.ClientConn, error) {
	_, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	target, transportSecurity := grpchelper.TransportSettingsAsDialOption(apiEndpoint)

	retryCodes := []codes.Code{
		codes.Unavailable, codes.Internal, codes.Unknown, codes.ResourceExhausted, codes.DeadlineExceeded,
	}
	return grpc.NewClient(
		target,
		transportSecurity,
		grpc.WithKeepaliveParams(
			keepalive.ClientParameters{
				Time:                30 * time.Second, // make connection is alive every 30 seconds
				Timeout:             60 * time.Second, // with a timeout of 60 seconds
				PermitWithoutStream: true,             // always send Pings even if there are no RPCs
			},
		),
		grpc.WithChainUnaryInterceptor(
			grpc_retry.UnaryClientInterceptor(
				grpc_retry.WithMax(3),
				grpc_retry.WithCodes(retryCodes...),
				grpc_retry.WithPerRetryTimeout(60*time.Second),
			),
			metadataInterceptor(md),
		),
	)
}

func metadataInterceptor(md metadata.MD) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = metadata.NewOutgoingContext(ctx, md)

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func runHeartbeat(taskId string, clientToken string, conn *grpc.ClientConn) {
	for {
		log.Println("Sending heartbeat...")
		_, err := client.CirrusClient.Heartbeat(
			context.Background(),
			&api.HeartbeatRequest{TaskIdentification: api.OldTaskIdentification(taskId, clientToken)},
		)
		if err != nil {
			log.Printf("Failed to send heartbeat: %v", err)
			connectionState := conn.GetState()
			log.Printf("Connection state: %v", connectionState.String())
			if connectionState == connectivity.TransientFailure {
				conn.ResetConnectBackoff()
			}
		} else {
			log.Printf("Sent heartbeat!")
		}
		time.Sleep(60 * time.Second)
	}
}
