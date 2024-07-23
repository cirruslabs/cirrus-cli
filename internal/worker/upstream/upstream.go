package upstream

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/endpoint"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/grpchelper"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"time"
)

var (
	ErrFailed = errors.New("upstream failed")
)

const (
	DefaultRPCEndpoint = "https://grpc.cirrus-ci.com:443"

	defaultPollIntervalSeconds = 10

	// Ridiculously large per-call timout in case some upstream hangs
	// which might happen, but we've never experienced so far.
	defaultDeadlineInSeconds = 900
)

type Upstream struct {
	workerName        string
	registrationToken string
	sessionToken      string

	rpcEndpoint string
	rpcTarget   string
	rpcInsecure bool
	rpcClient   api.CirrusWorkersServiceClient

	agentEndpoint endpoint.Endpoint

	pollIntervalSeconds uint32

	logger logrus.FieldLogger

	connected bool
}

func New(workerName string, registrationToken string, opts ...Option) (*Upstream, error) {
	upstream := &Upstream{
		workerName:        workerName,
		registrationToken: registrationToken,

		pollIntervalSeconds: defaultPollIntervalSeconds,

		logger: logrus.New(),
	}

	// Apply options
	for _, opt := range opts {
		opt(upstream)
	}

	// Apply defaults
	if upstream.rpcEndpoint == "" {
		upstream.rpcEndpoint = DefaultRPCEndpoint
	}
	if upstream.agentEndpoint == nil {
		upstream.agentEndpoint = endpoint.NewRemote(DefaultRPCEndpoint)
	}

	// Sanity check
	if upstream.workerName == "" {
		return nil, fmt.Errorf("%w: must provide a worker name", ErrFailed)
	}
	if upstream.registrationToken == "" {
		return nil, fmt.Errorf("%w: must provide a registration token", ErrFailed)
	}

	// Parse endpoint
	upstream.rpcTarget, upstream.rpcInsecure = grpchelper.TransportSettings(upstream.rpcEndpoint)

	return upstream, nil
}

func (upstream *Upstream) WorkerName() string {
	return upstream.workerName
}

func (upstream *Upstream) AgentEndpoint() endpoint.Endpoint {
	return upstream.agentEndpoint
}

func (upstream *Upstream) PollIntervalSeconds() uint32 {
	return upstream.pollIntervalSeconds
}

func (upstream *Upstream) Name() string {
	return upstream.rpcEndpoint
}

func (upstream *Upstream) Connect(ctx context.Context) error {
	if upstream.connected {
		return nil
	}

	var rpcSecurity grpc.DialOption

	if upstream.rpcInsecure {
		rpcSecurity = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		tlsCredentials := credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS13,
		})
		rpcSecurity = grpc.WithTransportCredentials(tlsCredentials)
	}

	// https://github.com/grpc/grpc-go/blob/master/Documentation/concurrency.md
	conn, err := grpc.DialContext(ctx, upstream.rpcTarget, rpcSecurity,
		grpc.WithUnaryInterceptor(deadlineUnaryInterceptor(defaultDeadlineInSeconds*time.Second)),
	)
	if err != nil {
		return fmt.Errorf("%w: failed to dial upstream %s: %v",
			ErrFailed, upstream.Name(), err)
	}

	upstream.rpcClient = api.NewCirrusWorkersServiceClient(conn)

	upstream.connected = true

	return nil
}

func (upstream *Upstream) Register(ctx context.Context, workerInfo *api.WorkerInfo) error {
	// Check if we've already registered
	if upstream.sessionToken != "" {
		return nil
	}

	if err := upstream.Connect(ctx); err != nil {
		return err
	}

	response, err := upstream.rpcClient.Register(ctx, &api.RegisterRequest{
		WorkerInfo:        workerInfo,
		RegistrationToken: upstream.registrationToken,
	})
	if err != nil {
		return err
	}

	upstream.sessionToken = response.SessionToken

	upstream.logger.Infof("worker successfully registered on upstream %s", upstream.Name())

	return nil
}

func (upstream *Upstream) Poll(ctx context.Context, request *api.PollRequest) (*api.PollResponse, error) {
	if err := upstream.Connect(ctx); err != nil {
		return nil, err
	}

	upstream.logger.Debugf("polling upstream %s", upstream.Name())

	response, err := upstream.rpcClient.Poll(ctx, request, grpc.PerRPCCredentials(upstream))
	if err != nil {
		return nil, err
	}

	if response.PollIntervalInSeconds != 0 && response.PollIntervalInSeconds <= uint32(time.Hour.Seconds()) {
		upstream.pollIntervalSeconds = response.PollIntervalInSeconds
	}

	return response, nil
}

func (upstream *Upstream) TaskFailed(ctx context.Context, request *api.TaskFailedRequest) error {
	if err := upstream.Connect(ctx); err != nil {
		return err
	}

	_, err := upstream.rpcClient.TaskFailed(ctx, request, grpc.PerRPCCredentials(upstream))

	return err
}

func (upstream *Upstream) TaskStarted(ctx context.Context, request *api.WorkerTaskIdentification) error {
	if err := upstream.Connect(ctx); err != nil {
		return err
	}

	_, err := upstream.rpcClient.TaskStarted(ctx, request, grpc.PerRPCCredentials(upstream))

	return err
}

func (upstream *Upstream) TaskStopped(ctx context.Context, request *api.WorkerTaskIdentification) error {
	if err := upstream.Connect(ctx); err != nil {
		return err
	}

	_, err := upstream.rpcClient.TaskStopped(ctx, request, grpc.PerRPCCredentials(upstream))

	return err
}

func (upstream *Upstream) SetDisabled(ctx context.Context, disabled bool) error {
	if err := upstream.Connect(ctx); err != nil {
		return err
	}

	request := &api.UpdateStatusRequest{
		Disabled: disabled,
	}

	response, err := upstream.rpcClient.UpdateStatus(ctx, request, grpc.PerRPCCredentials(upstream))
	if err != nil {
		return fmt.Errorf("%w: failed to set disabled state on upstream %s: %v",
			ErrFailed, upstream.Name(), err)
	}

	if response.Disabled != disabled {
		return fmt.Errorf("%w: failed to set disabled state on upstream %s, expected %t, got %t",
			ErrFailed, upstream.Name(), disabled, response.Disabled)
	}

	return err
}

func (upstream *Upstream) QueryRunningTasks(
	ctx context.Context,
	request *api.QueryRunningTasksRequest,
) (*api.QueryRunningTasksResponse, error) {
	if err := upstream.Connect(ctx); err != nil {
		return nil, err
	}

	return upstream.rpcClient.QueryRunningTasks(ctx, request, grpc.PerRPCCredentials(upstream))
}
