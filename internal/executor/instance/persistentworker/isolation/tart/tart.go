package tart

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/remoteagent"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"github.com/cirruslabs/echelon"
	"os/exec"
	"time"
)

var (
	ErrFailed = errors.New("tart isolation failed")
)

type Tart struct {
	logger      logger.Lightweight
	vmName      string
	sshUser     string
	sshPassword string
	cpu         uint32
	memory      uint32
}

func New(vmName string, sshUser string, sshPassword string, cpu uint32, memory uint32, opts ...Option) (*Tart, error) {
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

func (tart *Tart) Run(ctx context.Context, config *runconfig.RunConfig) (err error) {
	vm, err := NewVMClonedFrom(ctx, tart.vmName, tart.cpu, tart.memory, config.Logger())
	if err != nil {
		return fmt.Errorf("%w: failed to create VM cloned from %q: %v", ErrFailed, tart.vmName, err)
	}
	defer vm.Close()

	// Start the VM (asynchronously)
	vm.Start()

	// Wait for the VM to start and get it's DHCP address
	var ip string
	bootLogger := config.Logger().Scoped("boot virtual machine")

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

	tart.logger.Debugf("IP %s retrieved from VM %s, running agent...", ip, vm.Ident())

	bootLogger.Errorf("VM was assigned with %s IP", ip)
	bootLogger.Finish(true)

	err = tart.syncProjectDir(ctx, config, ip)
	if err != nil {
		return err
	}

	return remoteagent.WaitForAgent(ctx, tart.logger, ip,
		tart.sshUser, tart.sshPassword, "darwin", "arm64", config, true)
}

func (tart *Tart) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return platform.NewUnix().GenericWorkingDir()
}

func (tart *Tart) Close() error {
	return nil
}

func (tart *Tart) syncProjectDir(ctx context.Context, config *runconfig.RunConfig, ip string) error {
	if config.ProjectDir == "" {
		return nil
	}
	syncLogger := config.Logger().Scoped("syncing project directory")

	remoteDst := fmt.Sprintf("rsync://%s@%s:%s", tart.sshUser, ip, platform.NewUnix().GenericWorkingDir())
	rsyncCommand := fmt.Sprintf(
		"rsync -e 'ssh -o StrictHostKeyChecking=no' -r --rsync-path='mkdir -p %s && rsync' --filter=':- .gitignore' %s/ %s",
		platform.NewUnix().GenericWorkingDir(), config.ProjectDir, remoteDst,
	)

	cmd := exec.CommandContext(ctx, "sh", "-c", rsyncCommand)

	cmd.Env = []string{fmt.Sprintf("RSYNC_PASSWORD=%s", tart.sshPassword)}

	cmd.Stdout = &loggerAsWriter{
		level:    echelon.InfoLevel,
		delegate: syncLogger,
	}
	cmd.Stderr = &loggerAsWriter{
		level:    echelon.WarnLevel,
		delegate: syncLogger,
	}

	err := cmd.Run()
	syncLogger.Finish(err == nil)
	return err
}
