package rpc

import (
	"context"
	"strconv"

	"github.com/cirruslabs/cirrus-cli/internal/executor/build"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	taskIDMetadataKey       = "org.cirruslabs.task-id"
	clientSecretMetadataKey = "org.cirruslabs.client-secret"
)

func (r *RPC) taskFromMetadata(ctx context.Context) (*build.Task, error) {
	taskID, clientSecret, err := extractTaskAuthMetadata(ctx)
	if err != nil {
		return nil, err
	}

	if clientSecret != r.clientSecret {
		return nil, status.Error(codes.Unauthenticated, "provided secret value is invalid")
	}

	task := r.build.GetTask(taskID)
	if task == nil {
		return nil, status.Errorf(codes.NotFound, "cannot find the task with the specified ID")
	}

	return task, nil
}

func extractTaskAuthMetadata(ctx context.Context) (int64, string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, "", status.Error(codes.Unauthenticated, "request metadata is missing")
	}

	taskIDValue := md.Get(taskIDMetadataKey)
	if len(taskIDValue) == 0 {
		return 0, "", status.Error(codes.Unauthenticated, "task metadata is missing")
	}
	taskID, err := strconv.ParseInt(taskIDValue[0], 10, 64)
	if err != nil {
		return 0, "", status.Error(codes.InvalidArgument, "task metadata is invalid")
	}

	clientSecretValue := md.Get(clientSecretMetadataKey)
	if len(clientSecretValue) == 0 {
		return 0, "", status.Error(codes.Unauthenticated, "client secret metadata is missing")
	}

	return taskID, clientSecretValue[0], nil
}
