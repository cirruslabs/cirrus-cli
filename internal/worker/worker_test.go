package worker_test

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/endpoint"
	"github.com/cirruslabs/cirrus-cli/internal/executor/heuristic"
	"github.com/cirruslabs/cirrus-cli/internal/worker"
	"github.com/cirruslabs/cirrus-cli/internal/worker/upstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func workerTestHelper(
	t *testing.T,
	lis net.Listener,
	isolation *api.Isolation,
	checkScripts []string,
	opts ...worker.Option,
) {
	// Start the RPC server
	server := grpc.NewServer(grpc.UnaryInterceptor(unaryInterceptor))

	workersRPC := &WorkersRPC{Isolation: isolation}
	api.RegisterCirrusWorkersServiceServer(server, workersRPC)
	tasksRPC := NewTasksRPC(checkScripts)
	api.RegisterCirrusCIServiceServer(server, tasksRPC)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Error(err)
		}
	}()

	worker, err := worker.New(opts...)
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
	//nolint:gosec // this is a test, so it's fine to bind on 0.0.0.0
	lis, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}

	apiEndpoint := fmt.Sprintf("http://127.0.0.1:%d", lis.Addr().(*net.TCPAddr).Port)
	upstream, err := upstream.New("test", registrationToken,
		upstream.WithRPCEndpoint(apiEndpoint),
		upstream.WithAgentEndpoint(endpoint.NewLocal(apiEndpoint, apiEndpoint)),
	)
	require.NoError(t, err)

	workerTestHelper(t, lis, nil, nil, worker.WithUpstream(upstream))
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

	lis, err := net.Listen("tcp", "127.0.0.1:0")
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

	apiEndpoint := fmt.Sprintf("http://%s", lis.Addr().String())
	upstream, err := upstream.New("test", registrationToken,
		upstream.WithRPCEndpoint(apiEndpoint),
		upstream.WithAgentEndpoint(endpoint.NewLocal(apiEndpoint, apiEndpoint)),
	)
	require.NoError(t, err)

	workerTestHelper(t, lis, isolation, nil, worker.WithUpstream(upstream))
}

func TestWorkerIsolationContainer(t *testing.T) {
	if _, ok := os.LookupEnv("CIRRUS_CONTAINER_BACKEND"); !ok {
		t.Skip("no container backend configured")
	}

	lis, err := heuristic.NewListener(context.Background(), "localhost:0")
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

	var apiEndpoint string

	switch addr := lis.Addr().(type) {
	case *net.TCPAddr:
		apiEndpoint = fmt.Sprintf("http://127.0.0.1:%d", addr.Port)
	case *net.UnixAddr:
		apiEndpoint = fmt.Sprintf("unix:%s", addr.String())
	default:
		t.Fatalf("unknown listener address type: %T", addr)
	}

	upstream, err := upstream.New("test", registrationToken,
		upstream.WithRPCEndpoint(apiEndpoint),
		upstream.WithAgentEndpoint(endpoint.NewLocal(lis.ContainerEndpoint(), lis.ContainerEndpoint())),
	)
	require.NoError(t, err)

	workerTestHelper(t, lis, isolation, nil, worker.WithUpstream(upstream))
}

func TestWorkerIsolationTart(t *testing.T) {
	// Support Tart isolation testing configured via environment variables
	image, vmOk := os.LookupEnv("CIRRUS_INTERNAL_TART_VM")
	user, userOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_USER")
	password, passwordOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_PASSWORD")
	if !vmOk || !userOk || !passwordOk {
		t.Skip("no Tart credentials configured")
	}

	t.Logf("Using Tart VM %s for testing...", image)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	isolation := &api.Isolation{
		Type: &api.Isolation_Tart_{
			Tart: &api.Isolation_Tart{
				Image:    image,
				User:     user,
				Password: password,
				Cpu:      5,
				Memory:   1024 * 5,
			},
		},
	}

	listenerPort := lis.Addr().(*net.TCPAddr).Port
	rpcEndpoint := fmt.Sprintf("http://127.0.0.1:%d", listenerPort)
	upstream, err := upstream.New("test", registrationToken,
		upstream.WithRPCEndpoint(rpcEndpoint),
		upstream.WithAgentEndpoint(endpoint.NewLocal(rpcEndpoint, rpcEndpoint)),
	)
	require.NoError(t, err)

	workerTestHelper(t, lis, isolation, nil, worker.WithUpstream(upstream))
}

func TestWorkerIsolationTartMountTemporaryWorkingDirectoryFromHost(t *testing.T) {
	// Support Tart isolation testing configured via environment variables
	image, vmOk := os.LookupEnv("CIRRUS_INTERNAL_TART_VM")
	user, userOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_USER")
	password, passwordOk := os.LookupEnv("CIRRUS_INTERNAL_TART_SSH_PASSWORD")
	if !vmOk || !userOk || !passwordOk {
		t.Skip("no Tart credentials configured")
	}

	t.Logf("Using Tart VM %s for testing...", image)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	isolation := &api.Isolation{
		Type: &api.Isolation_Tart_{
			Tart: &api.Isolation_Tart{
				Image:                                  image,
				User:                                   user,
				Password:                               password,
				Cpu:                                    5,
				Memory:                                 1024 * 5,
				MountTemporaryWorkingDirectoryFromHost: true,
			},
		},
	}

	listenerPort := lis.Addr().(*net.TCPAddr).Port
	rpcEndpoint := fmt.Sprintf("http://127.0.0.1:%d", listenerPort)
	upstream, err := upstream.New("test", registrationToken,
		upstream.WithRPCEndpoint(rpcEndpoint),
		upstream.WithAgentEndpoint(endpoint.NewLocal(rpcEndpoint, rpcEndpoint)),
	)
	require.NoError(t, err)

	workerTestHelper(t, lis, isolation, []string{
		"pwd",
		"test \"$(pwd)\" = \"/Volumes/My Shared Files/working-dir\"",
	}, worker.WithUpstream(upstream))
}
