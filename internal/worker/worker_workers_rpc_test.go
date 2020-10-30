package worker_test

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const taskID = 42

type WorkersRPC struct {
	WorkerWasRegistered bool
	TaskWasAssigned     bool
	TaskWasStarted      bool
	TaskWasStopped      bool

	api.UnimplementedCirrusWorkersServiceServer
}

func (workersRPC *WorkersRPC) Register(
	ctx context.Context,
	request *api.RegisterRequest,
) (*api.RegisterResponse, error) {
	if request.RegistrationToken == registrationToken {
		workersRPC.WorkerWasRegistered = true

		return &api.RegisterResponse{AuthenticationToken: authenticationToken}, nil
	}

	return nil, status.Errorf(codes.PermissionDenied, "invalid registration token")
}

func (workersRPC *WorkersRPC) Poll(ctx context.Context, request *api.PollRequest) (*api.PollResponse, error) {
	var tasks []*api.PollResponse_AgentAwareTask

	if !workersRPC.TaskWasAssigned {
		workersRPC.TaskWasAssigned = true

		tasks = append(tasks, &api.PollResponse_AgentAwareTask{
			TaskId:       taskID,
			ClientSecret: clientSecret,
			ServerSecret: serverSecret,
		})
	}

	return &api.PollResponse{
		TasksToStart: tasks,
	}, nil
}

func (workersRPC *WorkersRPC) TaskStarted(ctx context.Context, request *api.TaskIdentification) (*empty.Empty, error) {
	if request.TaskId == taskID {
		workersRPC.TaskWasStarted = true
	}

	return &empty.Empty{}, nil
}

func (workersRPC *WorkersRPC) TaskStopped(ctx context.Context, request *api.TaskIdentification) (*empty.Empty, error) {
	if request.TaskId == taskID {
		workersRPC.TaskWasStopped = true
	}

	return &empty.Empty{}, nil
}
