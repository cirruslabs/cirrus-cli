package parseable

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/lestrrat-go/jsschema"
)

type DefaultParser struct {
	fields            []Field
	collectibleFields []CollectibleField
}

func (collectible *DefaultParser) OptionalField(name nameable.Nameable, schema *schema.Schema, onFound nodeFunc) {
	collectible.fields = append(collectible.fields, Field{
		name:    name,
		onFound: onFound,
		schema:  schema,
	})
}

func (collectible *DefaultParser) RequiredField(nameable nameable.Nameable, schema *schema.Schema, onFound nodeFunc) {
	collectible.fields = append(collectible.fields, Field{
		name:     nameable,
		required: true,
		onFound:  onFound,
		schema:   schema,
	})
}

func (collectible *DefaultParser) CollectibleField(name string, schema *schema.Schema, onFound nodeFunc) {
	collectible.collectibleFields = append(collectible.collectibleFields, CollectibleField{
		name:    name,
		onFound: onFound,
		schema:  schema,
	})
}
