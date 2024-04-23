package worker_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"io"
	"runtime"
)

type TasksRPC struct {
	checkScripts      []string
	SucceededCommands []string

	api.UnimplementedCirrusCIServiceServer
}

func NewTasksRPC(checkScripts []string) *TasksRPC {
	if len(checkScripts) == 0 {
		var checkScript string
		if runtime.GOOS == "windows" {
			checkScript = "type go.mod"
		} else {
			checkScript = "test -e go.mod"
		}

		checkScripts = []string{checkScript}
	}

	return &TasksRPC{
		checkScripts: checkScripts,
	}
}

func (tasksRPC *TasksRPC) InitialCommands(
	ctx context.Context,
	request *api.InitialCommandsRequest,
) (*api.CommandsResponse, error) {
	return &api.CommandsResponse{
		Environment: map[string]string{
			"CIRRUS_REPO_CLONE_URL": "http://github.com/cirruslabs/cirrus-cli.git",
			"CIRRUS_BRANCH":         "master",
		},
		Commands: []*api.Command{
			{
				Name: "clone",
				Instruction: &api.Command_CloneInstruction{
					CloneInstruction: &api.CloneInstruction{},
				},
			},
			{
				Name: "check",
				Instruction: &api.Command_ScriptInstruction{
					ScriptInstruction: &api.ScriptInstruction{
						Scripts: tasksRPC.checkScripts,
					},
				},
			},
		},
		ServerToken:      serverSecret,
		TimeoutInSeconds: 3600,
	}, nil
}

func (tasksRPC *TasksRPC) ReportCommandUpdates(
	ctx context.Context,
	request *api.ReportCommandUpdatesRequest,
) (*api.ReportCommandUpdatesResponse, error) {
	return &api.ReportCommandUpdatesResponse{}, nil
}

func (tasksRPC *TasksRPC) ReportAgentFinished(
	ctx context.Context,
	request *api.ReportAgentFinishedRequest,
) (*api.ReportAgentFinishedResponse, error) {
	for _, commandResult := range request.CommandResults {
		if commandResult.Status == api.Status_COMPLETED || commandResult.Status == api.Status_SKIPPED {
			tasksRPC.SucceededCommands = append(tasksRPC.SucceededCommands, commandResult.Name)
		}
	}

	return &api.ReportAgentFinishedResponse{}, nil
}

func (tasksRPC *TasksRPC) StreamLogs(stream api.CirrusCIService_StreamLogsServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return stream.SendAndClose(&api.UploadLogsResponse{})
}

func (tasksRPC *TasksRPC) Heartbeat(
	ctx context.Context,
	request *api.HeartbeatRequest,
) (*api.HeartbeatResponse, error) {
	return &api.HeartbeatResponse{}, nil
}
