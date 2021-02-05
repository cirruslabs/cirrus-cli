package parallels

import (
	"fmt"
	"github.com/mitchellh/go-ps"
	"syscall"
)

const vmProcessName = "prl_vm_app"

// Sometimes issuing the prlctl stop --kill command isn't enough, and we're not
// supposed to be running other VMs anyway, so clean them up by killing processes,
// according to the hint in the unofficial documentation[1].
//
// [1]: https://virtuozzosupport.force.com/s/article/000014272
func ensureNoVMsRunning() error {
	processes, err := ps.Processes()
	if err != nil {
		return fmt.Errorf("%w: failed to retrieve a list of processes: %v", ErrVMKiller, err)
	}

	for _, process := range processes {
		if process.Executable() != vmProcessName {
			continue
		}

		if err := syscall.Kill(process.Pid(), syscall.SIGKILL); err != nil {
			return fmt.Errorf("%w: failed to kill %s process: %v", ErrVMKiller, vmProcessName, err)
		}
	}

	return nil
}
