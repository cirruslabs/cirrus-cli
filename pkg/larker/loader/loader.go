package loader

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/loader/git"
	"go.starlark.net/starlark"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrCycle           = errors.New("import cycle detected")
	ErrRetrievalFailed = errors.New("failed to retrieve module")
)

type CacheEntry struct {
	globals starlark.StringDict
	err     error
}

type Loader struct {
	ctx   context.Context
	cache map[string]*CacheEntry
	fs    fs.FileSystem
}

func NewLoader(ctx context.Context, fs fs.FileSystem) *Loader {
	return &Loader{
		ctx:   ctx,
		cache: make(map[string]*CacheEntry),
		fs:    fs,
	}
}

func (loader *Loader) Retrieve(module string) ([]byte, error) {
	gitLocator := git.Parse(module)
	if gitLocator != nil {
		return git.Retrieve(loader.ctx, gitLocator)
	}

	return loader.fs.Get(module)
}

func (loader *Loader) LoadFunc() func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	return func(thread *starlark.Thread, module string) (starlark.StringDict, error) {
		// Lookup cache
		entry, ok := loader.cache[module]
		if ok {
			// Even through we've found the requested module in the cache,
			// a canary might indicate that it's still being loaded, which
			// means we've hit an import cycle
			if entry == nil {
				return nil, ErrCycle
			}

			// Return cached results
			return entry.globals, entry.err
		}

		// Retrieve module source code
		source, err := loader.Retrieve(module)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) || errors.Is(err, git.ErrFileNotFound) {
				var hint string

				if strings.Contains(module, ".start") {
					hint = ", perhaps you've meant the .star extension instead of the .start?"
				}

				return nil, fmt.Errorf("%w: module '%s' not found%s", ErrRetrievalFailed, module, hint)
			}

			return nil, err
		}

		// Place a canary to indicate the commencing load and detect cycles
		loader.cache[module] = nil

		// Load the module and cache results
		globals, err := starlark.ExecFile(thread, filepath.Base(module), source, nil)

		loader.cache[module] = &CacheEntry{
			globals: globals,
			err:     err,
		}

		return globals, err
	}
}
