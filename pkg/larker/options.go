package larker

import (
	"net/http"

	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
)

type Option func(*Larker)

func WithFileSystem(fs fs.FileSystem) Option {
	return func(e *Larker) {
		e.fs = fs
	}
}

func WithEnvironment(env map[string]string) Option {
	return func(e *Larker) {
		e.env = env
	}
}

func WithAffectedFiles(affectedFiles []string) Option {
	return func(e *Larker) {
		e.affectedFiles = affectedFiles
	}
}

func WithTestMode() Option {
	return func(e *Larker) {
		e.isTest = true
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(e *Larker) {
		e.httpClient = httpClient
	}
}
