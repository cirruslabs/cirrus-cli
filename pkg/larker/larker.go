package larker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/dummy"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/loader"
	"github.com/cirruslabs/cirrus-cli/pkg/yamlhelper"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"time"
)

var (
	ErrLoadFailed           = errors.New("load failed")
	ErrExecFailed           = errors.New("exec failed")
	ErrMainFailed           = errors.New("failed to call main")
	ErrHookFailed           = errors.New("failed to call hook")
	ErrMainUnexpectedResult = errors.New("main returned unexpected result")
	ErrSanity               = errors.New("sanity check failed")
)

type Larker struct {
	fs  fs.FileSystem
	env map[string]string
}

type HookResult struct {
	ErrorMessage  string
	OutputLogs    []byte
	DurationNanos int64
	Result        interface{}
}

func New(opts ...Option) *Larker {
	lrk := &Larker{
		fs:  dummy.New(),
		env: make(map[string]string),
	}

	// weird global init by Starlark
	// we need floats at least for configuring CPUs for containers
	resolve.AllowFloat = true

	// Apply options
	for _, opt := range opts {
		opt(lrk)
	}

	return lrk
}

func (larker *Larker) Main(ctx context.Context, source string) (string, error) {
	discard := func(thread *starlark.Thread, msg string) {}

	thread := &starlark.Thread{
		Load:  loader.NewLoader(ctx, larker.fs, larker.env).LoadFunc(),
		Print: discard,
	}

	resCh := make(chan starlark.Value)
	errCh := make(chan error)

	go func() {
		// Execute the source code for the main() to be visible
		globals, err := starlark.ExecFile(thread, ".cirrus.star", source, nil)
		if err != nil {
			errCh <- fmt.Errorf("%w: %v", ErrLoadFailed, err)
			return
		}

		// Retrieve main()
		main, ok := globals["main"]
		if !ok {
			errCh <- fmt.Errorf("%w: main() not found", ErrMainFailed)
			return
		}

		// Ensure that main() is a function
		if _, ok := main.(*starlark.Function); !ok {
			errCh <- fmt.Errorf("%w: main is not a function", ErrMainFailed)
			return
		}

		// Prepare a context to pass to main() as it's first argument
		mainCtx := &Context{}

		mainResult, err := starlark.Call(thread, main, starlark.Tuple{mainCtx}, nil)
		if err != nil {
			errCh <- fmt.Errorf("%w: %v", ErrExecFailed, err)
			return
		}

		resCh <- mainResult
	}()

	var mainResult starlark.Value

	select {
	case mainResult = <-resCh:
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		thread.Cancel(ctx.Err().Error())
		return "", ctx.Err()
	}

	// main() should return a list of tasks
	starlarkList, ok := mainResult.(*starlark.List)
	if !ok {
		return "", fmt.Errorf("%w: result is not a list", ErrMainUnexpectedResult)
	}

	tasksNode := convertTasks(starlarkList)
	if tasksNode == nil {
		return "", nil
	}
	formattedYaml, err := yamlhelper.PrettyPrint(tasksNode)
	if err != nil {
		return "", fmt.Errorf("%w: cannot marshal into YAML: %v", ErrMainUnexpectedResult, err)
	}

	return formattedYaml, nil
}

func (larker *Larker) Hook(
	ctx context.Context,
	source string,
	name string,
	arguments []interface{},
) (*HookResult, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: empty hook name specified", ErrSanity)
	}

	if len(arguments) != 1 {
		return nil, fmt.Errorf("%w: hook takes exactly 1 argument, got %d", ErrSanity, len(arguments))
	}

	outputLogsBuffer := &bytes.Buffer{}

	thread := &starlark.Thread{
		Load: loader.NewLoader(ctx, larker.fs, larker.env).LoadFunc(),
		Print: func(thread *starlark.Thread, msg string) {
			_, _ = fmt.Fprintln(outputLogsBuffer, msg)
		},
	}

	resCh := make(chan *HookResult)
	errCh := make(chan error)

	go func() {
		// Execute the source code for the hook to be visible
		globals, err := starlark.ExecFile(thread, ".cirrus.star", source, nil)
		if err != nil {
			errCh <- fmt.Errorf("%w: %v", ErrLoadFailed, err)
			return
		}

		// Retrieve hook
		hook, ok := globals[name]
		if !ok {
			errCh <- fmt.Errorf("%w: %s() not found", ErrHookFailed, name)
			return
		}

		// Ensure that hook is a function
		if _, ok := hook.(*starlark.Function); !ok {
			errCh <- fmt.Errorf("%w: %s is not a function", ErrHookFailed, name)
			return
		}

		hookArgument, err := interfaceAsStarlarkValue(arguments[0])
		if err != nil {
			errCh <- fmt.Errorf("%w: hook's ctx argument should be JSON-compatible: %v", ErrHookFailed, err)
			return
		}

		// Run hook and measure time spent
		//
		// We could've used unix.Getrusage() here instead, however:
		// * it's not clear if we even need such level of precision at the moment
		// * precise time measurement requires:
		//   * usage of the Linux-specific RUSAGE_THREAD flag
		//   * guarding starlark.Call() with runtime.LockOSThread()/runtime.UnlockOSThread()
		hookStartTime := time.Now()

		hookResult, err := starlark.Call(thread, hook, starlark.Tuple{hookArgument}, nil)
		if err != nil {
			errCh <- fmt.Errorf("%w: %v", ErrExecFailed, err)
			return
		}

		durationNanos := time.Since(hookStartTime).Nanoseconds()

		// Convert Starlark-style value to interface{}-style value
		hookResultStarlark, err := starlarkValueAsInterface(hookResult)
		if err != nil {
			errCh <- err
			return
		}

		// All good
		resCh <- &HookResult{
			OutputLogs:    outputLogsBuffer.Bytes(),
			DurationNanos: durationNanos,
			Result:        hookResultStarlark,
		}
	}()

	select {
	case hookResult := <-resCh:
		return hookResult, nil
	case err := <-errCh:
		return &HookResult{ErrorMessage: err.Error()}, err
	case <-ctx.Done():
		thread.Cancel(ctx.Err().Error())
		return nil, ctx.Err()
	}
}
