package worker_test

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/worker"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

const (
	registrationToken   = "registration token"
	authenticationToken = "session token"
	serverSecret        = "server secret"
	clientSecret        = "client secret"
)

func TestWorker(t *testing.T) {
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	server := grpc.NewServer()

	workersRPC := &WorkersRPC{}
	api.RegisterCirrusWorkersServiceServer(server, workersRPC)
	tasksRPC := &TasksRPC{}
	api.RegisterCirrusCIServiceServer(server, tasksRPC)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Error(err)
		}
	}()

	worker, err := worker.New(
		worker.WithRegistrationToken(registrationToken),
		worker.WithRPCEndpoint(lis.Addr().String()),
		worker.WithRPCInsecure(),
	)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	if err := worker.Run(ctx); err != nil {
		t.Fatal(err)
	}

	cancel()
	server.GracefulStop()

	assert.True(t, workersRPC.WorkerWasRegistered)
	assert.True(t, workersRPC.TaskWasAssigned)
	assert.True(t, workersRPC.TaskWasStarted)
	assert.True(t, workersRPC.TaskWasStopped)

	assert.Equal(t, []string{"clone", "check"}, tasksRPC.SucceededCommands)
}
