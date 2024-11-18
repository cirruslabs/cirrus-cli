package node

type Option func(node *config)

func WithoutYAMLNode() Option {
	return func(config *config) {
		config.withoutYAMLNode = true
	}
}
