package parseable

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/lestrrat-go/jsschema"
)

type DefaultParser struct {
	fields            []Field
	collectibleFields []CollectibleField
	collectible       bool
}

func (parser *DefaultParser) OptionalField(name nameable.Nameable, schema *schema.Schema, onFound nodeFunc) {
	parser.fields = append(parser.fields, Field{
		name:    name,
		onFound: onFound,
		schema:  schema,
	})
}

func (parser *DefaultParser) OptionalRepeatableField(name nameable.Nameable, schema *schema.Schema, onFound nodeFunc) {
	parser.fields = append(parser.fields, Field{
		name:       name,
		repeatable: true,
		onFound:    onFound,
		schema:     schema,
	})
}

func (parser *DefaultParser) RequiredField(nameable nameable.Nameable, schema *schema.Schema, onFound nodeFunc) {
	parser.fields = append(parser.fields, Field{
		name:     nameable,
		required: true,
		onFound:  onFound,
		schema:   schema,
	})
}

func (parser *DefaultParser) CollectibleField(name string, schema *schema.Schema, onFound nodeFunc) {
	parser.collectibleFields = append(parser.collectibleFields, CollectibleField{
		Name:    name,
		onFound: onFound,
		Schema:  schema,
	})
}

func (parser *DefaultParser) Collectible() bool {
	return parser.collectible
}

func (parser *DefaultParser) SetCollectible(value bool) {
	parser.collectible = value
}

func (parser *DefaultParser) NormalField(name string) *Field {
	for _, field := range parser.fields {
		if field.name.Matches(name) {
			return &field
		}
	}

	return nil
}
