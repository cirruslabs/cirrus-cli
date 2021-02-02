package larker

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/dummy"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/loader"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/utils"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"gopkg.in/yaml.v3"
	"strings"
)

var (
	ErrLoadFailed           = errors.New("load failed")
	ErrExecFailed           = errors.New("exec failed")
	ErrMainFailed           = errors.New("failed to call main")
	ErrMainUnexpectedResult = errors.New("main returned unexpected result")
)

const DefaultYamlMarshalIndent = 2

type Larker struct {
	fs  fs.FileSystem
	env map[string]string
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

	if yamlList == nil || len(yamlList.Content) == 0 {
		return "", nil
	}

	// Adapt a list of tasks to a YAML configuration format that expects a map on it's outer layer
	var serializableMainResult []*yaml.Node
	for _, listItem := range yamlList.Content {
		serializableMainResult = append(serializableMainResult, utils.NewStringNode("task"))
		serializableMainResult = append(serializableMainResult, listItem)
	}

	builder := &strings.Builder{}
	encoder := yaml.NewEncoder(builder)
	encoder.SetIndent(DefaultYamlMarshalIndent)
	err := encoder.Encode(utils.NewMapNode(serializableMainResult))
	if err != nil {
		return "", fmt.Errorf("%w: cannot marshal into YAML: %v", ErrMainUnexpectedResult, err)
	}
	err = encoder.Close()
	if err != nil {
		return "", fmt.Errorf("%w: cannot finish marshaling into YAML: %v", ErrMainUnexpectedResult, err)
	}

	return builder.String(), nil
}
