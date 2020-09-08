package parser

type Option func(*Parser)

func WithEnvironment(environment map[string]string) Option {
	return func(parser *Parser) {
		parser.environment = environment
	}
}
