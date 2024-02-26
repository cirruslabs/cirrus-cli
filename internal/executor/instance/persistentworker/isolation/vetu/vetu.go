package vetu

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/projectdirsyncer"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/remoteagent"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"github.com/cirruslabs/echelon"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"golang.org/x/crypto/ssh"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

var (
	ErrFailed     = errors.New("vetu isolation failed")
	ErrSyncFailed = errors.New("failed to sync project directory")

	tracer = otel.Tracer("vetu")
)

const vmNamePrefix = "cirrus-cli-"

type Vetu struct {
	cloneAndConfigureResult atomic.Pointer[abstract.CloneAndConfigureResult]
	cleanupFuncs            []func()

	logger           logger.Lightweight
	vmName           string
	sshUser          string
	sshPassword      string
	cpu              uint32
	memory           uint32
	diskSize         uint32
	bridgedInterface string
	hostNetworking   bool
	isolation        *api.Isolation
}

func New(
	vmName string,
	sshUser string,
	sshPassword string,
	cpu uint32,
	memory uint32,
	opts ...Option,
) (*Vetu, error) {
	vetu := &Vetu{
		vmName:      vmName,
		sshUser:     sshUser,
		sshPassword: sshPassword,
		cpu:         cpu,
		memory:      memory,
	}

	// Apply options
	for _, opt := range opts {
		opt(vetu)
	}

	// Apply default options (to cover those that weren't specified)
	if vetu.logger == nil {
		vetu.logger = &logger.LightweightStub{}
	}

	return vetu, nil
}

func NewFromIsolation(iso *api.Isolation_Vetu_, opts ...Option) (*Vetu, error) {
	vetu, err := New(iso.Vetu.Image, iso.Vetu.User, iso.Vetu.Password, iso.Vetu.Cpu, iso.Vetu.Memory,
		opts...)
	if err != nil {
		return nil, err
	}

	vetu.diskSize = iso.Vetu.DiskSize

	vetu.isolation = &api.Isolation{
		Type: iso,
	}

	return vetu, nil
}

func (vetu *Vetu) Pull(ctx context.Context, env map[string]string, logger *echelon.Logger) error {
	_, _, err := CmdWithLogger(ctx, env, logger, "pull", vetu.vmName)

	return err
}

func (vetu *Vetu) FQN(ctx context.Context) (string, error) {
	stdout, _, err := Cmd(ctx, nil, "fqn", vetu.vmName)
	if err != nil {
		return "", err
	}

	fqn := strings.TrimSpace(stdout)

	if fqn == "" {
		return "", fmt.Errorf("%w from Vetu", abstract.ErrEmptyFQN)
	}

	return fqn, nil
}

func (vetu *Vetu) CloneConfigureStart(
	ctx context.Context,
	config *runconfig.RunConfig,
) (*abstract.CloneAndConfigureResult, error) {
	ctx, prepareInstanceSpan := tracer.Start(ctx, "prepare-instance")
	defer prepareInstanceSpan.End()

	if cloneAndConfigureResult := vetu.cloneAndConfigureResult.Load(); cloneAndConfigureResult != nil {
		return cloneAndConfigureResult, nil
	}

	pullLogger := config.Logger().Scoped("pull virtual machine")
	if !config.VetuOptions.LazyPull {
		pullLogger.Infof("Pulling virtual machine %s...", vetu.vmName)

		if err := vetu.Pull(ctx, config.AdditionalEnvironment, pullLogger); err != nil {
			pullLogger.Errorf("Ignoring pull failure: %v", err)
			pullLogger.FinishWithType(echelon.FinishTypeFailed)
		} else {
			pullLogger.FinishWithType(echelon.FinishTypeSucceeded)
		}
	} else {
		pullLogger.FinishWithType(echelon.FinishTypeSkipped)
	}

	tmpVMName := fmt.Sprintf("%s%s-", vmNamePrefix, config.TaskID) + uuid.NewString()
	vm, err := NewVMClonedFrom(ctx,
		vetu.vmName, tmpVMName,
		config.AdditionalEnvironment,
		config.Logger(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create VM cloned from %q: %v", ErrFailed, vetu.vmName, err)
	}
	vetu.cleanupFuncs = append(vetu.cleanupFuncs, func() {
		if localHub := sentry.GetHubFromContext(ctx); localHub != nil {
			localHub.AddBreadcrumb(&sentry.Breadcrumb{
				Message: fmt.Sprintf("stopping and deleting the VM %s", vm.ident),
			}, nil)
		}

		_ = vm.Close()
	})

	if err := vm.Configure(ctx, vetu.cpu, vetu.memory, vetu.diskSize, config.Logger()); err != nil {
		return nil, fmt.Errorf("%w: failed to configure VM %q: %v", ErrFailed, vm.Ident(), err)
	}

	// Start the VM (asynchronously)
	vm.Start(ctx, vetu.bridgedInterface, vetu.hostNetworking)

	// Wait for the VM to start and get its IP address
	bootLogger := config.Logger().Scoped("boot virtual machine")

	var ip string

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-vm.ErrChan():
			return nil, err
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

	cloneAndConfigureResult := &abstract.CloneAndConfigureResult{
		IP: ip,
	}

	vetu.cloneAndConfigureResult.Store(cloneAndConfigureResult)

	return cloneAndConfigureResult, nil
}

func (vetu *Vetu) Run(ctx context.Context, config *runconfig.RunConfig) error {
	cloneAndConfigureResult, err := vetu.CloneConfigureStart(ctx, config)
	if err != nil {
		return err
	}

	err = remoteagent.WaitForAgent(ctx, vetu.logger, cloneAndConfigureResult.IP,
		vetu.sshUser, vetu.sshPassword, "linux", runtime.GOARCH,
		config, true, vetu.initializeHooks(config), nil, "")
	if err != nil {
		return err
	}

	return nil
}

func (vetu *Vetu) Isolation() *api.Isolation {
	return vetu.isolation
}

func (vetu *Vetu) Image() string {
	return vetu.vmName
}

func (vetu *Vetu) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return platform.NewUnix().GenericWorkingDir()
}

func (vetu *Vetu) Close() error {
	for _, cleanupFunc := range vetu.cleanupFuncs {
		cleanupFunc()
	}

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
