package vetu

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/worker/resourcemodifier"
	"github.com/cirruslabs/echelon"
	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"strings"
	"sync"
)

type VM struct {
	ident                 string
	env                   map[string]string
	resourceModifier      *resourcemodifier.Modifier
	backgroundCtxWithSpan context.Context
	runningVMCtx          context.Context
	runningVMCtxCancel    context.CancelFunc
	wg                    sync.WaitGroup
	errChan               chan error
}

func NewVMClonedFrom(
	ctx context.Context,
	from string,
	to string,
	lazyPull bool,
	env map[string]string,
	resourceModifier *resourcemodifier.Modifier,
	logger *echelon.Logger,
) (*VM, error) {
	backgroundCtxWithSpan := trace.ContextWithSpan(context.Background(), trace.SpanFromContext(ctx))

	runningVMCtx, runningVMCtxCancel := context.WithCancel(backgroundCtxWithSpan)

	vm := &VM{
		ident:                 to,
		env:                   env,
		resourceModifier:      resourceModifier,
		backgroundCtxWithSpan: backgroundCtxWithSpan,
		runningVMCtx:          runningVMCtx,
		runningVMCtxCancel:    runningVMCtxCancel,
		errChan:               make(chan error, 1),
	}

	pullLogger := logger.Scoped("pull virtual machine")
	if !lazyPull {
		pullLogger.Infof("Pulling virtual machine %s...", from)

		if _, _, err := CmdWithLogger(ctx, vm.env, pullLogger, "pull", from); err != nil {
			pullLogger.Errorf("Ignoring pull failure: %v", err)
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

	cloneLogger.Finish(true)
	return vm, nil
}

func (vm *VM) Ident() string {
	return vm.ident
}

func (vm *VM) Configure(
	ctx context.Context,
	cpu uint32,
	memory uint32,
	diskSize uint32,
	logger *echelon.Logger,
) error {
	configureLogger := logger.Scoped("configure virtual machine")
	configureLogger.Infof("Configuring virtual machine %s...", vm.ident)

	if cpu != 0 {
		cpuStr := strconv.FormatUint(uint64(cpu), 10)

		_, _, err := CmdWithLogger(ctx, vm.env, configureLogger, "set", vm.ident, "--cpu", cpuStr)
		if err != nil {
			configureLogger.Finish(false)
			return err
		}
	}

	if memory != 0 {
		memoryStr := strconv.FormatUint(uint64(memory), 10)

		_, _, err := CmdWithLogger(ctx, vm.env, configureLogger, "set", vm.ident, "--memory", memoryStr)
		if err != nil {
			configureLogger.Finish(false)
			return err
		}
	}

	if diskSize != 0 {
		diskSizeStr := strconv.FormatUint(uint64(diskSize), 10)

		_, _, err := CmdWithLogger(ctx, vm.env, configureLogger, "set", vm.ident, "--disk-size", diskSizeStr)
		if err != nil {
			configureLogger.Finish(false)
			return err
		}
	}

	configureLogger.Finish(true)

	return nil
}

func (vm *VM) Start(
	ctx context.Context,
	bridgedInterface string,
	hostNetworking bool,
) {
	vm.wg.Add(1)

	go func() {
		defer vm.wg.Done()

		var args []string

		if bridgedInterface != "" {
			args = append(args, "--net-bridged", bridgedInterface)
		} else if hostNetworking {
			args = append(args, "--net-host")
		}

		// Apply "run" resource modifier
		if vm.resourceModifier != nil {
			args = append(args, vm.resourceModifier.Append.Run...)
		}

		args = append(args, vm.ident)

		stdout, stderr, err := Cmd(vm.runningVMCtx, vm.env, "run", args...)
		if localHub := sentry.GetHubFromContext(ctx); localHub != nil {
			localHub.AddBreadcrumb(&sentry.Breadcrumb{
				Message: "\"vetu run\" finished",
				Data: map[string]interface{}{
					"err":    err,
					"stdout": stdout,
					"stderr": stderr,
				},
			}, nil)
		}
		vm.errChan <- err
	}()
}

func (vm *VM) ErrChan() chan error {
	return vm.errChan
}

func (vm *VM) RetrieveIP(ctx context.Context) (string, error) {
	// wait 30 seconds since usually a VM boots in 15
	stdout, _, err := Cmd(ctx, vm.env, "ip", "--wait", "30", vm.ident)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout), nil
}

func (vm *VM) Close() error {
	// Try to gracefully terminate the VM
	//nolint:dogsled // not interested in the output for now
	_, _, _ = Cmd(vm.backgroundCtxWithSpan, vm.env, "stop", "--timeout", "5", vm.ident)

	vm.runningVMCtxCancel()
	vm.wg.Wait()

	_, _, err := Cmd(vm.backgroundCtxWithSpan, vm.env, "delete", vm.ident)

	return err
}
