package larker

import "go.starlark.net/starlark"

type Context struct{}

func (context *Context) String() string {
	return "context"
}

func (context *Context) Type() string {
	return "context"
}

func (context *Context) Freeze() {}

func (context *Context) Truth() starlark.Bool {
	return true
}

func (context *Context) Hash() (uint32, error) {
	return 0, nil
}
