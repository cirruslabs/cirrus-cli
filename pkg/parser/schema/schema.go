package schema

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	schema "github.com/lestrrat-go/jsschema"
	"regexp"
	"sort"
	"strings"
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

func Boolean(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.PrimitiveTypes{schema.BooleanType},
		Description: description,
	}
}

func Integer(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.PrimitiveTypes{schema.IntegerType},
		Description: description,
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

func StringWithDefaultValue(description string, defaultValue string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.PrimitiveTypes{schema.StringType},
		Description: description,
		Default:     defaultValue,
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

func Enum(items []interface{}, description string) *schema.Schema {
	return &schema.Schema{
		Description:          description,
		Enum:                 items,
		AdditionalItems:      &schema.AdditionalItems{Schema: nil},
		AdditionalProperties: &schema.AdditionalProperties{Schema: nil},
	}
}

func TriggerType() *schema.Schema {
	return Enum([]interface{}{
		"automatic",
		"manual",
	}, "Trigger type")
}

func Platform(description string) *schema.Schema {
	// Prepare a list of platforms
	var platformsLowercased []string
	for _, platformName := range api.Platform_name {
		platformsLowercased = append(platformsLowercased, strings.ToLower(platformName))
	}

	sort.Strings(platformsLowercased)

	var platformsInterfaced []interface{}
	for _, lowercasePlatform := range platformsLowercased {
		platformsInterfaced = append(platformsInterfaced, lowercasePlatform)
	}

	return Enum(platformsInterfaced, description)
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

func Ports() *schema.Schema {
	result := ArrayOf(Port())

	result.Description = "Ports exposed by the container."

	return result
}

func Volumes() *schema.Schema {
	return &schema.Schema{
		Description: "A list of volumes mounted inside of the container.",
		Type:        schema.PrimitiveTypes{schema.ArrayType},
		Items: &schema.ItemSpec{
			TupleMode: true,
			Schemas: schema.SchemaList{
				String("A volume in the format of source:target[:ro]."),
			},
		},
		AdditionalItems: &schema.AdditionalItems{Schema: nil},
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
