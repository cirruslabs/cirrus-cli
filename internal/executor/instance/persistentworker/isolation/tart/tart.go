package tart

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
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"
	"golang.org/x/crypto/ssh"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"time"
)

var (
	ErrFailed     = errors.New("tart isolation failed")
	ErrSyncFailed = errors.New("failed to sync project directory")

	tracer = otel.Tracer("tart")
)

const (
	vmNamePrefix                = "cirrus-cli-"
	macOSAutomountDirectoryPath = "/Volumes/My Shared Files"
	macOSAutomountDirectoryItem = "working-dir"
)

type Tart struct {
	cloneAndConfigureResult atomic.Pointer[abstract.CloneAndConfigureResult]
	cleanupFuncs            []func()

	logger      logger.Lightweight
	vmName      string
	sshUser     string
	sshPassword string
	cpu         uint32
	memory      uint32
	diskSize    uint32
	softnet     bool
	display     string
	volumes     []*api.Isolation_Tart_Volume
	isolation   *api.Isolation

	mountTemporaryWorkingDirectoryFromHost bool
}

func New(
	vmName string,
	sshUser string,
	sshPassword string,
	cpu uint32,
	memory uint32,
	opts ...Option,
) (*Tart, error) {
	tart := &Tart{
		vmName:      vmName,
		sshUser:     sshUser,
		sshPassword: sshPassword,
		cpu:         cpu,
		memory:      memory,
	}

	// Apply options
	for _, opt := range opts {
		opt(tart)
	}

	// Apply default options (to cover those that weren't specified)
	if tart.logger == nil {
		tart.logger = &logger.LightweightStub{}
	}

	return tart, nil
}

func NewFromIsolation(iso *api.Isolation_Tart_, opts ...Option) (*Tart, error) {
	tart, err := New(iso.Tart.Image, iso.Tart.User, iso.Tart.Password, iso.Tart.Cpu, iso.Tart.Memory,
		opts...)
	if err != nil {
		return nil, err
	}

	tart.diskSize = iso.Tart.DiskSize
	tart.display = iso.Tart.Display
	tart.volumes = iso.Tart.Volumes

	tart.isolation = &api.Isolation{
		Type: iso,
	}

	tart.mountTemporaryWorkingDirectoryFromHost = iso.Tart.MountTemporaryWorkingDirectoryFromHost

	return tart, nil
}

func (tart *Tart) Pull(ctx context.Context, env map[string]string, logger *echelon.Logger) error {
	_, _, err := CmdWithLogger(ctx, env, logger, "pull", tart.vmName)

	return err
}

func (tart *Tart) FQN(ctx context.Context) (string, error) {
	stdout, _, err := Cmd(ctx, nil, "fqn", tart.vmName)
	if err != nil {
		return "", err
	}

	fqn := strings.TrimSpace(stdout)

	if fqn == "" {
		return "", fmt.Errorf("%w from Tart", abstract.ErrEmptyFQN)
	}

	return fqn, nil
}

