package tart

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
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
	"strconv"
	"strings"
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

	vm *VM
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

func (tart *Tart) Warmup(
	ctx context.Context,
	ident string,
	additionalEnvironment map[string]string,
	logger *echelon.Logger,
) error {
	return tart.bootVM(ctx, ident, additionalEnvironment, "", logger)
}

func (tart *Tart) bootVM(
	ctx context.Context,
	ident string,
	additionalEnvironment map[string]string,
	automountDir string,
	logger *echelon.Logger,
) error {
	ctx, prepareInstanceSpan := tracer.Start(ctx, "prepare-instance")
	defer prepareInstanceSpan.End()

	if localHub := sentry.GetHubFromContext(ctx); localHub != nil {
		localHub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetExtra("Softnet enabled", tart.softnet)
		})
	}

	var identToBeInjected string
	if ident != "" {
		identToBeInjected = fmt.Sprintf("%s-", ident)
	}

	tmpVMName := vmNamePrefix + identToBeInjected + uuid.NewString()
	vm, err := NewVMClonedFrom(ctx,
		tart.vmName, tmpVMName,
		false, // always clone from the base image
		additionalEnvironment,
		logger,
	)
	if err != nil {
		return fmt.Errorf("%w: failed to create VM cloned from %q: %v", ErrFailed, tart.vmName, err)
	}
	tart.vm = vm

	if err := vm.Configure(ctx, tart.cpu, tart.memory, tart.diskSize, tart.display, logger); err != nil {
		return fmt.Errorf("%w: failed to configure VM %q: %v", ErrFailed, vm.Ident(), err)
	}

	var directoryMounts []directoryMount
	if automountDir != "" {
		directoryMounts = append(directoryMounts, directoryMount{
			Name:     macOSAutomountDirectoryItem,
			Path:     automountDir,
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
					return fmt.Errorf("%w: volume source %q doesn't exist, failed to pre-create it: %v",
						ErrFailed, volume.Source, err)
				}

				volume.Cleanup = true
			} else {
				return fmt.Errorf("%w: volume source %q cannot be accessed: %v",
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
	bootLogger := logger.Scoped("boot virtual machine")

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
			tart.logger.Debugf("failed to retrieve VM %s IP: %v\n", vm.Ident(), err)
			continue
		}

		break
	}

	bootLogger.Errorf("VM was assigned with %s IP", ip)
	bootLogger.Finish(true)

	return nil
}

func (tart *Tart) Run(ctx context.Context, config *runconfig.RunConfig) (err error) {
	if tart.vm != nil && config.DirtyMode {
		return fmt.Errorf("%w: dirty mode is not supported for a warmed instance", ErrFailed)
	}
	if tart.vm == nil {
		automountProjectDir := ""
		if config.DirtyMode && config.ProjectDir != "" {
			automountProjectDir = config.ProjectDir
		}
		err = tart.bootVM(ctx, strconv.FormatInt(config.TaskID, 10), config.AdditionalEnvironment,
			automountProjectDir, config.Logger())
		if err != nil {
			return err
		}
	}

	ip, err := tart.vm.RetrieveIP(ctx)
	if err != nil {
		tart.logger.Debugf("failed to retrieve VM %s IP: %v\n", tart.vm.Ident(), err)
		return err
	}

	initializeHooks := tart.initializeHooks(config)
	terminateHooks := tart.terminateHooks(config)

	addTartListBreadcrumb(ctx)
	addDHCPDLeasesBreadcrumb(ctx)

	tart.logger.Debugf("IP %s retrieved from VM %s, running agent...", ip, tart.vm.Ident())

	err = remoteagent.WaitForAgent(ctx, tart.logger, ip,
		tart.sshUser, tart.sshPassword, "darwin", "arm64",
		config, true, initializeHooks, terminateHooks, "",
		map[string]string{"CIRRUS_VM_ID": tart.vm.Ident()})
	if err != nil {
		addTartListBreadcrumb(ctx)
		addDHCPDLeasesBreadcrumb(ctx)

		return err
	}

	return nil
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

func (tart *Tart) Close(ctx context.Context) error {
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

	if tart.vm == nil {
		return nil
	}

	if localHub := sentry.GetHubFromContext(ctx); localHub != nil {
		localHub.AddBreadcrumb(&sentry.Breadcrumb{
			Message: fmt.Sprintf("stopping and deleting the VM %s", tart.vm.ident),
		}, nil)
	}

	return tart.vm.Close()
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
