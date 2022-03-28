package tart

import (
	"context"
	"github.com/google/uuid"
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

func NewVMClonedFrom(ctx context.Context, from string) (*VM, error) {
	subCtx, subCtxCancel := context.WithCancel(ctx)

	vm := &VM{
		ident:        "cirrus-cli-" + uuid.New().String(),
		subCtx:       subCtx,
		subCtxCancel: subCtxCancel,
		errChan:      make(chan error, 1),
	}

	if _, _, err := Cmd(ctx, "clone", from, vm.ident); err != nil {
		return nil, err
	}

	return vm, nil
}

func (vm *VM) Ident() string {
	return vm.ident
}

func (vm *VM) Start() {
	vm.wg.Add(1)

	go func() {
		defer vm.wg.Done()

		_, _, err := Cmd(vm.subCtx, "run", vm.ident)
		vm.errChan <- err
	}()
}

func (vm *VM) ErrChan() chan error {
	return vm.errChan
}

func (vm *VM) RetrieveIP(ctx context.Context) (string, error) {
	stdout, _, err := Cmd(ctx, "ip", vm.ident)
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
