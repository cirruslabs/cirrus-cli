package rpc

import (
	"context"
	"strconv"
	"strings"
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

func TestGenerateCacheUploadURLUsesDirectEndpointForLoopbackClient(t *testing.T) {
	task := testTask(t)
	rpcServer, conn := startRPCServerAndConnect(t, task)
	cirrusClient := api.NewCirrusCIServiceClient(conn)

	ctx := authenticatedContext(t, rpcServer, task.LocalGroupId)

	response, err := cirrusClient.GenerateCacheUploadURL(ctx, &api.CacheKey{
		CacheKey: "cache-key",
	})
	require.NoError(t, err)

	expected := strings.Replace(rpcServer.DirectEndpoint(), "http://", "grpc://", 1)
	require.Equal(t, expected, response.Url)
}

func TestGenerateCacheDownloadURLsUsesDirectEndpointForLoopbackClient(t *testing.T) {
	task := testTask(t)
	rpcServer, conn := startRPCServerAndConnect(t, task)
	cirrusClient := api.NewCirrusCIServiceClient(conn)

	ctx := authenticatedContext(t, rpcServer, task.LocalGroupId)

	response, err := cirrusClient.GenerateCacheDownloadURLs(ctx, &api.CacheKey{
		CacheKey: "cache-key",
	})
	require.NoError(t, err)

	expected := strings.Replace(rpcServer.DirectEndpoint(), "http://", "grpc://", 1)
	require.Equal(t, []string{expected}, response.Urls)
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

func authenticatedContext(t *testing.T, rpcServer *RPC, taskID int64) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	return metadata.AppendToOutgoingContext(
		ctx,
		taskIDMetadataKey, strconv.FormatInt(taskID, 10),
		clientSecretMetadataKey, rpcServer.ClientSecret(),
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
