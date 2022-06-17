package tart

import (
	"context"
	"github.com/cirruslabs/echelon"
	"github.com/google/uuid"
	"strconv"
	"strings"
	"sync"
)

type VM struct {
	ident string

	subCtx       context.Context
	subCtxCancel context.CancelFunc
	wg           sync.WaitGroup
	errChan      chan error
}

func NewVMClonedFrom(
	ctx context.Context,
	from string,
	cpu uint32,
	memory uint32,
	lazyPull bool,
	logger *echelon.Logger,
) (*VM, error) {
	subCtx, subCtxCancel := context.WithCancel(ctx)

	vm := &VM{
		ident:        "cirrus-cli-" + uuid.New().String(),
		subCtx:       subCtx,
		subCtxCancel: subCtxCancel,
		errChan:      make(chan error, 1),
	}

	pullLogger := logger.Scoped("pull virtual machine")
	if !lazyPull {
		pullLogger.Infof("Pulling virtual machine %s...", from)

		if _, _, err := CmdWithLogger(ctx, pullLogger, "pull", from); err != nil {
			pullLogger.FinishWithType(echelon.FinishTypeFailed)
			return nil, err
		}

		pullLogger.FinishWithType(echelon.FinishTypeSucceeded)
	} else {
		pullLogger.FinishWithType(echelon.FinishTypeSkipped)
	}

	cloneLogger := logger.Scoped("clone virtual machine")
	cloneLogger.Infof("Cloning virtual machine %s...", from)

	if _, _, err := CmdWithLogger(ctx, cloneLogger, "clone", from, vm.ident); err != nil {
		cloneLogger.Finish(false)
		return nil, err
	}

	if cpu != 0 {
		cpuStr := strconv.FormatUint(uint64(cpu), 10)
		if _, _, err := CmdWithLogger(ctx, cloneLogger, "set", vm.ident, "--cpu", cpuStr); err != nil {
			cloneLogger.Finish(false)
			return nil, err
		}
	}
	if memory != 0 {
		memoryStr := strconv.FormatUint(uint64(memory), 10)
		if _, _, err := CmdWithLogger(ctx, cloneLogger, "set", vm.ident, "--memory", memoryStr); err != nil {
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

func (vm *VM) Start() {
	vm.wg.Add(1)

	go func() {
		defer vm.wg.Done()

		_, _, err := Cmd(vm.subCtx, "run", "--no-graphics", vm.ident)
		vm.errChan <- err
	}()
}

func (vm *VM) ErrChan() chan error {
	return vm.errChan
}

func (vm *VM) RetrieveIP(ctx context.Context) (string, error) {
	stdout, _, err := Cmd(ctx, "ip", "--wait", "60", vm.ident)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout), nil
}

func (vm *VM) Close() error {
	vm.subCtxCancel()
	vm.wg.Wait()

	_, _, err := Cmd(context.Background(), "delete", vm.ident)

	return err
}
