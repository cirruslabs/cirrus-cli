package loader

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/certifi/gocertifi"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/builtin"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/loader/git"
	"github.com/qri-io/starlib/encoding/base64"
	"github.com/qri-io/starlib/encoding/yaml"
	"github.com/qri-io/starlib/hash"
	"github.com/qri-io/starlib/http"
	"github.com/qri-io/starlib/re"
	"github.com/qri-io/starlib/zipfile"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkjson"
	"go.starlark.net/starlarkstruct"
	gohttp "net/http"
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
	ctx           context.Context
	cache         map[string]*CacheEntry
	fs            fs.FileSystem
	env           map[string]string
	affectedFiles []string
}

func NewLoader(ctx context.Context, fs fs.FileSystem, env map[string]string, affectedFiles []string) *Loader {
	return &Loader{
		ctx:           ctx,
		cache:         make(map[string]*CacheEntry),
		fs:            fs,
		env:           env,
		affectedFiles: affectedFiles,
	}
}

func (loader *Loader) Retrieve(module string) ([]byte, error) {
	gitLocator := git.Parse(module)
	if gitLocator != nil {
		return git.Retrieve(loader.ctx, gitLocator)
	}

	return loader.fs.Get(loader.ctx, module)
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

		// A special case for loading Cirrus-provided builtins (e.g. load("cirrus", "fs"))
		if module == "cirrus" {
			return loader.loadCirrusModule()
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

func (loader *Loader) loadCirrusModule() (starlark.StringDict, error) {
	result := make(starlark.StringDict)

	starlarkEnv := starlark.NewDict(len(loader.env))
	for key, value := range loader.env {
		if err := starlarkEnv.SetKey(starlark.String(key), starlark.String(value)); err != nil {
			return nil, err
		}
	}
	result["env"] = starlarkEnv

	result["changes_include"] = generateChangesIncludeBuiltin(loader.affectedFiles)

	result["fs"] = &starlarkstruct.Module{
		Name:    "fs",
		Members: builtin.FS(loader.ctx, loader.fs),
	}

	certPool, err := gocertifi.CACerts()
	if err != nil {
		http.Client = &gohttp.Client{
			Transport: &gohttp.Transport{
				TLSClientConfig: &tls.Config{RootCAs: certPool},
			},
		}
	}

	httpModule, err := http.LoadModule()
	if err != nil {
		return nil, err
	}
	result["http"] = httpModule["http"]

	hashModule, err := hash.LoadModule()
	if err != nil {
		return nil, err
	}
	result["hash"] = hashModule["hash"]

	base64Module, err := base64.LoadModule()
	if err != nil {
		return nil, err
	}
	result["base64"] = base64Module["base64"]

	// Work around https://github.com/qri-io/starlib/pull/70
	fixedJSONModule := &starlarkstruct.Module{
		Name: "json",
		Members: starlark.StringDict{
			"loads": starlarkjson.Module.Members["decode"],
			"dumps": starlarkjson.Module.Members["encode"],
		},
	}
	result["json"] = fixedJSONModule

	yamlModule, err := yaml.LoadModule()
	if err != nil {
		return nil, err
	}
	result["yaml"] = yamlModule["yaml"]

	reModule, err := re.LoadModule()
	if err != nil {
		return nil, err
	}
	result["re"] = reModule["re"]

	zipfileModule, err := zipfile.LoadModule()
	if err != nil {
		return nil, err
	}
	result["zipfile"] = &starlarkstruct.Module{
		Name:    "zipfile",
		Members: zipfileModule,
	}

	return result, nil
}
