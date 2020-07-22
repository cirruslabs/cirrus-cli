package rpc

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build/taskstatus"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"io/ioutil"
	"net"
	"sync"

	// Registers a gzip compressor needed for streaming logs from the agent
	_ "google.golang.org/grpc/encoding/gzip"
)

type RPC struct {
	listener                   net.Listener
	server                     *grpc.Server
	serverWaitGroup            sync.WaitGroup
	serverSecret, clientSecret string

	build *build.Build

	logger *logrus.Logger

	api.UnimplementedCirrusCIServiceServer
}

func New(build *build.Build, opts ...Option) *RPC {
	r := &RPC{
		server:       grpc.NewServer(),
		serverSecret: uuid.New().String(),
		clientSecret: uuid.New().String(),
		build:        build,
	}

	// Register itself
	api.RegisterCirrusCIServiceServer(r.server, r)

	// Apply options
	for _, opt := range opts {
		opt(r)
	}

	// Apply default options (to cover those that weren't specified)
	if r.logger == nil {
		r.logger = logrus.New()
		r.logger.Out = ioutil.Discard
	}

	return r
}

func (r *RPC) ServerSecret() string {
	return r.serverSecret
}

func (r *RPC) ClientSecret() string {
	return r.clientSecret
}

// Start creates the listener and starts RPC server in a separate goroutine.
func (r *RPC) Start() {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	r.listener = listener

	r.serverWaitGroup.Add(1)
	go func() {
		if err := r.server.Serve(listener); err != nil {
			r.logger.WithError(err).Error("RPC server failed")
		}
		r.serverWaitGroup.Done()
	}()

	r.logger.Debugf("gRPC server is listening at %s", r.Endpoint())
}

// Endpoint returns RPC server address suitable for use in agent's "-api-endpoint" flag.
func (r *RPC) Endpoint() string {
	port := r.listener.Addr().(*net.TCPAddr).Port

	return fmt.Sprintf("host.docker.internal:%d", port)
}

// Stop gracefully stops the RPC server.
func (r *RPC) Stop() {
	r.server.GracefulStop()
	r.serverWaitGroup.Wait()
}

func (r *RPC) InitialCommands(
	ctx context.Context,
	req *api.InitialCommandsRequest,
) (*api.CommandsResponse, error) {
	r.build.Mutex.Lock()
	defer r.build.Mutex.Unlock()

	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	return &api.CommandsResponse{
		Environment:      r.build.Environment,
		Commands:         task.ProtoTask.Commands,
		ServerToken:      r.serverSecret,
		TimeoutInSeconds: int64(task.Timeout.Seconds()),
	}, nil
}

func getCommandAfter(commands []*api.Command, currentCommand string) string {
	var doReturn bool

	for _, command := range commands {
		if doReturn {
			return command.Name
		}

		if command.Name == currentCommand {
			doReturn = true
		}
	}

	return ""
}

func (r *RPC) ReportSingleCommand(
	ctx context.Context,
	req *api.ReportSingleCommandRequest,
) (*api.ReportSingleCommandResponse, error) {
	r.build.Mutex.Lock()
	defer r.build.Mutex.Unlock()

	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	var nextCommand string

	// Register whether the current command succeeded or failed
	// so that the main loop can make the decision whether
	// to proceed with the execution or not.
	if req.Succeded {
		nextCommand = getCommandAfter(task.ProtoTask.Commands, req.CommandName)
		if nextCommand == "" {
			task.Status = taskstatus.Succeeded
		}
	} else {
		// An empty string instructs the agent to do nothing and terminate
		nextCommand = ""
		task.Status = taskstatus.Failed
	}

	return &api.ReportSingleCommandResponse{
		NextCommandName: nextCommand,
	}, nil
}

func (r *RPC) StreamLogs(stream api.CirrusCIService_StreamLogsServer) error {
	var currentTaskName string
	var currentCommand string

	for {
		logEntry, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			r.logger.WithContext(stream.Context()).Warn(err)
			return err
		}

		switch x := logEntry.Value.(type) {
		case *api.LogEntry_Key:
			r.build.Mutex.Lock()

			task, err := r.build.GetTaskFromIdentification(x.Key.TaskIdentification, r.clientSecret)
			if err != nil {
				return err
			}
			currentTaskName = task.ProtoTask.Name
			currentCommand = x.Key.CommandName

			r.build.Mutex.Unlock()

			r.logger.WithFields(map[string]interface{}{
				"task":    currentTaskName,
				"command": currentCommand,
			}).Debug("begin streaming logs")
		case *api.LogEntry_Chunk:
			if currentTaskName == "" {
				return status.Error(codes.PermissionDenied, "not authenticated")
			}

			r.logger.WithFields(map[string]interface{}{
				"task":    currentTaskName,
				"command": currentCommand,
			}).Debugf("received chunk: %s", string(x.Chunk.Data))
		}
	}

	if err := stream.SendAndClose(&api.UploadLogsResponse{}); err != nil {
		r.logger.WithContext(stream.Context()).WithError(err).Warn("while closing log stream")
		return err
	}

	return nil
}

func (r *RPC) Heartbeat(ctx context.Context, req *api.HeartbeatRequest) (*api.HeartbeatResponse, error) {
	r.build.Mutex.Lock()
	defer r.build.Mutex.Unlock()

	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.WithField("task", task.ProtoTask.Name).Debug("received heartbeat")

	return &api.HeartbeatResponse{}, nil
}

func (r *RPC) ReportAgentError(ctx context.Context, req *api.ReportAgentProblemRequest) (*empty.Empty, error) {
	r.build.Mutex.Lock()
	defer r.build.Mutex.Unlock()

	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.WithField("task", task.ProtoTask.Name).Debugf("agent error: %s", req.Message)

	return &empty.Empty{}, nil
}

func (r *RPC) ReportAgentWarning(ctx context.Context, req *api.ReportAgentProblemRequest) (*empty.Empty, error) {
	r.build.Mutex.Lock()
	defer r.build.Mutex.Unlock()

	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.WithField("task", task.ProtoTask.Name).Debugf("agent warning: %s", req.Message)

	return &empty.Empty{}, nil
}

func (r *RPC) ReportAgentSignal(ctx context.Context, req *api.ReportAgentSignalRequest) (*empty.Empty, error) {
	r.build.Mutex.Lock()
	defer r.build.Mutex.Unlock()

	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.WithField("task", task.ProtoTask.Name).Debugf("agent signal: %s", req.Signal)

	return &empty.Empty{}, nil
}
