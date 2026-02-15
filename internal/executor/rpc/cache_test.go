package rpc

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/cirruslabs/cirrus-cli/internal/executor/build"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/cirrus-cli/pkg/grpchelper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestGenerateCacheUploadURLUsesReportedAPIEndpoint(t *testing.T) {
	task := testTask(t)
	rpcServer, conn := startRPCServerAndConnect(t, task)
	cirrusClient := api.NewCirrusCIServiceClient(conn)

	reportedAPIEndpoint := "http://127.0.0.1:31337"
	ctx := authenticatedContext(t, rpcServer, task.LocalGroupId, reportedAPIEndpoint)

	response, err := cirrusClient.GenerateCacheUploadURL(ctx, &api.CacheKey{
		CacheKey: "cache-key",
	})
	require.NoError(t, err)

	require.Equal(t, asGRPCEndpoint(reportedAPIEndpoint), response.Url)
}

func TestGenerateCacheDownloadURLsUsesReportedAPIEndpoint(t *testing.T) {
	task := testTask(t)
	rpcServer, conn := startRPCServerAndConnect(t, task)
	cirrusClient := api.NewCirrusCIServiceClient(conn)

	reportedAPIEndpoint := "http://127.0.0.1:31337"
	ctx := authenticatedContext(t, rpcServer, task.LocalGroupId, reportedAPIEndpoint)

	response, err := cirrusClient.GenerateCacheDownloadURLs(ctx, &api.CacheKey{
		CacheKey: "cache-key",
	})
	require.NoError(t, err)

	require.Equal(t, []string{asGRPCEndpoint(reportedAPIEndpoint)}, response.Urls)
}

func TestGenerateCacheUploadURLFallsBackToContainerEndpoint(t *testing.T) {
	task := testTask(t)
	rpcServer, conn := startRPCServerAndConnect(t, task)
	cirrusClient := api.NewCirrusCIServiceClient(conn)

	ctx := authenticatedContext(t, rpcServer, task.LocalGroupId, "")

	response, err := cirrusClient.GenerateCacheUploadURL(ctx, &api.CacheKey{
		CacheKey: "cache-key",
	})
	require.NoError(t, err)

	require.Equal(t, asGRPCEndpoint(rpcServer.ContainerEndpoint()), response.Url)
}

func startRPCServerAndConnect(t *testing.T, task *api.Task) (*RPC, *grpc.ClientConn) {
	t.Helper()

	b, err := build.New(t.TempDir(), []*api.Task{task}, &logger.LightweightStub{})
	require.NoError(t, err)

	rpcServer := New(b)
	require.NoError(t, rpcServer.Start(context.Background(), "localhost:0", true))
	t.Cleanup(rpcServer.Stop)

	target, dialOption := grpchelper.TransportSettingsAsDialOption(rpcServer.DirectEndpoint())
	conn, err := grpc.NewClient(target, dialOption)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	return rpcServer, conn
}

func authenticatedContext(t *testing.T, rpcServer *RPC, taskID int64, apiEndpoint string) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	metadataPairs := []string{
		taskIDMetadataKey, strconv.FormatInt(taskID, 10),
		clientSecretMetadataKey, rpcServer.ClientSecret(),
	}

	if apiEndpoint != "" {
		metadataPairs = append(metadataPairs, apiEndpointMetadataKey, apiEndpoint)
	}

	return metadata.AppendToOutgoingContext(
		ctx,
		metadataPairs...,
	)
}

func testTask(t *testing.T) *api.Task {
	t.Helper()

	anyInstance, err := anypb.New(&api.ContainerInstance{
		Image: "debian:latest",
	})
	require.NoError(t, err)

	return &api.Task{
		LocalGroupId: 1,
		Name:         "task",
		Status:       api.Status_CREATED,
		Instance:     anyInstance,
		Commands: []*api.Command{
			{
				Name: "script",
				Instruction: &api.Command_ScriptInstruction{
					ScriptInstruction: &api.ScriptInstruction{
						Scripts: []string{"true"},
					},
				},
			},
		},
	}
}
