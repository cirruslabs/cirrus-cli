package expander

type Option func(expander *expander)

func WithMaxExpansionIterations(maxExpansionIterations int) Option {
	return func(parser *expander) {
		parser.maxExpansionIterations = maxExpansionIterations
	}
}

func WithPrecise() Option {
	return func(parser *expander) {
		parser.precise = true
	}
}