//nolint:gocognit // keep it the same as it was before for now to aid in PR review
func (tart *Tart) CloneConfigureStart(
	ctx context.Context,
	config *runconfig.RunConfig,
) (*abstract.CloneAndConfigureResult, error) {
	ctx, prepareInstanceSpan := tracer.Start(ctx, "prepare-instance")
	defer prepareInstanceSpan.End()

	if cloneAndConfigureResult := tart.cloneAndConfigureResult.Load(); cloneAndConfigureResult != nil {
		return cloneAndConfigureResult, nil
	}

	if localHub := sentry.GetHubFromContext(ctx); localHub != nil {
		localHub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetExtra("Softnet enabled", tart.softnet)
		})
	}

	pullLogger := config.Logger().Scoped("pull virtual machine")
	if !config.TartOptions.LazyPull {
		pullLogger.Infof("Pulling virtual machine %s...", tart.vmName)

		if err := tart.Pull(ctx, config.AdditionalEnvironment, pullLogger); err != nil {
			pullLogger.Errorf("Ignoring pull failure: %v", err)
			pullLogger.FinishWithType(echelon.FinishTypeFailed)
		} else {
			pullLogger.FinishWithType(echelon.FinishTypeSucceeded)
		}
	} else {
		pullLogger.FinishWithType(echelon.FinishTypeSkipped)
	}

	tmpVMName := fmt.Sprintf("%s%d-", vmNamePrefix, config.TaskID) + uuid.NewString()
	vm, err := NewVMClonedFrom(ctx,
		tart.vmName, tmpVMName,
		config.AdditionalEnvironment,
		config.Logger(),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create VM cloned from %q: %v", ErrFailed, tart.vmName, err)
	}
	tart.cleanupFuncs = append(tart.cleanupFuncs, func() {
		if localHub := sentry.GetHubFromContext(ctx); localHub != nil {
			localHub.AddBreadcrumb(&sentry.Breadcrumb{
				Message: fmt.Sprintf("stopping and deleting the VM %s", vm.ident),
			}, nil)
		}

		_ = vm.Close()
	})

	if err := vm.Configure(ctx, tart.cpu, tart.memory, tart.diskSize, tart.display, config.Logger()); err != nil {
		return nil, fmt.Errorf("%w: failed to configure VM %q: %v", ErrFailed, vm.Ident(), err)
	}

	// Start the VM (asynchronously)
	var preCreatedWorkingDir string

	if tart.mountTemporaryWorkingDirectoryFromHost {
		tmpDir, err := os.MkdirTemp("", "")
		if err != nil {
			return nil, fmt.Errorf("%w: failed to create temporary directory: %v",
				ErrFailed, err)
		}
		tart.cleanupFuncs = append(tart.cleanupFuncs, func() {
			_ = os.RemoveAll(tmpDir)
		})

		config.ProjectDir = tmpDir
		config.DirtyMode = true
		preCreatedWorkingDir = tart.WorkingDirectory(config.ProjectDir, config.DirtyMode)
	}

	var directoryMounts []directoryMount
	if config.DirtyMode {
		directoryMounts = append(directoryMounts, directoryMount{
			Name:     macOSAutomountDirectoryItem,
			Path:     config.ProjectDir,
			ReadOnly: false,
		})
	}

	// Convert volumes to directory mounts
	for _, volume := range tart.volumes {
		if volume.Name == "" {
			volume.Name = uuid.NewString()
		}

		_, err = os.Stat(volume.Source)
		if err != nil {
			if os.IsNotExist(err) {
				if err := os.Mkdir(volume.Source, 0755); err != nil {
					return nil, fmt.Errorf("%w: volume source %q doesn't exist, failed to pre-create it: %v",
						ErrFailed, volume.Source, err)
				}

				volume.Cleanup = true
			} else {
				return nil, fmt.Errorf("%w: volume source %q cannot be accessed: %v",
					ErrFailed, volume.Source, err)
			}
		}

		directoryMounts = append(directoryMounts, directoryMount{
			Name:     volume.Name,
			Path:     volume.Source,
			ReadOnly: volume.ReadOnly,
		})
	}

	vm.Start(ctx, tart.softnet, directoryMounts)

	// Wait for the VM to start and get it's DHCP address
	var ip string
	bootLogger := config.Logger().Scoped("boot virtual machine")

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
			tart.logger.Debugf("failed to retrieve VM %s IP: %v\n", vm.Ident(), err)
			continue
		}

		break
	}

	tart.logger.Debugf("IP %s retrieved from VM %s, running agent...", ip, vm.Ident())

	bootLogger.Errorf("VM was assigned with %s IP", ip)
	bootLogger.Finish(true)

	addTartListBreadcrumb(ctx)
	addDHCPDLeasesBreadcrumb(ctx)

	cloneAndConfigureResult := &abstract.CloneAndConfigureResult{
		IP:                   ip,
		PreCreatedWorkingDir: preCreatedWorkingDir,
	}

	tart.cloneAndConfigureResult.Store(cloneAndConfigureResult)

	return cloneAndConfigureResult, nil
}

func (tart *Tart) Run(ctx context.Context, config *runconfig.RunConfig) error {
	cloneAndConfigureResult, err := tart.CloneConfigureStart(ctx, config)
	if err != nil {
		return err
	}

	initializeHooks := tart.initializeHooks(config)
	terminateHooks := tart.terminateHooks(config)

	err = remoteagent.WaitForAgent(ctx, tart.logger, cloneAndConfigureResult.IP,
		tart.sshUser, tart.sshPassword, "darwin", "arm64",
		config, true, initializeHooks, terminateHooks, cloneAndConfigureResult.PreCreatedWorkingDir)
	if err != nil {
		addTartListBreadcrumb(ctx)
		addDHCPDLeasesBreadcrumb(ctx)

		return err
	}

	return nil
}

