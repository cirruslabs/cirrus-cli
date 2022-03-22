package tart

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/remoteagent"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
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
}

func New(vmName string, sshUser string, sshPassword string, opts ...Option) (*Tart, error) {
	parallels := &Tart{
		vmName:      vmName,
		sshUser:     sshUser,
		sshPassword: sshPassword,
	}

	// Apply options
	for _, opt := range opts {
		opt(parallels)
	}

	// Apply default options (to cover those that weren't specified)
	if parallels.logger == nil {
		parallels.logger = &logger.LightweightStub{}
	}

	return parallels, nil
}

func (tart *Tart) Run(ctx context.Context, config *runconfig.RunConfig) (err error) {
	vm, err := NewVMClonedFrom(ctx, tart.vmName)
	if err != nil {
		return fmt.Errorf("%w: failed to create VM cloned from %q: %v", ErrFailed, tart.vmName, err)
	}
	defer vm.Close()

	// Start the VM (asynchronously)
	vm.Start()

	// Wait for the VM to start and get it's DHCP address
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
			tart.logger.Debugf("failed to retrieve VM %s IP: %v\n", vm.Ident(), err)
			continue
		}

		break
	}

	tart.logger.Debugf("IP %s retrieved from VM %s, running agent...", ip, vm.Ident())

	return remoteagent.WaitForAgent(ctx, tart.logger, ip,
		tart.sshUser, tart.sshPassword, "darwin", "arm64", config, true)
}

func (tart *Tart) WorkingDirectory(projectDir string, dirtyMode bool) string {
	return platform.NewUnix().GenericWorkingDir()
}

func (tart *Tart) Close() error {
	return nil
}
