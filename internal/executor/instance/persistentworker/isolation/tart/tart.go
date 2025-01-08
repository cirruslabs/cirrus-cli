package tart

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/projectdirsyncer"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/remoteagent"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/cirruslabs/echelon"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"path"
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
	macOSAutomountDirectoryPath = "$HOME/working-dir"
	macOSAutomountDirectoryItem = "working-dir"
)

type Tart struct {
	logger      logger.Lightweight
	vmName      string
	sshUser     string
	sshPassword string
	sshPort     uint16
	cpu         uint32
	memory      uint32
	diskSize    uint32
	softnet     bool
	display     string
	volumes     []*api.Isolation_Tart_Volume

	vm              *VM
	initializeHooks []remoteagent.WaitForAgentHook
	terminateHooks  []remoteagent.WaitForAgentHook
}

func New(
	vmName string,
	sshUser string,
	sshPassword string,
	sshPort uint16,
	cpu uint32,
	memory uint32,
	opts ...Option,
) (*Tart, error) {
	tart := &Tart{
		vmName:      vmName,
		sshUser:     sshUser,
		sshPassword: sshPassword,
		sshPort:     sshPort,
		cpu:         cpu,
		memory:      memory,
	}

	// Apply options
	for _, opt := range opts {
		opt(tart)
	}

	// Apply default options (to cover those that weren't specified)
	if tart.sshPort == 0 {
		tart.sshPort = 22
	}

	if tart.logger == nil {
		tart.logger = &logger.LightweightStub{}
	}

	return tart, nil
}

func (tart *Tart) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("image", tart.Image()),
		attribute.String("instance_type", "tart"),
		attribute.Float64("instance_cpus", float64(tart.cpu)),
	}
}

func (tart *Tart) Warmup(
	ctx context.Context,
	ident string,
	additionalEnvironment map[string]string,
	lazyPull bool,
	warmupScript string,
	warmupTimeout time.Duration,
	logger *echelon.Logger,
) error {
	err := tart.bootVM(ctx, ident, additionalEnvironment, "", lazyPull, logger)
	if err != nil {
		return err
	}
	ip, err := tart.vm.RetrieveIP(ctx)
	if err != nil {
		return err
	}

	sshClient, err := remoteagent.WaitForSSH(ctx, fmt.Sprintf("%s:%d", ip, tart.sshPort), tart.sshUser,
		tart.sshPassword, tart.logger)
	if err != nil {
		return err
	}
	defer func() { _ = sshClient.Close() }()

	if warmupScript == "" {
		return nil
	}

	logger.Infof("running warm-up script...")

	ctx, prepareInstanceSpan := tracer.Start(ctx, "warmup-script")
	defer prepareInstanceSpan.End()

	// Work around x/crypto/ssh not being context.Context-friendly (e.g. https://github.com/golang/go/issues/20288)
	var monitorCtx context.Context
	var monitorCancel context.CancelFunc
	if warmupTimeout != 0 {
		monitorCtx, monitorCancel = context.WithTimeoutCause(ctx, warmupTimeout, abstract.ErrWarmupTimeout)
	} else {
		monitorCtx, monitorCancel = context.WithCancel(ctx)
	}
	go func() {
		<-monitorCtx.Done()
		_ = sshClient.Close()
	}()
	defer monitorCancel()

	sshSess, err := sshClient.NewSession()
	if err != nil {
		return fmt.Errorf("%w: failed to create new SSH session: %v", abstract.ErrWarmupScriptFailed, err)
	}

	// Log output from the virtual machine
	stdout, err := sshSess.StdoutPipe()
	if err != nil {
		return fmt.Errorf("%w: failed to open SSH session stdout pipe: %v", abstract.ErrWarmupScriptFailed, err)
	}
	stderr, err := sshSess.StderrPipe()
	if err != nil {
		return fmt.Errorf("%w: failed to open SSH session stderr pipe: %v", abstract.ErrWarmupScriptFailed, err)
	}
	go func() {
		output := io.MultiReader(stdout, stderr)

		scanner := bufio.NewScanner(output)

		for scanner.Scan() {
			logger.Debugf("VM: %s", scanner.Text())
		}
	}()

	stdinBuf, err := sshSess.StdinPipe()
	if err != nil {
		return fmt.Errorf("%w: failed to open SSH session stdin pipe: %v", abstract.ErrWarmupScriptFailed, err)
	}

	if err := sshSess.Shell(); err != nil {
		return fmt.Errorf("%w: failed to invoke SSH shell on the VM: %v", abstract.ErrWarmupScriptFailed, err)
	}

	_, err = stdinBuf.Write([]byte(warmupScript + "\nexit\n"))
	if err != nil {
		return fmt.Errorf("%w: failed to write the warm-up script to the shell: %v",
			abstract.ErrWarmupScriptFailed, err)
	}

	if err := sshSess.Wait(); err != nil {
		// Work around x/crypto/ssh not being context.Context-friendly (e.g. https://github.com/golang/go/issues/20288)
		if err := monitorCtx.Err(); err != nil {
			if errors.Is(context.Cause(monitorCtx), abstract.ErrWarmupTimeout) {
				logger.Warnf("%v, ignoring...", context.Cause(monitorCtx))

				return nil
			}

			return err
		}

		return fmt.Errorf("%w: failed to execute the warm-up script: %v", abstract.ErrWarmupScriptFailed, err)
	}

	return nil
}

