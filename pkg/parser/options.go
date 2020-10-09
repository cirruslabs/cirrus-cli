package parser

import (
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Option func(*Parser)

func WithEnvironment(environment map[string]string) Option {
	return func(parser *Parser) {
		parser.environment = environment
	}
}

func WithFileSystem(fs fs.FileSystem) Option {
	return func(parser *Parser) {
		parser.fs = fs
	}
}

func WithAffectedFiles(affectedFiles []string) Option {
	return func(parser *Parser) {
		parser.affectedFiles = affectedFiles
	}
}

func WithAdditionalInstances(additionalInstances map[string]protoreflect.MessageDescriptor) Option {
	return func(parser *Parser) {
		parser.additionalInstances = additionalInstances
	}
}
