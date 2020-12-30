package parallels

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"path/filepath"
)

func cloneFromSuspended(ctx context.Context, vmPathFrom string) (*VM, error) {
	vm := &VM{
		uuid: uuid.New().String(),
	}

	serverInfo, err := GetServerInfo(ctx)
	if err != nil {
		return nil, err
	}

	newHome := filepath.Join(serverInfo.VMHome, fmt.Sprintf("cirrus-%s.pvm", vm.uuid))

	if err := CopyDir(vmPathFrom, newHome); err != nil {
		return nil, fmt.Errorf("%w: failed to copy VM from %q to %q: %v", ErrVMFailed, vmPathFrom, newHome, err)
	}

	_, stderr, err := Prlctl(ctx, "register", newHome, "--uuid", vm.uuid)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to import VM from %q: %q", ErrVMFailed, newHome, firstNonEmptyLine(stderr))
	}

	_, stderr, err = Prlctl(ctx, "start", vm.Ident())
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start VM %q: %q", ErrVMFailed, vm.Ident(), firstNonEmptyLine(stderr))
	}

	// Here isolation is done after the VM is started because
	// it's impossible to change suspended VM's settings
	if err := vm.isolate(ctx); err != nil {
		return nil, err
	}

	return vm, nil
}

func cloneFromDefault(ctx context.Context, vmNameFrom string) (*VM, error) {
	vm := &VM{
		name: fmt.Sprintf("cirrus-%s", uuid.New().String()),
	}

	_, stderr, err := Prlctl(ctx, "clone", vmNameFrom, "--name", vm.Ident())
	if err != nil {
		return nil, fmt.Errorf("%w: failed to clone VM %q: %q", ErrVMFailed, vm.Ident(), firstNonEmptyLine(stderr))
	}

	if err := vm.isolate(ctx); err != nil {
		return nil, err
	}

	_, stderr, err = Prlctl(ctx, "start", vm.Ident())
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start VM %q: %q", ErrVMFailed, vm.Ident(), firstNonEmptyLine(stderr))
	}

	return vm, nil
}
