package testutil

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"net"
	"sync/atomic"
	"testing"
)

type CirrusCIServiceMock struct {
	// Initialization state
	lis              net.Listener
	commandsResponse *api.CommandsResponse

	// Running state
	finished  atomic.Bool
	succeeded atomic.Bool

	api.UnimplementedCirrusCIServiceServer
}

func NewCirrusCIServiceMock(t *testing.T, commandsResponse *api.CommandsResponse) *CirrusCIServiceMock {
	t.Helper()

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	m := &CirrusCIServiceMock{
		lis:              lis,
		commandsResponse: commandsResponse,
	}

	server := grpc.NewServer()
	api.RegisterCirrusCIServiceServer(server, m)
	go func() {
		if err := server.Serve(lis); err != nil {
			panic(err)
		}
	}()

	return m
}

func (m *CirrusCIServiceMock) Address() string {
	return m.lis.Addr().String()
}

func (m *CirrusCIServiceMock) Finished() bool {
	return m.finished.Load()
}

func (m *CirrusCIServiceMock) Succeeded() bool {
	return m.succeeded.Load()
}

func (m *CirrusCIServiceMock) InitialCommands(ctx context.Context, request *api.InitialCommandsRequest) (*api.CommandsResponse, error) {
	return m.commandsResponse, nil
}

func (m *CirrusCIServiceMock) ReportCommandUpdates(ctx context.Context, request *api.ReportCommandUpdatesRequest) (*api.ReportCommandUpdatesResponse, error) {
	return &api.ReportCommandUpdatesResponse{}, nil
}

func (m *CirrusCIServiceMock) StreamLogs(server api.CirrusCIService_StreamLogsServer) error {
	for {
		_, err := server.Recv()
		if err != nil {
			return err
		}
	}
}

func (m *CirrusCIServiceMock) SaveLogs(server api.CirrusCIService_SaveLogsServer) error {
	for {
		_, err := server.Recv()
		if err != nil {
			return err
		}
	}

}

func (m *CirrusCIServiceMock) ReportAgentFinished(ctx context.Context, request *api.ReportAgentFinishedRequest) (*api.ReportAgentFinishedResponse, error) {
	m.finished.Store(true)

	commandNameToStatus := lo.Associate(request.CommandResults, func(commandResult *api.CommandResult) (string, api.Status) {
		return commandResult.Name, commandResult.Status
	})
	succeeded := lo.EveryBy(lo.Values(commandNameToStatus), func(status api.Status) bool {
		return status == api.Status_COMPLETED || status == api.Status_SKIPPED
	})
	m.succeeded.Store(succeeded)

	return &api.ReportAgentFinishedResponse{}, nil
}
