package worker_test

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/heuristic"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/parallels"
	"github.com/cirruslabs/cirrus-cli/internal/worker"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
	"os"
	"testing"
)

const (
	registrationToken = "registration token"
	sessionToken      = "session token"
	serverSecret      = "server secret"
	clientSecret      = "client secret"
)

func unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	// Only authenticate workers RPC methods
	if _, ok := info.Server.(*WorkersRPC); !ok {
		return handler(ctx, req)
	}

	// Allow Register() method to be unauthenticated
	if info.FullMethod == "/org.cirruslabs.ci.services.cirruscigrpc.CirrusWorkersService/Register" {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "failed to retrieve request metadata")
	}

	sessionTokens, ok := md["session-token"]
	if !ok || len(sessionTokens) == 0 {
		return nil, status.Errorf(codes.PermissionDenied, "no session token was provided")
	}

	if sessionTokens[0] != sessionToken {
		return nil, status.Errorf(codes.PermissionDenied, "invalid session token")
	}

	return handler(ctx, req)
}

func workerTestHelper(t *testing.T, lis net.Listener, isolation *api.Isolation, opts ...worker.Option) {
	// Start the RPC server
	server := grpc.NewServer(grpc.UnaryInterceptor(unaryInterceptor))

	workersRPC := &WorkersRPC{Isolation: isolation}
	api.RegisterCirrusWorkersServiceServer(server, workersRPC)
	tasksRPC := &TasksRPC{}
	api.RegisterCirrusCIServiceServer(server, tasksRPC)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Error(err)
		}
	}()

	var rearrangedOpts []worker.Option

	// Append lower-priority options first
	rearrangedOpts = append(rearrangedOpts, worker.WithRegistrationToken(registrationToken))

	// Only after lower-priority options
	// append high-priority user-specified options
	rearrangedOpts = append(rearrangedOpts, opts...)

	worker, err := worker.New(rearrangedOpts...)
	if err != nil {
		t.Fatal(err)
	}

	if err := worker.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	server.GracefulStop()

	assert.True(t, workersRPC.WorkerWasRegistered)
	assert.True(t, workersRPC.TaskWasAssigned)
	assert.True(t, workersRPC.TaskWasStarted)
	assert.True(t, workersRPC.TaskWasStopped)

	assert.Equal(t, []string{"clone", "check"}, tasksRPC.SucceededCommands)
}

func TestWorkerIsolationNone(t *testing.T) {
	// nolint:gosec // this is a test, so it's fine to bind on 0.0.0.0
	lis, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}

	endpoint := fmt.Sprintf("http://127.0.0.1:%d", lis.Addr().(*net.TCPAddr).Port)

	workerTestHelper(t, lis, nil, worker.WithRPCEndpoint(endpoint),
		worker.WithAgentDirectRPCEndpoint(endpoint))
}

func TestWorkerIsolationParallels(t *testing.T) {
	// Support Parallels isolation testing configured via environment variables
	image, imageOk := os.LookupEnv("CIRRUS_INTERNAL_PARALLELS_DARWIN_VM")
	user, userOk := os.LookupEnv("CIRRUS_INTERNAL_PARALLELS_DARWIN_SSH_USER")
	password, passwordOk := os.LookupEnv("CIRRUS_INTERNAL_PARALLELS_DARWIN_SSH_PASSWORD")
	if !imageOk || !userOk || !passwordOk {
		t.Skip("no Parallels credentials configured")
	}

	t.Logf("Using Parallels VM %s for testing...", image)
	sharedNetworkHostIP, err := parallels.SharedNetworkHostIP(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	lis, err := net.Listen("tcp", sharedNetworkHostIP+":0")
	if err != nil {
		t.Fatal(err)
	}

	isolation := &api.Isolation{
		Type: &api.Isolation_Parallels_{
			Parallels: &api.Isolation_Parallels{
				Image:    image,
				User:     user,
				Password: password,
				Platform: api.Platform_DARWIN,
			},
		},
	}

	endpoint := fmt.Sprintf("http://%s", lis.Addr().String())

	workerTestHelper(t, lis, isolation, worker.WithRPCEndpoint(endpoint), worker.WithAgentDirectRPCEndpoint(endpoint))
}

func TestWorkerIsolationContainer(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	lis, err := heuristic.NewListener(context.Background(), "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}

	isolation := &api.Isolation{
		Type: &api.Isolation_Container_{
			Container: &api.Isolation_Container{
				Image: "debian:latest",
			},
		},
	}

	var endpoint string

	switch addr := lis.Addr().(type) {
	case *net.TCPAddr:
		endpoint = fmt.Sprintf("http://127.0.0.1:%d", addr.Port)
	case *net.UnixAddr:
		endpoint = fmt.Sprintf("unix:%s", addr.String())
	default:
		t.Fatalf("unknown listener address type: %T", addr)
	}

	workerTestHelper(t, lis, isolation, worker.WithRPCEndpoint(endpoint),
		worker.WithAgentContainerRPCEndpoint(lis.ContainerEndpoint()))
}

func TestWorkerIsolationTart(t *testing.T) {
	// Support Tart isolation testing configured via environment variables
	vm, vmOk := os.LookupEnv("CIRRUS_INTERNAL_TART_VM")
	user, userOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_USER")
	password, passwordOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_PASSWORD")
	if !vmOk || !userOk || !passwordOk {
		t.Skip("no Tart credentials configured")
	}

	t.Logf("Using Tart VM %s for testing...", vm)

	// Virtualization.Framework only creates the network interface once the VM is running,
	// and binding to 127.0.0.1 wouldn't work in terms of reachability from the VM,
	// so this should be relatively OK, assuming we only do this in tests.
	// nolint:gosec
	lis, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}

	isolation := &api.Isolation{
		Type: &api.Isolation_Tart_{
			Tart: &api.Isolation_Tart{
				Vm:       vm,
				User:     user,
				Password: password,
			},
		},
	}

	listenerPort := lis.Addr().(*net.TCPAddr).Port
	rpcEndpoint := fmt.Sprintf("http://127.0.0.1:%d", listenerPort)
	agentRPCEndpoint := fmt.Sprintf("http://192.168.64.1:%d", listenerPort)

	workerTestHelper(t, lis, isolation, worker.WithRPCEndpoint(rpcEndpoint),
		worker.WithAgentDirectRPCEndpoint(agentRPCEndpoint))
}
