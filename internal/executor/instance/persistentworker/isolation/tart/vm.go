package tart

import (
	"context"
	"fmt"
	"github.com/cirruslabs/echelon"
	"strconv"
	"strings"
	"sync"
)

type VM struct {
	ident string

	env map[string]string

	subCtx       context.Context
	subCtxCancel context.CancelFunc
	wg           sync.WaitGroup
	errChan      chan error
}

type directoryMount struct {
	Name     string
	Path     string
	ReadOnly bool
}

func NewVMClonedFrom(
	ctx context.Context,
	from string,
	to string,
	cpu uint32,
	memory uint32,
	lazyPull bool,
	env map[string]string,
	logger *echelon.Logger,
) (*VM, error) {
	subCtx, subCtxCancel := context.WithCancel(ctx)

	vm := &VM{
		ident:        to,
		env:          env,
		subCtx:       subCtx,
		subCtxCancel: subCtxCancel,
		errChan:      make(chan error, 1),
	}

	pullLogger := logger.Scoped("pull virtual machine")
	if !lazyPull {
		pullLogger.Infof("Pulling virtual machine %s...", from)

		if _, _, err := CmdWithLogger(ctx, vm.env, pullLogger, "pull", from); err != nil {
			pullLogger.Errorf("Ignoring pull failure: %w", err)
			pullLogger.FinishWithType(echelon.FinishTypeFailed)
		} else {
			pullLogger.FinishWithType(echelon.FinishTypeSucceeded)
		}
	} else {
		pullLogger.FinishWithType(echelon.FinishTypeSkipped)
	}

	cloneLogger := logger.Scoped("clone virtual machine")
	cloneLogger.Infof("Cloning virtual machine %s...", from)

	if _, _, err := CmdWithLogger(ctx, vm.env, cloneLogger, "clone", from, vm.ident); err != nil {
		cloneLogger.Finish(false)
		return nil, err
	}

	if cpu != 0 {
		cpuStr := strconv.FormatUint(uint64(cpu), 10)

		_, _, err := CmdWithLogger(ctx, vm.env, cloneLogger, "set", vm.ident, "--cpu", cpuStr)
		if err != nil {
			cloneLogger.Finish(false)
			return nil, err
		}
	}
	if memory != 0 {
		memoryStr := strconv.FormatUint(uint64(memory), 10)

		_, _, err := CmdWithLogger(ctx, vm.env, cloneLogger, "set", vm.ident, "--memory", memoryStr)
		if err != nil {
			cloneLogger.Finish(false)
			return nil, err
		}
	}

	cloneLogger.Finish(true)
	return vm, nil
}

func (vm *VM) Ident() string {
	return vm.ident
}

func (vm *VM) Start(
	softnet bool,
	directoryMounts []directoryMount,
) {
	vm.wg.Add(1)

	go func() {
		defer vm.wg.Done()

		args := []string{"run", "--no-graphics"}

		if softnet {
			args = append(args, "--net-softnet")
		}

		for _, directoryMount := range directoryMounts {
			dirArgumentValue := fmt.Sprintf("%s:%s", directoryMount.Name, directoryMount.Path)

			if directoryMount.ReadOnly {
				dirArgumentValue += ":ro"
			}

			args = append(args, "--dir", dirArgumentValue)
		}

		args = append(args, vm.ident)

		_, _, err := Cmd(vm.subCtx, vm.env, args...)
		vm.errChan <- err
	}()
}

func (vm *VM) ErrChan() chan error {
	return vm.errChan
}

func (vm *VM) RetrieveIP(ctx context.Context) (string, error) {
	stdout, _, err := Cmd(ctx, vm.env, "ip", vm.ident)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout), nil
}

func (vm *VM) Close() error {
	ctx := context.Background()

	// Try to gracefully terminate the VM
	_, _, _ = Cmd(ctx, vm.env, "stop", "--timeout", "5", vm.ident)

	vm.subCtxCancel()
	vm.wg.Wait()

	_, _, err := Cmd(ctx, vm.env, "delete", vm.ident)

	return err
}
