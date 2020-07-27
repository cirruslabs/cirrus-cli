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
	"runtime"
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

func getDockerBridgeIP() string {
	// Worst-case scenario, but still better than nothing
	// since there's still a chance this would work with
	// a Docker daemon configured by default.
	const assumedBridgeIP = "172.17.0.1"

	iface, err := net.InterfaceByName("bridge0")
	if err != nil {
		return assumedBridgeIP
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return assumedBridgeIP
	}

	if len(addrs) > 1 {
		ip, _, err := net.ParseCIDR(addrs[0].String())
		if err != nil {
			return assumedBridgeIP
		}

		return ip.String()
	}

	return assumedBridgeIP
}

// Start creates the listener and starts RPC server in a separate goroutine.
func (r *RPC) Start() {
	host := "localhost"

	// Work around host.docker.internal missing on Linux
	//
	// See the following tickets:
	// * https://github.com/docker/for-linux/issues/264
	// * https://github.com/moby/moby/pull/40007
	if runtime.GOOS == "linux" {
		host = getDockerBridgeIP()
	}

	address := fmt.Sprintf("%s:0", host)

	listener, err := net.Listen("tcp", address)
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
	// Work around host.docker.internal missing on Linux
	if runtime.GOOS == "linux" {
		return r.listener.Addr().String()
	}

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
	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	var nextCommand string

	logEntry := r.logger.WithFields(map[string]interface{}{
		"task":    task.ProtoTask.Name,
		"command": req.CommandName,
	})

	// Register whether the current command succeeded or failed
	// so that the main loop can make the decision whether
	// to proceed with the execution or not.
	if req.Succeded {
		logEntry.Debug("command succeeded")

		nextCommand = getCommandAfter(task.ProtoTask.Commands, req.CommandName)
		if nextCommand == "" {
			task.SetStatus(taskstatus.Succeeded)
		}
	} else {
		logEntry.Debug("command failed")

		// An empty string instructs the agent to do nothing and terminate
		nextCommand = ""
		task.SetStatus(taskstatus.Failed)
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
			task, err := r.build.GetTaskFromIdentification(x.Key.TaskIdentification, r.clientSecret)
			if err != nil {
				return err
			}
			currentTaskName = task.ProtoTask.Name
			currentCommand = x.Key.CommandName

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
	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.WithField("task", task.ProtoTask.Name).Debug("received heartbeat")

	return &api.HeartbeatResponse{}, nil
}

func (r *RPC) ReportAgentError(ctx context.Context, req *api.ReportAgentProblemRequest) (*empty.Empty, error) {
	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.WithField("task", task.ProtoTask.Name).Debugf("agent error: %s", req.Message)

	return &empty.Empty{}, nil
}

func (r *RPC) ReportAgentWarning(ctx context.Context, req *api.ReportAgentProblemRequest) (*empty.Empty, error) {
	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.WithField("task", task.ProtoTask.Name).Debugf("agent warning: %s", req.Message)

	return &empty.Empty{}, nil
}

func (r *RPC) ReportAgentSignal(ctx context.Context, req *api.ReportAgentSignalRequest) (*empty.Empty, error) {
	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.WithField("task", task.ProtoTask.Name).Debugf("agent signal: %s", req.Signal)

	return &empty.Empty{}, nil
}
