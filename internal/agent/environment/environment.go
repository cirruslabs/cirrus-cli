package environment

import "strings"

type Environment struct {
	env             map[string]string
	sensitiveValues []string
}

func New(items map[string]string) *Environment {
	env := NewEmpty()

	env.Merge(items, false)

	return env
}

func NewEmpty() *Environment {
	return &Environment{
		env:             map[string]string{},
		sensitiveValues: []string{},
	}
}

func (env *Environment) Get(key string) string {
	return env.env[key]
}

func (env *Environment) Lookup(key string) (string, bool) {
	value, ok := env.env[key]

	return value, ok
}

func (env *Environment) Set(key string, value string) {
	env.env[key] = value

	if isWellKnownSensitive(key) {
		env.AddSensitiveValues(value)
	}
}

func (env *Environment) Merge(otherEnv map[string]string, isSensitive bool) {
	if len(otherEnv) == 0 {
		return
	}

	// Accommodate new environment variables
	for key, value := range otherEnv {
		env.env[key] = value
	}

	// Do one more expansion pass since we've introduced
	// new and potentially unexpanded variables
	env.env = ExpandEnvironmentRecursively(env.env)

	for key, value := range otherEnv {
		if isSensitive || isWellKnownSensitive(key) {
			env.AddSensitiveValues(value)
		}
	}
}

func (env *Environment) Items() map[string]string {
	return env.env
}

func (env *Environment) AddSensitiveValues(sensitiveValues ...string) {
	for _, sensitiveValue := range sensitiveValues {
		// Nothing to mask
		if sensitiveValue == "" {
			continue
		}

		env.sensitiveValues = append(env.sensitiveValues, sensitiveValue)
	}
}

func (env *Environment) SensitiveValues() []string {
	return env.sensitiveValues
}

func isWellKnownSensitive(key string) bool {
	return strings.HasSuffix(key, "_PASSWORD") ||
		strings.HasSuffix(key, "_SECRET") ||
		strings.HasSuffix(key, "_TOKEN") ||
		strings.HasSuffix(key, "_ACCESS_KEY") ||
		strings.HasSuffix(key, "_SECRET_KEY")
}
