package worker

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/certifi/gocertifi"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/version"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"os"
	"runtime"
	"time"
)

const (
	defaultRPCEndpoint         = "grpc.cirrus-ci.com:443"
	defaultPollIntervalSeconds = 10
)

var (
	ErrWorker   = errors.New("worker failed")
	ErrShutdown = errors.New("worker is shutting down")
)

type Worker struct {
	rpcEndpoint string
	rpcInsecure bool
	rpcClient   api.CirrusWorkersServiceClient

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
		rpcEndpoint: defaultRPCEndpoint,

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

func (worker *Worker) Run(ctx context.Context) error {
	var rpcSecurity grpc.DialOption

	if worker.rpcInsecure {
		rpcSecurity = grpc.WithInsecure()
	} else {
		certPool, _ := gocertifi.CACerts()
		tlsCredentials := credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS13,
			RootCAs:    certPool,
		})
		rpcSecurity = grpc.WithTransportCredentials(tlsCredentials)
	}

	// https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	conn, err := grpc.DialContext(ctx, worker.rpcEndpoint, rpcSecurity)
	if err != nil {
		worker.logger.Errorf("failed to dial %s: %v", worker.rpcEndpoint, err)
	}
	worker.rpcClient = api.NewCirrusWorkersServiceClient(conn)
	defer conn.Close()

	for {
		if worker.sessionToken == "" {
			if err := worker.register(ctx); err != nil {
				worker.logger.Errorf("failed to register worker: %v", err)
			}
		} else {
			err := worker.poll(ctx)

			if errors.Is(err, ErrShutdown) {
				return nil
			}

			if err != nil {
				worker.logger.Errorf("failed to poll: %v", err)
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Duration(worker.pollIntervalSeconds) * time.Second):
			// continue the loop
		}
	}
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

	worker.logger.Infof("polling %s", worker.rpcEndpoint)

	request := &api.PollRequest{
		WorkerInfo:   worker.info(),
		RunningTasks: worker.runningTasks(),
	}

	response, err := worker.rpcClient.Poll(ctx, request)
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
