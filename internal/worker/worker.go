package worker

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/certifi/gocertifi"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-ci-agent/pkg/grpchelper"
	"github.com/cirruslabs/cirrus-cli/internal/executor/endpoint"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	DefaultRPCEndpoint         = "https://grpc.cirrus-ci.com:443"
	defaultPollIntervalSeconds = 10
)

var (
	ErrWorker   = errors.New("worker failed")
	ErrShutdown = errors.New("worker is shutting down")
)

type Worker struct {
	rpcEndpoint string
	rpcTarget   string
	rpcInsecure bool
	rpcClient   api.CirrusWorkersServiceClient

	agentEndpoint endpoint.Endpoint

	name                string
	userSpecifiedLabels map[string]string
	pollIntervalSeconds uint32

	registrationToken string
	sessionToken      string

	tasks           map[int64]context.CancelFunc
	taskCompletions chan int64

	logger logrus.FieldLogger
}

func New(opts ...Option) (*Worker, error) {
	worker := &Worker{
		rpcEndpoint:   DefaultRPCEndpoint,
		agentEndpoint: endpoint.NewRemote(DefaultRPCEndpoint),

		userSpecifiedLabels: make(map[string]string),
		pollIntervalSeconds: defaultPollIntervalSeconds,

		tasks:           make(map[int64]context.CancelFunc),
		taskCompletions: make(chan int64),

		logger: logrus.New(),
	}

	// Apply options
	for _, opt := range opts {
		opt(worker)
	}

	// Parse endpoint
	worker.rpcTarget, worker.rpcInsecure = grpchelper.TransportSettings(worker.rpcEndpoint)

	if worker.registrationToken == "" {
		return nil, fmt.Errorf("%w: must provide a registration token", ErrWorker)
	}

	return worker, nil
}

func (worker *Worker) info() *api.WorkerInfo {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}

	const (
		ReservedLabelName         = "name"
		ReservedLabelVersion      = "version"
		ReservedLabelHostname     = "hostname"
		ReservedLabelOS           = "os"
		ReservedLabelArchitecture = "arch"
	)

	// Create base labels
	labels := map[string]string{
		ReservedLabelName:         worker.name,
		ReservedLabelVersion:      version.FullVersion,
		ReservedLabelHostname:     hostname,
		ReservedLabelOS:           runtime.GOOS,
		ReservedLabelArchitecture: runtime.GOARCH,
	}

	// Merge with the user specified labels
	for key, value := range worker.userSpecifiedLabels {
		if _, alreadyExists := labels[key]; !alreadyExists {
			labels[key] = value
		}
	}

	return &api.WorkerInfo{
		Labels: labels,
	}
}

// https://github.com/cirruslabs/cirrus-cli/issues/357
func (worker *Worker) oldWorkingDirectoryCleanup() {
	// Fix tests failing due to /tmp/cirrus-ci-build removal
	if _, runningInCi := os.LookupEnv("CIRRUS_CI"); runningInCi {
		return
	}

	tmpDir := os.TempDir()

	// Clean-up static directory[1]
	//
	// nolint:lll
	// [1]: https://github.com/cirruslabs/cirrus-ci-agent/blob/f88afe342106a6691d9e5b2d2e9187080c69fd2d/internal/executor/executor.go#L190
	staticWorkingDir := filepath.Join(tmpDir, "cirrus-ci-build")
	if err := os.RemoveAll(staticWorkingDir); err != nil {
		worker.logger.Infof("failed to clean up old cirrus-ci-build static working directory %s: %v",
			staticWorkingDir, err)
	}

	// Clean-up dynamic directories[1]
	//
	// nolint:lll
	// [1]: https://github.com/cirruslabs/cirrus-ci-agent/blob/f88afe342106a6691d9e5b2d2e9187080c69fd2d/internal/executor/executor.go#L197
	entries, err := ioutil.ReadDir(tmpDir)
	if err != nil {
		worker.logger.Infof("failed to clean up old cirrus-task-* dynamic working directories: %v", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), "cirrus-task-") {
			dynamicWorkingDir := filepath.Join(tmpDir, entry.Name())

			if err := os.RemoveAll(dynamicWorkingDir); err != nil {
				worker.logger.Infof("failed to clean up old cirrus-task-* dynamic working directory %s: %v",
					dynamicWorkingDir, err)
			}
		}
	}
}

