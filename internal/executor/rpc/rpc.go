package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/commands/logs"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build"
	"github.com/cirruslabs/cirrus-cli/internal/executor/build/commandstatus"
	"github.com/cirruslabs/cirrus-cli/internal/executor/heuristic"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"strings"
	"sync"

	// Registers a gzip compressor needed for streaming logs from the agent.
	_ "google.golang.org/grpc/encoding/gzip"
)

var ErrRPCFailed = errors.New("RPC server failed")

type RPC struct {
	// must be embedded to have forward compatible implementations
	api.UnimplementedCirrusCIServiceServer

	listener                   *heuristic.Listener
	server                     *grpc.Server
	serverWaitGroup            sync.WaitGroup
	serverSecret, clientSecret string

	build *build.Build

	logger       *echelon.Logger
	artifactsDir string
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
		renderer := renderers.NewSimpleRenderer(io.Discard, nil)
		r.logger = echelon.NewLogger(echelon.InfoLevel, renderer)
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
func (r *RPC) Start(ctx context.Context, address string) error {
	listener, err := heuristic.NewListener(ctx, address)
	if err != nil {
		return fmt.Errorf("%w: failed to start RPC service on %s: %v", ErrRPCFailed, address, err)
	}
	r.listener = listener

	r.serverWaitGroup.Add(1)
	go func() {
		if err := r.server.Serve(listener); err != nil {
			if !errors.Is(err, grpc.ErrServerStopped) {
				r.logger.Errorf("RPC server failed: %v", err)
			}
		}
		r.serverWaitGroup.Done()
	}()

	r.logger.Debugf("gRPC server is listening at %s (%s inside of a container)",
		r.DirectEndpoint(), r.ContainerEndpoint())

	return nil
}

// ContainerEndpoint returns RPC server address suitable for use in agent's "-api-endpoint" flag
// when running inside of a container.
func (r *RPC) ContainerEndpoint() string {
	return r.listener.ContainerEndpoint()
}

// DirectEndpoint returns RPC server address suitable for use in agent's "-api-endpoint" flag
// when running on the host.
func (r *RPC) DirectEndpoint() string {
	return r.listener.DirectEndpoint()
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
		Environment:       task.Environment,
		Commands:          task.ProtoCommands(),
		ServerToken:       r.serverSecret,
		TimeoutInSeconds:  int64(task.Timeout.Seconds()),
		FailedAtLeastOnce: task.FailedAtLeastOnce(),
	}, nil
}

func (r *RPC) ReportCommandUpdates(
	ctx context.Context,
	req *api.ReportCommandUpdatesRequest,
) (*api.ReportCommandUpdatesResponse, error) {
	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	for _, update := range req.Updates {
		command := task.GetCommand(update.Name)
		if command == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "attempt to set status for non-existent command %s",
				update.Name)
		}

		commandLogger := r.getCommandLogger(task, command)

		// Register whether the current command succeeded or failed
		// so that the main loop can make the decision whether
		// to proceed with the execution or not.
		switch {
		case update.Status == api.Status_COMPLETED:
			command.SetStatus(commandstatus.Success)
			commandLogger.Debugf("command %s succeeded", update.Name)
			commandLogger.FinishWithType(echelon.FinishTypeSucceeded)
		case update.Status == api.Status_SKIPPED:
			command.SetStatus(commandstatus.Success)
			commandLogger.Debugf("command %s was skipped", update.Name)
			commandLogger.FinishWithType(echelon.FinishTypeSkipped)
		case update.Status == api.Status_ABORTED || update.Status == api.Status_FAILED:
			command.SetStatus(commandstatus.Failure)
			commandLogger.Debugf("command %s failed", update.Name)
			commandLogger.FinishWithType(echelon.FinishTypeFailed)
		}
	}

	return &api.ReportCommandUpdatesResponse{}, nil
}

func (r *RPC) ReportAgentFinished(
	ctx context.Context,
	req *api.ReportAgentFinishedRequest,
) (*api.ReportAgentFinishedResponse, error) {
	return &api.ReportAgentFinishedResponse{}, nil
}

