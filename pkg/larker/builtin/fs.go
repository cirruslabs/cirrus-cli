package builtin

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"go.starlark.net/starlark"
	"os"
)

type ResolveFSFunc func(string) (fs.FileSystem, string, error)

func FS(ctx context.Context, resolveFS ResolveFSFunc) starlark.StringDict {
	return starlark.StringDict{
		"exists":  exists(ctx, resolveFS),
		"read":    read(ctx, resolveFS),
		"readdir": readdir(ctx, resolveFS),
		"isdir":   isdir(ctx, resolveFS),
	}
}

func exists(ctx context.Context, resolveFS ResolveFSFunc) starlark.Value {
	const funcName = "exists"

	return starlark.NewBuiltin(funcName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(funcName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}

		resolvedFS, path, err := resolveFS(path)
		if err != nil {
			return nil, err
		}

		_, err = resolvedFS.Stat(ctx, path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return starlark.Bool(false), nil
			}

			return nil, err
		}

		return starlark.Bool(true), nil
	})
}

func read(ctx context.Context, resolveFS ResolveFSFunc) starlark.Value {
	const funcName = "read"

	return starlark.NewBuiltin(funcName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(funcName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}

		resolvedFS, path, err := resolveFS(path)
		if err != nil {
			return nil, err
		}

		fileBytes, err := resolvedFS.Get(ctx, path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return starlark.None, nil
			}

			return nil, err
		}

		return starlark.String(fileBytes), nil
	})
}

func readdir(ctx context.Context, resolveFS ResolveFSFunc) starlark.Value {
	const funcName = "readdir"

	return starlark.NewBuiltin(funcName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(funcName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}

		resolvedFS, path, err := resolveFS(path)
		if err != nil {
			return nil, err
		}

		entries, err := resolvedFS.ReadDir(ctx, path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return starlark.None, nil
			}

			return nil, err
		}

		var starlarkEntries []starlark.Value
		for _, entry := range entries {
			starlarkEntries = append(starlarkEntries, starlark.String(entry))
		}

		return starlark.NewList(starlarkEntries), nil
	})
}

func isdir(ctx context.Context, resolveFS ResolveFSFunc) starlark.Value {
	const funcName = "isdir"

	return starlark.NewBuiltin(funcName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(funcName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}

		resolvedFS, path, err := resolveFS(path)
		if err != nil {
			return nil, err
		}

		fileInfo, err := resolvedFS.Stat(ctx, path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return starlark.False, nil
			}

			return nil, err
		}

		return starlark.Bool(fileInfo.IsDir), nil
	})
}
