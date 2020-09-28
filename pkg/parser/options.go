package parser

import (
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Option func(*Parser)

func WithEnvironment(environment map[string]string) Option {
	return func(parser *Parser) {
		parser.environment = environment
	}
}

func WithAdditionalInstances(additionalInstances map[string]protoreflect.MessageDescriptor) Option {
	return func(parser *Parser) {
		parser.additionalInstances = additionalInstances
	}
}
