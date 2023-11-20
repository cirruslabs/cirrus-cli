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
	"golang.org/x/crypto/ssh"
	"strings"
	"time"
)

var (
	ErrFailed     = errors.New("vetu isolation failed")
	ErrSyncFailed = errors.New("failed to sync project directory")
)

const vmNamePrefix = "cirrus-cli-"

type Vetu struct {
	logger           logger.Lightweight
	vmName           string
	sshUser          string
	sshPassword      string
	cpu              uint32
	memory           uint32
	bridgedInterface string
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

func (vetu *Vetu) Run(ctx context.Context, config *runconfig.RunConfig) error {
	tmpVMName := fmt.Sprintf("%s%d-", vmNamePrefix, config.TaskID) + uuid.NewString()
	vm, err := NewVMClonedFrom(ctx,
		vetu.vmName, tmpVMName,
		vetu.cpu, vetu.memory,
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

	// Start the VM (asynchronously)
	vm.Start(ctx, vetu.bridgedInterface)

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

	err = remoteagent.WaitForAgent(ctx, vetu.logger, ip,
		vetu.sshUser, vetu.sshPassword, "linux", "arm64",
		config, true, vetu.initializeHooks(config), nil, "")
	if err != nil {
		return err
	}

	return nil
}

func (vetu *Vetu) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return platform.NewUnix().GenericWorkingDir()
}

func (vetu *Vetu) Close() error {
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
