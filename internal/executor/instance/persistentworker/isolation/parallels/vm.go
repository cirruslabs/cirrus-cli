package parallels

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var ErrVMFailed = errors.New("Parallels VM operation failed")

type VM struct {
	uuid string
	name string

	shouldRenewDHCP  bool
	delayedIsolation bool
}

type NetworkAdapterInfo struct {
	MAC string
}

type HardwareInfo struct {
	Net0 NetworkAdapterInfo
}

type VirtualMachineInfo struct {
	ID       string
	Name     string
	State    string
	Home     string
	Hardware HardwareInfo
}

func NewVMClonedFrom(ctx context.Context, vmNameFrom string) (*VM, error) {
	if err := ensureNoVMsRunning(); err != nil {
		return nil, err
	}

	// We use different cloning strategy depending on the source VM's state
	vmInfoFrom, err := retrieveInfo(ctx, vmNameFrom)
	if err != nil {
		return nil, err
	}

	// Check if VM is packed
	if strings.HasSuffix(vmInfoFrom.Home, ".pvmp") {
		// Let's unpack it!
		_, stderr, err := Prlctl(ctx, "unpack", vmNameFrom)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to unpack VM %q: %q", ErrVMFailed, vmNameFrom, firstNonEmptyLine(stderr))
		}
		// Update info after unpacking
		vmInfoFrom, err = retrieveInfo(ctx, vmNameFrom)
		if err != nil {
			return nil, err
		}
	}

	if vmInfoFrom.State == "suspended" {
		return cloneFromSuspended(ctx, vmInfoFrom.Home)
	}

	return cloneFromDefault(ctx, vmInfoFrom.Name)
}

func (vm *VM) Start(ctx context.Context) error {
	if !vm.delayedIsolation {
		if err := vm.isolate(ctx); err != nil {
			return err
		}
	}

	_, stderr, err := Prlctl(ctx, "start", vm.Ident())
	if err != nil {
		return fmt.Errorf("%w: failed to start VM %q: %q", ErrVMFailed, vm.Ident(), firstNonEmptyLine(stderr))
	}

	if vm.shouldRenewDHCP {
		if err := vm.renewDHCP(ctx); err != nil {
			return err
		}
	}

	if vm.delayedIsolation {
		if err := vm.isolate(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Returns an identifier suitable for use in Parallels CLI commands.
func (vm *VM) Ident() string {
	if vm.uuid != "" {
		return vm.uuid
	}

	return vm.name
}

func (vm *VM) isolate(ctx context.Context) error {
	// Ensure that the VM is isolated[1] from the host (e.g. shared folders, clipboard, etc.)
	// nolint:lll // https://github.com/walle/lll/issues/12
	// [1]: https://download.parallels.com/desktop/v14/docs/en_US/Parallels%20Desktop%20Pro%20Edition%20Command-Line%20Reference/43645.htm
	_, stderr, err := Prlctl(ctx, "set", vm.Ident(), "--isolate-vm", "on")
	if err != nil {
		return fmt.Errorf("%w: failed to isolate VM %q: %q", ErrVMFailed, vm.Ident(), firstNonEmptyLine(stderr))
	}

	return nil
}

func (vm *VM) renewDHCP(ctx context.Context) error {
	// Poke DHCP to renew a lease because suspended on another host VMs don't yet have IPs on the current host
	_, stderr, err := Prlctl(ctx, "set", vm.Ident(), "--dhcp", "yes", "--dhcp6", "yes")
	if err != nil {
		return fmt.Errorf("%w: failed to poke DHCP for VM %q: %q", ErrVMFailed, vm.Ident(), firstNonEmptyLine(stderr))
	}

	return nil
}

func (vm *VM) Close() error {
	ctx := context.Background()

	_, stderr, err := Prlctl(ctx, "stop", vm.Ident(), "--kill")
	if err != nil {
		return fmt.Errorf("%w: failed to stop VM %q: %q", ErrVMFailed, vm.Ident(), firstNonEmptyLine(stderr))
	}

	_, stderr, err = Prlctl(ctx, "delete", vm.Ident())
	if err != nil {
		return fmt.Errorf("%w: failed to delete VM %q: %q", ErrVMFailed, vm.Ident(), firstNonEmptyLine(stderr))
	}

	return nil
}

func retrieveInfo(ctx context.Context, ident string) (*VirtualMachineInfo, error) {
	stdout, stderr, err := Prlctl(ctx, "list", "--info", "--json", ident)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get info for VM with %q UUID or name: %q",
			ErrVMFailed, ident, firstNonEmptyLine(stderr))
	}

	var vmInfos []VirtualMachineInfo

	if err := json.Unmarshal([]byte(stdout), &vmInfos); err != nil {
		return nil, err
	}

	switch len(vmInfos) {
	case 0:
		return nil, fmt.Errorf("%w: failed to find VM with %q UUID or name", ErrVMFailed, ident)
	case 1:
		return &vmInfos[0], nil
	default:
		return nil, fmt.Errorf("%w: more than one VM found with %q UUID or name", ErrVMFailed, ident)
	}
}

func (vm *VM) RetrieveIP(ctx context.Context) (string, error) {
	vmInfo, err := retrieveInfo(ctx, vm.Ident())
	if err != nil {
		return "", err
	}

	mac, err := hex.DecodeString(vmInfo.Hardware.Net0.MAC)
	if err != nil {
		return "", fmt.Errorf("%w: failed to decode MAC %q for VM %q: %v",
			ErrVMFailed, vmInfo.Hardware.Net0.MAC, vm.Ident(), err)
	}

	snooper := &DHCPSnooper{}
	lease, err := snooper.FindNewestLease(mac)
	if err != nil {
		return "", err
	}

	return lease.IP, nil
}

func firstNonEmptyLine(lines string) string {
	for _, line := range strings.Split(lines, "\n") {
		if line != "" {
			return line
		}
	}

	return ""
}
