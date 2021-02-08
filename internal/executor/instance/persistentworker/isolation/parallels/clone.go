package parallels

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"os"
	"path/filepath"
)

func cloneFromSuspended(ctx context.Context, vmPathFrom string) (*VM, error) {
	vm := &VM{
		uuid: uuid.New().String(),

		shouldRenewDHCP:  true,
		delayedIsolation: true,
	}

	serverInfo, err := GetServerInfo(ctx)
	if err != nil {
		return nil, err
	}

	newHome := filepath.Join(serverInfo.VMHome, fmt.Sprintf("cirrus-%s.pvm", vm.uuid))

	if err := CopyDir(vmPathFrom, newHome); err != nil {
		return nil, fmt.Errorf("%w: failed to copy VM from %q to %q: %v", ErrVMFailed, vmPathFrom, newHome, err)
	}

	_, _, err = Prlctl(ctx, "register", newHome, "--uuid", vm.uuid)
	if err != nil {
		// Cleanup
		_ = os.RemoveAll(newHome)

		return nil, fmt.Errorf("%w: failed to import VM from %q: %v", ErrVMFailed, newHome, err)
	}

	return vm, nil
}

func cloneFromDefault(ctx context.Context, vmNameFrom string) (*VM, error) {
	vm := &VM{
		name: fmt.Sprintf("cirrus-%s", uuid.New().String()),
	}

	_, _, err := Prlctl(ctx, "clone", vmNameFrom, "--name", vm.Ident())
	if err != nil {
		return nil, fmt.Errorf("%w: failed to clone VM %q: %v", ErrVMFailed, vm.Ident(), err)
	}

	return vm, nil
}
