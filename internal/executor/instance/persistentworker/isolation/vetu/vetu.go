package vetu

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/projectdirsyncer"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/remoteagent"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/ssh"
	"runtime"
	"strings"
	"time"
)

var (
	ErrFailed     = errors.New("vetu isolation failed")
	ErrSyncFailed = errors.New("failed to sync project directory")

	tracer = otel.Tracer("vetu")
)

const vmNamePrefix = "cirrus-cli-"

type Vetu struct {
	logger           logger.Lightweight
	vmName           string
	sshUser          string
	sshPassword      string
	sshPort          uint16
	cpu              uint32
	memory           uint32
	diskSize         uint32
	bridgedInterface string
	hostNetworking   bool
}

func New(
	vmName string,
	sshUser string,
	sshPassword string,
	sshPort uint16,
	cpu uint32,
	memory uint32,
	opts ...Option,
) (*Vetu, error) {
	vetu := &Vetu{
		vmName:      vmName,
		sshUser:     sshUser,
		sshPassword: sshPassword,
		sshPort:     sshPort,
		cpu:         cpu,
		memory:      memory,
	}

	// Apply options
	for _, opt := range opts {
		opt(vetu)
	}

	// Apply default options (to cover those that weren't specified)
	if vetu.sshPort == 0 {
		vetu.sshPort = 22
	}

	if vetu.logger == nil {
		vetu.logger = &logger.LightweightStub{}
	}

	return vetu, nil
}

func (vetu *Vetu) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("image", vetu.Image()),
		attribute.String("instance_type", "vetu"),
	}
}

func (vetu *Vetu) Run(ctx context.Context, config *runconfig.RunConfig) error {
	ctx, prepareInstanceSpan := tracer.Start(ctx, "prepare-instance")
	defer prepareInstanceSpan.End()

	tmpVMName := fmt.Sprintf("%s%d-", vmNamePrefix, config.TaskID) + uuid.NewString()
	vm, err := NewVMClonedFrom(ctx,
		vetu.vmName, tmpVMName,
		config.VetuOptions.LazyPull,
		config.AdditionalEnvironment,
		config.Logger(),
	)
	if err != nil {
		return fmt.Errorf("%w: failed to create VM cloned from %q: %v", ErrFailed, vetu.vmName, err)
	}
	defer func() {
		if localHub := sentry.GetHubFromContext(ctx); localHub != nil {
			localHub.AddBreadcrumb(&sentry.Breadcrumb{
				Message: fmt.Sprintf("stopping and deleting the VM %s", vm.ident),
			}, nil)
		}

		_ = vm.Close()
	}()

	if err := vm.Configure(ctx, vetu.cpu, vetu.memory, vetu.diskSize, config.Logger()); err != nil {
		return fmt.Errorf("%w: failed to configure VM %q: %v", ErrFailed, vm.Ident(), err)
	}

	// Start the VM (asynchronously)
	vm.Start(ctx, vetu.bridgedInterface, vetu.hostNetworking)

	// Wait for the VM to start and get its IP address
	bootLogger := config.Logger().Scoped("boot virtual machine")

	var ip string

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-vm.ErrChan():
			return err
		default:
			time.Sleep(time.Second)
		}

		ip, err = vm.RetrieveIP(ctx)
		if err != nil {
			vetu.logger.Debugf("failed to retrieve VM %s IP: %v\n", vm.Ident(), err)
			continue
		}

		break
	}

	vetu.logger.Debugf("IP %s retrieved from VM %s, running agent...", ip, vm.Ident())

	bootLogger.Errorf("VM was assigned with %s IP", ip)
	bootLogger.Finish(true)

	prepareInstanceSpan.End()

	err = remoteagent.WaitForAgent(ctx, vetu.logger, fmt.Sprintf("%s:%d", ip, vetu.sshPort),
		vetu.sshUser, vetu.sshPassword, "linux", runtime.GOARCH,
		config, true, vetu.initializeHooks(config), nil, "",
		map[string]string{"CIRRUS_VM_ID": vm.Ident()})
	if err != nil {
		return err
	}

	return nil
}

func (vetu *Vetu) Image() string {
	return vetu.vmName
}

func (vetu *Vetu) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return platform.NewUnix().GenericWorkingDir()
}

func (vetu *Vetu) Close(context.Context) error {
	return nil
}

func Cleanup() error {
	stdout, _, err := Cmd(context.Background(), nil, "list", "--quiet")
	if err != nil {
		return err
	}

	for _, vmName := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if !strings.HasPrefix(vmName, vmNamePrefix) {
			continue
		}

		if _, _, err := Cmd(context.Background(), nil, "delete", vmName); err != nil {
			return err
		}
	}

	return nil
}

func (vetu *Vetu) initializeHooks(config *runconfig.RunConfig) []remoteagent.WaitForAgentHook {
	var hooks []remoteagent.WaitForAgentHook

	if config.ProjectDir != "" {
		hooks = append(hooks, func(ctx context.Context, sshClient *ssh.Client) error {
			syncLogger := config.Logger().Scoped("syncing working directory")

			if err := projectdirsyncer.SyncProjectDir(config.ProjectDir, sshClient); err != nil {
				syncLogger.Finish(false)

				return fmt.Errorf("%w: %v", ErrSyncFailed, err)
			}

			syncLogger.Finish(true)

			return nil
		})
	}

	return hooks
}
