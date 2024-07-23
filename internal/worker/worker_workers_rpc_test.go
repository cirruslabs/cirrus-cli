package worker_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const taskID = 42

type WorkersRPC struct {
	Isolation *api.Isolation

	WorkerWasRegistered bool
	TaskWasAssigned     bool
	TaskWasStarted      bool
	TaskWasStopped      bool
	TaskWasFailed       bool
	TaskFailureMesage   string

	ShouldStopTasks     bool
	NoAutomaticShutdown bool

	api.UnimplementedCirrusWorkersServiceServer
}

func (workersRPC *WorkersRPC) Register(
	ctx context.Context,
	request *api.RegisterRequest,
) (*api.RegisterResponse, error) {
	if request.RegistrationToken == registrationToken {
		workersRPC.WorkerWasRegistered = true

		return &api.RegisterResponse{SessionToken: sessionToken}, nil
	}

	return nil, status.Errorf(codes.PermissionDenied, "invalid registration token")
}

func (workersRPC *WorkersRPC) Poll(ctx context.Context, request *api.PollRequest) (*api.PollResponse, error) {
	if !workersRPC.TaskWasAssigned {
		workersRPC.TaskWasAssigned = true

		return &api.PollResponse{
			TasksToStart: []*api.PollResponse_AgentAwareTask{
				{
					TaskIdOld:    taskID,
					ClientSecret: clientSecret,
					ServerSecret: serverSecret,
					Isolation:    workersRPC.Isolation,
				},
			},
		}, nil
	}

	if workersRPC.TaskWasStopped && !workersRPC.NoAutomaticShutdown {
		return &api.PollResponse{
			Shutdown: true,
		}, nil
	}

	if workersRPC.ShouldStopTasks {
		pollResponse := &api.PollResponse{
			TasksToStopOld: []int64{taskID},
		}

		workersRPC.ShouldStopTasks = false

		return pollResponse, nil
	}

	return &api.PollResponse{}, nil
}

func (workersRPC *WorkersRPC) TaskStarted(
	ctx context.Context,
	request *api.WorkerTaskIdentification,
) (*empty.Empty, error) {
	if request.TaskIdOld == taskID {
		workersRPC.TaskWasStarted = true
	}

	return &empty.Empty{}, nil
}

func (workersRPC *WorkersRPC) TaskStopped(
	ctx context.Context,
	request *api.WorkerTaskIdentification,
) (*empty.Empty, error) {
	if request.TaskIdOld == taskID {
		workersRPC.TaskWasStopped = true
	}

	return &empty.Empty{}, nil
}

func (workersRPC *WorkersRPC) TaskFailed(ctx context.Context, request *api.TaskFailedRequest) (*empty.Empty, error) {
	if request.TaskIdentification.TaskIdOld == taskID {
		workersRPC.TaskWasFailed = true
		workersRPC.TaskFailureMesage = request.Message
	}

	return &empty.Empty{}, nil
}