func (worker *Worker) Run(ctx context.Context) error {
	// https://github.com/cirruslabs/cirrus-cli/issues/357
	worker.oldWorkingDirectoryCleanup()

	// A sub-context to cancel out all Run() side-effects when it finishes
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	connCancel, err := worker.initializeConnection(subCtx)
	if err != nil {
		worker.logger.Errorf("failed to dial %s: %v", worker.rpcEndpoint, err)
		return err
	}
	defer connCancel()

	for {
		if worker.sessionToken == "" {
			if err := worker.register(subCtx); err != nil {
				worker.logger.Errorf("failed to register worker: %v", err)
			}
		} else {
			err := worker.poll(subCtx)

			if errors.Is(err, ErrShutdown) {
				return nil
			}

			if err != nil {
				worker.logger.Errorf("failed to poll: %v", err)
			}
		}

		select {
		case <-subCtx.Done():
			return nil
		case <-time.After(time.Duration(worker.pollIntervalSeconds) * time.Second):
			// continue the loop
		}
	}
}

func (worker *Worker) initializeConnection(subCtx context.Context) (func(), error) {
	var rpcSecurity grpc.DialOption

	if worker.rpcInsecure {
		rpcSecurity = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		certPool, _ := gocertifi.CACerts()
		tlsCredentials := credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS13,
			RootCAs:    certPool,
		})
		rpcSecurity = grpc.WithTransportCredentials(tlsCredentials)
	}

	// https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	conn, err := grpc.DialContext(subCtx, worker.rpcTarget, rpcSecurity)
	if err == nil {
		worker.rpcClient = api.NewCirrusWorkersServiceClient(conn)
	}
	return func() {
		_ = conn.Close()
	}, err
}

func (worker *Worker) register(ctx context.Context) error {
	response, err := worker.rpcClient.Register(ctx, &api.RegisterRequest{
		WorkerInfo:        worker.info(),
		RegistrationToken: worker.registrationToken,
	})
	if err != nil {
		return err
	}

	worker.sessionToken = response.SessionToken

	worker.logger.Infof("worker successfully registered")

	return nil
}

func (worker *Worker) poll(ctx context.Context) error {
	// De-register completed tasks
	worker.registerTaskCompletions()

	worker.logger.Debugf("polling %s", worker.rpcEndpoint)

	request := &api.PollRequest{
		WorkerInfo:   worker.info(),
		RunningTasks: worker.runningTasks(),
	}

	response, err := worker.rpcClient.Poll(ctx, request, grpc.PerRPCCredentials(worker))
	if err != nil {
		return err
	}

	if response.Shutdown {
		worker.logger.Info("received shutdown signal from the server, terminating...")
		return ErrShutdown
	}

	for _, taskToStop := range response.TasksToStop {
		worker.stopTask(taskToStop)
	}

	for _, taskToStart := range response.TasksToStart {
		worker.runTask(ctx, taskToStart)
	}

	if response.PollIntervalInSeconds != 0 && response.PollIntervalInSeconds <= uint32(time.Hour.Seconds()) {
		worker.pollIntervalSeconds = response.PollIntervalInSeconds
	}

	return nil
}

// PerRPCCredentials interface implementation.
func (worker *Worker) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"registration-token": worker.registrationToken,
		"session-token":      worker.sessionToken,
		"worker-name":        worker.name,
	}, nil
}

// PerRPCCredentials interface implementation.
func (worker *Worker) RequireTransportSecurity() bool {
	return !worker.rpcInsecure
}

func (worker *Worker) Pause(ctx context.Context, wait bool) error {
	// A sub-context to cancel out all Run() side-effects when it finishes
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	connCancel, err := worker.initializeConnection(subCtx)
	if err != nil {
		worker.logger.Errorf("failed to dial %s: %v", worker.rpcEndpoint, err)
		return err
	}
	defer connCancel()

	_, err = worker.rpcClient.UpdateStatus(ctx, &api.UpdateStatusRequest{Disabled: true}, grpc.PerRPCCredentials(worker))
	if err != nil {
		return err
	}
	if !wait {
		return nil
	}
	for {
		response, err := worker.rpcClient.QueryRunningTasks(
			ctx, &api.QueryRunningTasksRequest{}, grpc.PerRPCCredentials(worker),
		)
		if err != nil {
			worker.logger.Errorf("Ignoring an API error while waiting: %w", err)
		} else if len(response.RunningTasks) == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Duration(worker.pollIntervalSeconds) * time.Second):
			// continue the loop
		}
	}
}

func (worker *Worker) Resume(ctx context.Context) error {
	// A sub-context to cancel out all Run() side-effects when it finishes
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	connCancel, err := worker.initializeConnection(subCtx)
	if err != nil {
		worker.logger.Errorf("failed to dial %s: %v", worker.rpcEndpoint, err)
		return err
	}
	defer connCancel()
	_, err = worker.rpcClient.UpdateStatus(ctx, &api.UpdateStatusRequest{Disabled: false}, grpc.PerRPCCredentials(worker))
	return err
}