func (r *RPC) getCommandLogger(task *build.Task, command *build.Command) *echelon.Logger {
	commandLoggerScope := fmt.Sprintf("'%s'", command.ProtoCommand.Name)
	command.ProtoCommand.GetScriptInstruction()
	switch command.ProtoCommand.Instruction.(type) {
	case *api.Command_ScriptInstruction:
		commandLoggerScope += " script"
	case *api.Command_BackgroundScriptInstruction:
		commandLoggerScope += " background script"
	case *api.Command_CacheInstruction:
		commandLoggerScope += " cache"
	case *api.Command_ArtifactsInstruction:
		commandLoggerScope += " artifacts"
	}
	return r.logger.Scoped(task.UniqueDescription()).Scoped(commandLoggerScope)
}

func (r *RPC) StreamLogs(stream api.CirrusCIService_StreamLogsServer) error {
	var currentTaskName string
	var currentCommand string
	streamLogger := r.logger

	for {
		logEntry, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			streamLogger.Warnf("Error while receiving logs: %v", err)
			return err
		}

		switch x := logEntry.Value.(type) {
		case *api.LogEntry_Key:
			task, err := r.build.GetTaskFromIdentification(x.Key.TaskIdentification, r.clientSecret)
			if err != nil {
				return err
			}
			currentTaskName = task.Name
			currentCommand = x.Key.CommandName

			command := task.GetCommand(currentCommand)
			if command == nil {
				return status.Errorf(codes.FailedPrecondition, "attempt to stream logs for non-existent command %s",
					currentCommand)
			}

			streamLogger = r.getCommandLogger(task, command)
			streamLogger.Debugf("begin streaming logs")
		case *api.LogEntry_Chunk:
			if currentTaskName == "" {
				return status.Error(codes.PermissionDenied, "not authenticated")
			}

			streamLogger.Debugf("received log chunk of %d bytes", len(x.Chunk.Data))

			// Ignore the newline at the end of the chunk,
			// echelon's Infof() below already adds one
			log := strings.TrimSuffix(string(x.Chunk.Data), "\n")

			logLines := strings.Split(log, "\n")

			for _, logLine := range logLines {
				streamLogger.Infof(logLine)
			}
		}
	}

	if err := stream.SendAndClose(&api.UploadLogsResponse{}); err != nil {
		streamLogger.Warnf("Error while closing log stream: %v", err)
		return err
	}

	return nil
}

func (r *RPC) Heartbeat(ctx context.Context, req *api.HeartbeatRequest) (*api.HeartbeatResponse, error) {
	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.Scoped(task.UniqueDescription()).Debugf("received heartbeat")

	return &api.HeartbeatResponse{}, nil
}

func (r *RPC) ReportAgentError(ctx context.Context, req *api.ReportAgentProblemRequest) (*empty.Empty, error) {
	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.Scoped(task.UniqueDescription()).Errorf("agent error: %s", req.Message)

	return &empty.Empty{}, nil
}

func (r *RPC) ReportAgentWarning(ctx context.Context, req *api.ReportAgentProblemRequest) (*empty.Empty, error) {
	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.Scoped(task.UniqueDescription()).Warnf("agent warning: %s", req.Message)

	return &empty.Empty{}, nil
}

func (r *RPC) ReportAgentSignal(ctx context.Context, req *api.ReportAgentSignalRequest) (*empty.Empty, error) {
	task, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	r.logger.Scoped(task.UniqueDescription()).Debugf("agent signal: %s", req.Signal)

	return &empty.Empty{}, nil
}

func (r *RPC) ReportAnnotations(ctx context.Context, req *api.ReportAnnotationsCommandRequest) (*empty.Empty, error) {
	_, err := r.build.GetTaskFromIdentification(req.TaskIdentification, r.clientSecret)
	if err != nil {
		return nil, err
	}

	ghaRenderer, ok := r.logger.Renderer().(*logs.GithubActionsLogsRenderer)
	if !ok {
		return &empty.Empty{}, nil
	}

	for _, annotation := range req.Annotations {
		if annotation.FileLocation == nil {
			continue
		}

		var mappedLevel string

		switch annotation.Level {
		case api.Annotation_NOTICE:
			mappedLevel = "notice"
		case api.Annotation_WARNING:
			mappedLevel = "warning"
		case api.Annotation_FAILURE:
			mappedLevel = "error"
		}

		rawMessage := fmt.Sprintf("::%s file=%s,line=%d,endLine=%d,title=%s::%s\n", mappedLevel,
			annotation.FileLocation.Path, annotation.FileLocation.StartLine, annotation.FileLocation.EndLine,
			annotation.Message, annotation.RawDetails)
		ghaRenderer.RenderRawMessage(rawMessage)
	}

	return &empty.Empty{}, nil
}