func (tart *Tart) Isolation() *api.Isolation {
	return tart.isolation
}

func (tart *Tart) Image() string {
	return tart.vmName
}

func (tart *Tart) WorkingDirectory(projectDir string, dirtyMode bool) string {
	if dirtyMode {
		return path.Join(macOSAutomountDirectoryPath, macOSAutomountDirectoryItem)
	}

	return platform.NewUnix().GenericWorkingDir()
}

func (tart *Tart) Close() error {
	// Cleanup volumes created by us
	for _, volume := range tart.volumes {
		if !volume.Cleanup {
			continue
		}

		volumeIdent := volume.Name

		if volumeIdent == "" {
			volumeIdent = volume.Source
		}

		if err := os.RemoveAll(volume.Source); err != nil {
			return fmt.Errorf("%w: failed to cleanup volume %q: %v", ErrFailed, volumeIdent, err)
		}
	}

	for _, cleanupFunc := range tart.cleanupFuncs {
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

func (tart *Tart) initializeHooks(config *runconfig.RunConfig) []remoteagent.WaitForAgentHook {
	var hooks []remoteagent.WaitForAgentHook

	if config.ProjectDir != "" && !config.DirtyMode {
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

	if len(tart.volumes) != 0 {
		hooks = append(hooks, func(ctx context.Context, sshClient *ssh.Client) error {
			syncLogger := config.Logger().Scoped("symlinking volume mounts")

			for _, volume := range tart.volumes {
				if volume.Target == "" {
					continue
				}

				command := fmt.Sprintf("ln -s \"/Volumes/My Shared Files/%s\" \"%s\"",
					volume.Name, volume.Target)

				syncLogger.Infof("running command: %s", command)

				sshSess, err := sshClient.NewSession()
				if err != nil {
					return err
				}

				if err := sshSess.Run(command); err != nil {
					_ = sshSess.Close()

					return err
				}

				_ = sshSess.Close()
			}

			syncLogger.Finish(true)
			return nil
		})
	}

	return hooks
}

func (tart *Tart) terminateHooks(config *runconfig.RunConfig) []remoteagent.WaitForAgentHook {
	var hooks []remoteagent.WaitForAgentHook

	targetfulVolumes := lo.Filter(tart.volumes, func(volume *api.Isolation_Tart_Volume, index int) bool {
		return volume.Target != ""
	})

	if len(targetfulVolumes) != 0 {
		hooks = append(hooks, func(ctx context.Context, sshClient *ssh.Client) error {
			syncLogger := config.Logger().Scoped("removing volume mount symlinks")

			for _, volume := range targetfulVolumes {
				command := fmt.Sprintf("rm -f \"%s\"", volume.Target)

				syncLogger.Infof("running command: %s", command)

				sshSess, err := sshClient.NewSession()
				if err != nil {
					return err
				}

				if err := sshSess.Run(command); err != nil {
					_ = sshSess.Close()

					return err
				}

				_ = sshSess.Close()
			}

			syncLogger.Finish(true)
			return nil
		})
	}

	return hooks
}

func addTartListBreadcrumb(ctx context.Context) {
	localHub := sentry.GetHubFromContext(ctx)
	if localHub == nil {
		return
	}

	stdout, stderr, err := Cmd(context.Background(), nil, "list", "--format=json")

	localHub.AddBreadcrumb(&sentry.Breadcrumb{
		Message: "Tart VM list",
		Data: map[string]interface{}{
			"err":    err,
			"stdout": stdout,
			"stderr": stderr,
		},
	}, nil)
}

func addDHCPDLeasesBreadcrumb(ctx context.Context) {
	localHub := sentry.GetHubFromContext(ctx)
	if localHub == nil {
		return
	}

	dhcpdLeasesBytes, err := os.ReadFile("/var/db/dhcpd_leases")

	localHub.AddBreadcrumb(&sentry.Breadcrumb{
		Message: "DHCPD server leases",
		Data: map[string]interface{}{
			"err":                  err,
			"/var/db/dhcpd_leases": string(dhcpdLeasesBytes),
		},
	}, nil)
}
