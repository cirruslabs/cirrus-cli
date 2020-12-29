package parallels

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"strings"
)

var ErrVMFailed = errors.New("Parallels VM operation failed")

type VM struct {
	name string
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
	Hardware HardwareInfo
}

func NewVMClonedFrom(ctx context.Context, vmNameFrom string) (*VM, error) {
	vm := &VM{
		name: "cirrus-" + uuid.New().String(),
	}

	_, stderr, err := Prlctl(ctx, "clone", vmNameFrom, "--name", vm.name)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to clone VM %q: %q", ErrVMFailed, vm.name, firstLine(stderr))
	}

	_, stderr, err = Prlctl(ctx, "start", vm.name)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start VM %q: %q", ErrVMFailed, vm.name, firstLine(stderr))
	}

	return vm, nil
}

func (vm *VM) Close() error {
	ctx := context.Background()

	_, stderr, err := Prlctl(ctx, "stop", vm.name, "--kill")
	if err != nil {
		return fmt.Errorf("%w: failed to stop VM %q: %q", ErrVMFailed, vm.name, firstLine(stderr))
	}

	_, stderr, err = Prlctl(ctx, "delete", vm.name)
	if err != nil {
		return fmt.Errorf("%w: failed to delete VM %q: %q", ErrVMFailed, vm.name, firstLine(stderr))
	}

	return nil
}

func (vm *VM) retrieveInfo(ctx context.Context) (*VirtualMachineInfo, error) {
	stdout, stderr, err := Prlctl(ctx, "list", "--info", "--json")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get VM %q info: %q", ErrVMFailed, vm.name, firstLine(stderr))
	}

	var vmInfos []VirtualMachineInfo

	if err := json.Unmarshal([]byte(stdout), &vmInfos); err != nil {
		return nil, err
	}

	for _, vmInfo := range vmInfos {
		if vmInfo.Name == vm.name {
			return &vmInfo, nil
		}
	}

	return nil, fmt.Errorf("%w: failed to find VM %q", ErrVMFailed, vm.name)
}

func (vm *VM) RetrieveIP(ctx context.Context) (string, error) {
	vmInfo, err := vm.retrieveInfo(ctx)
	if err != nil {
		return "", err
	}

	mac, err := hex.DecodeString(vmInfo.Hardware.Net0.MAC)
	if err != nil {
		return "", fmt.Errorf("%w: failed to decode MAC %q for VM %q: %v",
			ErrVMFailed, vmInfo.Hardware.Net0.MAC, vm.name, err)
	}

	snooper := &DHCPSnooper{}
	lease, err := snooper.FindNewestLease(mac)
	if err != nil {
		return "", err
	}

	return lease.IP, nil
}

func firstLine(lines string) string {
	return strings.Split(lines, "\n")[0]
}
