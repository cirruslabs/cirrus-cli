// +build !windows

package worker_test

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/worker"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
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

func TestWorker(t *testing.T) {
	// Start the RPC server
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(unaryInterceptor))

	workersRPC := &WorkersRPC{}
	api.RegisterCirrusWorkersServiceServer(server, workersRPC)
	tasksRPC := &TasksRPC{}
	api.RegisterCirrusCIServiceServer(server, tasksRPC)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Error(err)
		}
	}()

	// Start the worker
	worker, err := worker.New(
		worker.WithRegistrationToken(registrationToken),
		worker.WithRPCEndpoint("http://"+lis.Addr().String()),
	)
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
