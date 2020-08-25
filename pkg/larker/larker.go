package larker

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"go.starlark.net/starlark"
	"gopkg.in/yaml.v2"
)

var (
	ErrLoadFailed           = errors.New("load failed")
	ErrExecFailed           = errors.New("exec failed")
	ErrMainFailed           = errors.New("failed to call main")
	ErrMainUnexpectedResult = errors.New("main returned unexpected result")
)

type Larker struct {
	fs     fs.FileSystem
	loader *Loader
}

func New(opts ...Option) *Larker {
	lrk := &Larker{
		fs: fs.NewDummyFileSystem(),
	}

	// Apply options
	for _, opt := range opts {
		opt(lrk)
	}

	// Some fields can only be set after we apply the options
	lrk.loader = NewLoader(lrk.fs)

	return lrk
}

func (larker *Larker) Main(ctx context.Context, source string) (string, error) {
	discard := func(thread *starlark.Thread, msg string) {}

	thread := &starlark.Thread{
		Load:  larker.loader.LoadFunc(),
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
		return "", ctx.Err()
	}

	// main() should return a list of tasks
	starlarkList, ok := mainResult.(*starlark.List)
	if !ok {
		return "", fmt.Errorf("%w: result is not a list", ErrMainUnexpectedResult)
	}

	// Recurse into starlarkList and convert starlark.List's to []interface{}'s and
	// starlark.Dict's to yaml.MapSlice's to make them YAML-serializable
	yamlList := convertList(starlarkList)

	// Adapt a list of tasks to a YAML configuration format that expects a map on it's outer layer
	var serializableMainResult yaml.MapSlice
	for _, listItem := range yamlList {
		serializableMainResult = append(serializableMainResult, yaml.MapItem{
			Key:   "task",
			Value: listItem,
		})
	}

	// Produce the YAML configuration
	yamlBytes, err := yaml.Marshal(&serializableMainResult)
	if err != nil {
		return "", fmt.Errorf("%w: cannot marshal into YAML: %v", ErrMainUnexpectedResult, err)
	}

	return string(yamlBytes), nil
}
