package schema

import (
	schema "github.com/lestrrat-go/jsschema"
	"regexp"
)

var (
	TodoSchema = &schema.Schema{}
)

func Map(description string) *schema.Schema {
	if description == "" {
		description = "Map represented as an object."
	}

	return &schema.Schema{
		Type:        schema.PrimitiveTypes{schema.ObjectType},
		Description: description,
		PatternProperties: map[*regexp.Regexp]*schema.Schema{
			regexp.MustCompile(".*"): {
				Type:                 schema.PrimitiveTypes{schema.StringType},
				AdditionalProperties: &schema.AdditionalProperties{Schema: nil},
			},
		},
		AdditionalProperties: &schema.AdditionalProperties{Schema: nil},
	}
}

func Condition(description string) *schema.Schema {
	var fullDescription string

	if description == "" {
		fullDescription = "Boolean expression that can use environment variables."
	} else {
		fullDescription = "Boolean expression. " + description
	}

	return &schema.Schema{
		Type:        schema.PrimitiveTypes{schema.StringType},
		Description: fullDescription,
	}
}

func Number(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.PrimitiveTypes{schema.NumberType},
		Description: description,
	}
}

func String(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.PrimitiveTypes{schema.StringType},
		Description: description,
	}
}

func Memory() *schema.Schema {
	return &schema.Schema{
		Type:    schema.PrimitiveTypes{schema.StringType},
		Pattern: regexp.MustCompile(`\d+(G|Mb)?`),
	}
}

func StringOrListOfStrings(description string) *schema.Schema {
	return &schema.Schema{
		Description: description,
		AnyOf: schema.SchemaList{
			String(""),
			ArrayOf(String("")),
		},
		AdditionalItems:      &schema.AdditionalItems{Schema: nil},
		AdditionalProperties: &schema.AdditionalProperties{Schema: nil},
	}
}

func TriggerType() *schema.Schema {
	return &schema.Schema{
		Description: "Trigger type",
		Enum: []interface{}{
			"automatic",
			"manual",
		},
		AdditionalItems:      &schema.AdditionalItems{Schema: nil},
		AdditionalProperties: &schema.AdditionalProperties{Schema: nil},
	}
}

func Script(description string) *schema.Schema {
	return &schema.Schema{
		Description: description,
		AnyOf: schema.SchemaList{
			String(""),
			ArrayOf(String("")),
		},
		AdditionalItems:      &schema.AdditionalItems{Schema: nil},
		AdditionalProperties: &schema.AdditionalProperties{Schema: nil},
	}
}

func Port() *schema.Schema {
	return &schema.Schema{
		Description: "Port exposed by the container.",
		AnyOf: schema.SchemaList{
			Number(""),
			String(""),
		},
		AdditionalItems:      &schema.AdditionalItems{Schema: nil},
		AdditionalProperties: &schema.AdditionalProperties{Schema: nil},
	}
}

func ArrayOf(arrayItemSchema *schema.Schema) *schema.Schema {
	return &schema.Schema{
		Type: schema.PrimitiveTypes{schema.ArrayType},
		Items: &schema.ItemSpec{
			TupleMode: true,
			Schemas: schema.SchemaList{
				arrayItemSchema,
			},
		},
		AdditionalItems: &schema.AdditionalItems{Schema: nil},
	}
}
