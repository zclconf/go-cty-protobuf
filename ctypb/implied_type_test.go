package ctypb

import (
	"testing"

	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/zclconf/go-cty-protobuf/internal/testproto"
)

func TestImpliedTypeForMessageDesc(t *testing.T) {
	tests := []struct {
		Input   protoreflect.MessageDescriptor
		Want    cty.Type
		WantErr string
	}{
		{
			Input: (*testproto.Assorted)(nil).ProtoReflect().Descriptor(),
			Want: cty.Object(map[string]cty.Type{
				"t_bool":    cty.Bool,
				"t_bytes":   cty.String,
				"t_double":  cty.Number,
				"t_fixed32": cty.Number,
				"t_fixed64": cty.Number,
				"t_float":   cty.Number,
				"t_int32":   cty.Number,
				"t_int64":   cty.Number,
				"t_message": cty.Object(map[string]cty.Type{
					"t_nested_field": cty.String,
				}),
				"t_sfixed32": cty.Number,
				"t_sfixed64": cty.Number,
				"t_sint32":   cty.Number,
				"t_sint64":   cty.Number,
				"t_string":   cty.String,
				"t_uint32":   cty.Number,
				"t_uint64":   cty.Number,
			}),
		},
		{
			Input: (*testproto.WithOptional)(nil).ProtoReflect().Descriptor(),
			// "optional" has no effect on the implied type, because
			// all values attributes are nullable in cty, but it
			// does decide whether a particular attribute can be null
			// when converting values between the two systems.
			Want: cty.Object(map[string]cty.Type{
				"int32_opt":   cty.Number,
				"int32_req":   cty.Number,
				"message_opt": cty.EmptyObject,
				"message_req": cty.EmptyObject,
				"string_opt":  cty.String,
				"string_req":  cty.String,
			}),
		},
		{
			Input: (*testproto.WithOneOf)(nil).ProtoReflect().Descriptor(),
			Want: cty.Object(map[string]cty.Type{
				"a":       cty.String,
				"b":       cty.String,
				"outside": cty.String,
			}),
		},
		{
			Input: (*testproto.WithRepeated)(nil).ProtoReflect().Descriptor(),
			Want: cty.Object(map[string]cty.Type{
				"t_map_number_bool": cty.Set(
					cty.Object(map[string]cty.Type{
						"key":   cty.Number,
						"value": cty.Bool,
					}),
				),
				"t_map_number_message": cty.Set(
					cty.Object(map[string]cty.Type{
						"key": cty.Number,
						"value": cty.Object(map[string]cty.Type{
							"t_nested_field": cty.String,
						}),
					}),
				),
				"t_map_string_bool": cty.Map(cty.Bool),
				"t_map_string_message": cty.Map(
					cty.Object(map[string]cty.Type{
						"t_nested_field": cty.String,
					}),
				),
				"t_message": cty.List(
					cty.Object(map[string]cty.Type{
						"t_nested_field": cty.String,
					}),
				),
				"t_strings": cty.List(cty.String),
			}),
		},
		{
			Input: (*testproto.WithAny)(nil).ProtoReflect().Descriptor(),
			// This package doesn't treat the "Any" type as special, so
			// it ends up just being a normal object type with a type URL
			// and a value. The caller can then optionally make a further
			// call to decode the value, if needed.
			Want: cty.Object(map[string]cty.Type{
				"t_any": cty.Object(map[string]cty.Type{
					"type_url": cty.String,
					"value":    cty.String,
				}),
				"t_any_list": cty.List(
					cty.Object(map[string]cty.Type{
						"type_url": cty.String,
						"value":    cty.String,
					}),
				),
				"t_any_map_number": cty.Set(
					cty.Object(map[string]cty.Type{
						"key": cty.Number,
						"value": cty.Object(map[string]cty.Type{
							"type_url": cty.String,
							"value":    cty.String,
						}),
					}),
				),
				"t_any_map_string": cty.Map(
					cty.Object(map[string]cty.Type{
						"type_url": cty.String,
						"value":    cty.String,
					}),
				),
				"t_string": cty.String,
			}),
		},
		{
			Input: (*testproto.WithEnum)(nil).ProtoReflect().Descriptor(),
			Want: cty.Object(map[string]cty.Type{
				"t_enum":   cty.String,
				"t_string": cty.String,
			}),
		},
	}

	for _, test := range tests {
		t.Run(string(test.Input.FullName()), func(t *testing.T) {
			got, err := ImpliedTypeForMessageDesc(test.Input)

			if test.WantErr != "" {
				if err == nil {
					t.Fatalf("succeeded; want error\nwant: %s", test.WantErr)
				}
				if got, want := err.Error(), test.WantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			} else if err != nil {
				t.Fatalf("unexpected error\ngot: %s", err.Error())
			}

			if !test.Want.Equals(got) {
				t.Errorf(
					"wrong result\ngot: %s\nwant: %s",
					ctydebug.TypeString(got),
					ctydebug.TypeString(test.Want),
				)
			}
		})
	}
}
