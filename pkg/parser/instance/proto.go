package instance

import (
	"github.com/cirruslabs/cirrus-cli/pkg/parser/nameable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/node"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/parseable"
	"github.com/cirruslabs/cirrus-cli/pkg/parser/schema"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"strconv"
	"strings"
)

type ProtoInstance struct {
	proto *dynamicpb.Message

	parseable.DefaultParser
}

//nolint:gocognit // it's a parser, there is a lot of boilerplate
func NewProtoParser(desc protoreflect.MessageDescriptor, mergedEnv map[string]string) *ProtoInstance {
	instance := &ProtoInstance{
		proto: dynamicpb.NewMessage(desc),
	}

	fields := desc.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		fieldName := string(field.Name())
		switch field.Kind() {
		case protoreflect.MessageKind:
			instance.OptionalField(nameable.NewSimpleNameable(fieldName), schema.TodoSchema, func(node *node.Node) error {
				if field.IsMap() {
					fieldInstance := instance.proto.NewField(field)
					mapping, err := node.GetStringMapping()
					if err != nil {
						return err
					}
					for key, value := range mapping {
						fieldInstance.Map().Set(
							protoreflect.ValueOfString(key).MapKey(),
							protoreflect.ValueOfString(value),
						)
					}
					instance.proto.Set(field, fieldInstance)
					return nil
				} else if field.IsList() {
					fieldInstance := instance.proto.NewField(field)
					for _, child := range node.Children {
						childParser := NewProtoParser(field.Message(), mergedEnv)
						parserChild, err := childParser.Parse(child)
						if err != nil {
							return err
						}
						fieldInstance.List().Append(protoreflect.ValueOfMessage(parserChild))
					}
					instance.proto.Set(field, fieldInstance)
					return nil
				} else {
					childParser := NewProtoParser(field.Message(), mergedEnv)
					parserChild, err := childParser.Parse(node)
					if err != nil {
						return err
					}
					instance.proto.Set(field, protoreflect.ValueOfMessage(parserChild))
					return nil
				}
			})
		case protoreflect.EnumKind:
			instance.OptionalField(nameable.NewSimpleNameable(fieldName), schema.TodoSchema, func(node *node.Node) error {
				value, err := node.GetExpandedStringValue(mergedEnv)
				if err != nil {
					return err
				}
				enumValueDescriptor := field.Enum().Values().ByName(protoreflect.Name(strings.ToUpper(value)))
				instance.proto.Set(field, protoreflect.ValueOfEnum(enumValueDescriptor.Number()))
				return nil
			})
		case protoreflect.StringKind:
			instance.OptionalField(nameable.NewSimpleNameable(fieldName), schema.TodoSchema, func(node *node.Node) error {
				value, err := node.GetExpandedStringValue(mergedEnv)
				if err != nil {
					return err
				}
				instance.proto.Set(field, protoreflect.ValueOfString(value))
				return nil
			})
		case protoreflect.Int64Kind, protoreflect.Sint64Kind,
			protoreflect.Fixed64Kind, protoreflect.Sfixed64Kind:
			instance.OptionalField(nameable.NewSimpleNameable(fieldName), schema.TodoSchema, func(node *node.Node) error {
				value, err := node.GetExpandedStringValue(mergedEnv)
				if err != nil {
					return err
				}
				var parsedValue int64
				if strings.EqualFold(fieldName, "memory") {
					parsedValue, err = ParseMegaBytes(value)
				} else {
					parsedValue, err = strconv.ParseInt(value, 10, 64)
				}
				if err != nil {
					return err
				}
				instance.proto.Set(field, protoreflect.ValueOfInt64(parsedValue))
				return nil
			})
		case protoreflect.Uint64Kind:
			instance.OptionalField(nameable.NewSimpleNameable(fieldName), schema.TodoSchema, func(node *node.Node) error {
				value, err := node.GetExpandedStringValue(mergedEnv)
				if err != nil {
					return err
				}
				var parsedValue int64
				if strings.EqualFold(fieldName, "memory") {
					parsedValue, err = ParseMegaBytes(value)
				} else {
					parsedValue, err = strconv.ParseInt(value, 10, 64)
				}
				if err != nil {
					return err
				}
				instance.proto.Set(field, protoreflect.ValueOfUint64(uint64(parsedValue)))
				return nil
			})
		case protoreflect.Int32Kind, protoreflect.Sint32Kind,
			protoreflect.Fixed32Kind, protoreflect.Sfixed32Kind:
			instance.OptionalField(nameable.NewSimpleNameable(fieldName), schema.TodoSchema, func(node *node.Node) error {
				value, err := node.GetExpandedStringValue(mergedEnv)
				if err != nil {
					return err
				}
				parsedValue, err := strconv.ParseInt(value, 10, 32)
				if strings.EqualFold(fieldName, "memory") {
					parsedValue, err = ParseMegaBytes(value)
				}
				if err != nil {
					return err
				}
				instance.proto.Set(field, protoreflect.ValueOfInt32(int32(parsedValue)))
				return nil
			})
		case protoreflect.Uint32Kind:
			instance.OptionalField(nameable.NewSimpleNameable(fieldName), schema.TodoSchema, func(node *node.Node) error {
				value, err := node.GetExpandedStringValue(mergedEnv)
				if err != nil {
					return err
				}
				parsedValue, err := strconv.ParseInt(value, 10, 32)
				if strings.EqualFold(fieldName, "memory") {
					parsedValue, err = ParseMegaBytes(value)
				}
				if err != nil {
					return err
				}
				instance.proto.Set(field, protoreflect.ValueOfUint32(uint32(parsedValue)))
				return nil
			})
		case protoreflect.BoolKind:
			instance.OptionalField(nameable.NewSimpleNameable(fieldName), schema.TodoSchema, func(node *node.Node) error {
				evaluation, err := node.GetBoolValue(mergedEnv)
				if err != nil {
					return err
				}
				instance.proto.Set(field, protoreflect.ValueOfBool(evaluation))
				return nil
			})
		case protoreflect.FloatKind:
			instance.OptionalField(nameable.NewSimpleNameable(fieldName), schema.TodoSchema, func(node *node.Node) error {
				value, err := node.GetExpandedStringValue(mergedEnv)
				if err != nil {
					return err
				}
				parsedValue, err := strconv.ParseFloat(value, 32)
				if err != nil {
					return err
				}
				instance.proto.Set(field, protoreflect.ValueOfFloat32(float32(parsedValue)))
				return nil
			})
		case protoreflect.DoubleKind:
			instance.OptionalField(nameable.NewSimpleNameable(fieldName), schema.TodoSchema, func(node *node.Node) error {
				value, err := node.GetExpandedStringValue(mergedEnv)
				if err != nil {
					return err
				}
				parsedValue, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return err
				}
				instance.proto.Set(field, protoreflect.ValueOfFloat64(parsedValue))
				return nil
			})
		case protoreflect.GroupKind, protoreflect.BytesKind:
			// not supported
		}
	}

	return instance
}

func (p *ProtoInstance) Parse(node *node.Node) (*dynamicpb.Message, error) {
	if err := p.DefaultParser.Parse(node); err != nil {
		return nil, err
	}
	return p.proto, nil
}