func PrePull(ctx context.Context, image string, logger *echelon.Logger) error {
	_, _, err := CmdWithLogger(ctx, nil, logger, "pull", image)

	return err
}

func (tart *Tart) bootVM(
	ctx context.Context,
	ident string,
	additionalEnvironment map[string]string,
	automountDir string,
	lazyPull bool,
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
		lazyPull,
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
		tag := fmt.Sprintf("tart.virtiofs.%s", macOSAutomountDirectoryItem)

		directoryMounts = append(directoryMounts, directoryMount{
			Name:     macOSAutomountDirectoryItem,
			Path:     automountDir,
			Tag:      tag,
			ReadOnly: false,
		})

		tart.initializeHooks = append(tart.initializeHooks, mountWorkingDirectoryHook(tag, logger))
		tart.terminateHooks = append(tart.terminateHooks, unmountWorkingDirectoryHook(logger))
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
	bootLogger := logger.Scoped("boot virtual machine")

	ipCtx, ipCtxCancel := context.WithTimeoutCause(ctx, 10*time.Minute,
		fmt.Errorf("timed out while trying to retrieve the VM %s IP", vm.Ident()))
	defer ipCtxCancel()

	ip, err := tart.retrieveIPLoop(ipCtx, vm)
	if err != nil {
		return err
	}

	bootLogger.Errorf("VM was assigned with %s IP", ip)

	bootLogger.Finish(true)

	return nil
}

func (tart *Tart) retrieveIPLoop(ctx context.Context, vm *VM) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case err := <-vm.ErrChan():
			return "", err
		default:
			time.Sleep(time.Second)
		}

		ip, err := vm.RetrieveIP(ctx)
		if err != nil {
			tart.logger.Debugf("failed to retrieve VM %s IP: %v\n", vm.Ident(), err)
			continue
		}

		return ip, nil
	}
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
		err = tart.bootVM(ctx, config.TaskID, config.AdditionalEnvironment,
			automountProjectDir, config.TartOptions.LazyPull, config.Logger())
		if err != nil {
			return err
		}
	}

	ip, err := tart.vm.RetrieveIP(ctx)
	if err != nil {
		tart.logger.Debugf("failed to retrieve VM %s IP: %v\n", tart.vm.Ident(), err)
		return err
	}

	initializeHooks := tart.getInitializeHooks(config)
	terminateHooks := tart.getTerminateHooks(config)

	addTartListBreadcrumb(ctx)
	addDHCPDLeasesBreadcrumb(ctx)

	tart.logger.Debugf("IP %s retrieved from VM %s, running agent...", ip, tart.vm.Ident())

	// Try to determine the agent OS, falling back to "darwin"
	// in case anything goes wrong to preserve the old behavior
	agentOS := "darwin"

	info, err := tart.vm.Info(ctx)
	if err != nil {
		tart.logger.Debugf("failed to auto-detect the VM's operating system, assuming %q", agentOS)
	} else if info.OS != "" {
		agentOS = info.OS
		tart.logger.Debugf("successfully auto-detected the VM's operating system as %q", agentOS)
	}

	err = remoteagent.WaitForAgent(ctx, tart.logger, fmt.Sprintf("%s:%d", ip, tart.sshPort),
		tart.sshUser, tart.sshPassword, agentOS, "arm64",
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

func (tart *Tart) getInitializeHooks(config *runconfig.RunConfig) []remoteagent.WaitForAgentHook {
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
					syncLogger.Finish(false)
					return err
				}

				if err := sshSess.Run(command); err != nil {
					_ = sshSess.Close()

					syncLogger.Finish(false)
					return err
				}

				_ = sshSess.Close()
			}

			syncLogger.Finish(true)
			return nil
		})
	}

	return append(tart.initializeHooks, hooks...)
}

func (tart *Tart) getTerminateHooks(config *runconfig.RunConfig) []remoteagent.WaitForAgentHook {
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

	return append(tart.terminateHooks, hooks...)
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
