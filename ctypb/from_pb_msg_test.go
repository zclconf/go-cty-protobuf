package ctypb

import (
	"testing"

	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty-protobuf/internal/testproto"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestFromProtobufMessage(t *testing.T) {
	ptrString := func(s string) *string {
		return &s
	}
	ptrInt32 := func(i int32) *int32 {
		return &i
	}
	mustAny := func(msg protoreflect.ProtoMessage) *anypb.Any {
		v, err := anypb.New(msg)
		if err != nil {
			panic(err)
		}
		return v
	}

	tests := map[string]struct {
		Input   protoreflect.ProtoMessage
		Want    cty.Value
		WantErr string
	}{
		"assorted all unset": {
			Input: &testproto.Assorted{},
			// With all of the fields unset, everything is set to its default.
			// Of all of the kinds we've used here, only messages track
			// presence by default, and so all of these take on the default
			// value of their kind rather than appearing as null.
			Want: cty.ObjectVal(map[string]cty.Value{
				"t_bool":    cty.False,
				"t_bytes":   cty.StringVal(""),
				"t_double":  cty.NumberIntVal(0),
				"t_fixed32": cty.NumberIntVal(0),
				"t_fixed64": cty.NumberIntVal(0),
				"t_float":   cty.NumberIntVal(0),
				"t_int32":   cty.NumberIntVal(0),
				"t_int64":   cty.NumberIntVal(0),
				"t_message": cty.NullVal(cty.Object(map[string]cty.Type{
					"t_nested_field": cty.String,
				})),
				"t_sfixed32": cty.NumberIntVal(0),
				"t_sfixed64": cty.NumberIntVal(0),
				"t_sint32":   cty.NumberIntVal(0),
				"t_sint64":   cty.NumberIntVal(0),
				"t_string":   cty.StringVal(""),
				"t_uint32":   cty.NumberIntVal(0),
				"t_uint64":   cty.NumberIntVal(0),
			}),
		},
		"assorted all set": {
			Input: &testproto.Assorted{
				TDouble:   1.5,
				TFloat:    2.5,
				TInt32:    -7,
				TInt64:    -8,
				TUint32:   9,
				TUint64:   10,
				TSint32:   -11,
				TSint64:   -12,
				TFixed32:  64,
				TFixed64:  65,
				TSfixed32: -64,
				TSfixed64: -65,
				TBool:     true,
				TString:   "hello",
				TBytes:    []byte("HELLO COMPUTER"),
				TMessage: &testproto.Assorted_Nested{
					TNestedField: "beep",
				},
			},
			Want: cty.ObjectVal(map[string]cty.Value{
				"t_bool":    cty.True,
				"t_bytes":   cty.StringVal("SEVMTE8gQ09NUFVURVI="),
				"t_double":  cty.NumberFloatVal(1.5),
				"t_fixed32": cty.NumberIntVal(64),
				"t_fixed64": cty.NumberIntVal(65),
				"t_float":   cty.NumberFloatVal(2.5),
				"t_int32":   cty.NumberIntVal(-7),
				"t_int64":   cty.NumberIntVal(-8),
				"t_message": cty.ObjectVal(map[string]cty.Value{
					"t_nested_field": cty.StringVal("beep"),
				}),
				"t_sfixed32": cty.NumberIntVal(-64),
				"t_sfixed64": cty.NumberIntVal(-65),
				"t_sint32":   cty.NumberIntVal(-11),
				"t_sint64":   cty.NumberIntVal(-12),
				"t_string":   cty.StringVal("hello"),
				"t_uint32":   cty.NumberIntVal(9),
				"t_uint64":   cty.NumberIntVal(10),
			}),
		},
		"Optional all unset": {
			Input: &testproto.WithOptional{},
			// Only the fields with presence tracking appear as null.
			// The others take on their default values.
			Want: cty.ObjectVal(map[string]cty.Value{
				"int32_opt":   cty.NullVal(cty.Number),
				"int32_req":   cty.NumberIntVal(0),
				"message_opt": cty.NullVal(cty.EmptyObject), // message fields always track presence
				"message_req": cty.NullVal(cty.EmptyObject),
				"string_opt":  cty.NullVal(cty.String),
				"string_req":  cty.StringVal(""),
			}),
		},
		"Optional all set": {
			Input: &testproto.WithOptional{
				StringReq:  "hi required",
				StringOpt:  ptrString("hi optional"),
				Int32Req:   12,
				Int32Opt:   ptrInt32(13),
				MessageReq: &testproto.WithOptional_Nested{},
				MessageOpt: &testproto.WithOptional_Nested{},
			},
			Want: cty.ObjectVal(map[string]cty.Value{
				"int32_opt":   cty.NumberIntVal(13),
				"int32_req":   cty.NumberIntVal(12),
				"message_opt": cty.EmptyObjectVal,
				"message_req": cty.EmptyObjectVal,
				"string_opt":  cty.StringVal("hi optional"),
				"string_req":  cty.StringVal("hi required"),
			}),
		},
		"OneOf all unset": {
			Input: &testproto.WithOneOf{},
			// In the presence of a OneOf, the unused choices are left as
			// null to unambiguously signal which one was set.
			Want: cty.ObjectVal(map[string]cty.Value{
				"a":       cty.NullVal(cty.String),
				"b":       cty.NullVal(cty.String),
				"outside": cty.StringVal(""),
			}),
		},
		"OneOf all set": {
			Input: &testproto.WithOneOf{
				Outside: "hello",
				TOneof:  &testproto.WithOneOf_B{B: "boop"},
			},
			// In the presence of a OneOf, the unused choices are left as
			// null to unambiguously signal which one was set.
			Want: cty.ObjectVal(map[string]cty.Value{
				"a":       cty.NullVal(cty.String), // null because b is set
				"b":       cty.StringVal("boop"),
				"outside": cty.StringVal("hello"),
			}),
		},
		"Repeated all unset": {
			Input: &testproto.WithRepeated{},
			Want: cty.ObjectVal(map[string]cty.Value{
				"t_map_number_bool": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"key":   cty.Number,
					"value": cty.Bool,
				})),
				"t_map_number_message": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"key": cty.Number,
					"value": cty.Object(map[string]cty.Type{
						"t_nested_field": cty.String,
					}),
				})),
				"t_map_string_bool": cty.MapValEmpty(cty.Bool),
				"t_map_string_message": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"t_nested_field": cty.String,
				})),
				"t_message": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"t_nested_field": cty.String,
				})),
				"t_strings": cty.ListValEmpty(cty.String),
			}),
		},
		"Repeated all set": {
			Input: &testproto.WithRepeated{
				TStrings:       []string{"hello", "world"},
				TMessage:       []*testproto.WithRepeated_Nested{{}},
				TMapStringBool: map[string]bool{"a": true, "b": false},
				TMapNumberBool: map[int64]bool{1: true, 2: false},
				TMapStringMessage: map[string]*testproto.WithRepeated_Nested{
					"a": {}, "b": {},
				},
				TMapNumberMessage: map[int64]*testproto.WithRepeated_Nested{
					1: {}, 2: {},
				},
			},
			Want: cty.ObjectVal(map[string]cty.Value{
				"t_map_number_bool": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"key":   cty.NumberIntVal(1),
						"value": cty.True,
					}),
					cty.ObjectVal(map[string]cty.Value{
						"key":   cty.NumberIntVal(2),
						"value": cty.False,
					}),
				}),
				"t_map_number_message": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"key": cty.NumberIntVal(1),
						"value": cty.ObjectVal(map[string]cty.Value{
							"t_nested_field": cty.StringVal(""),
						}),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"key": cty.NumberIntVal(2),
						"value": cty.ObjectVal(map[string]cty.Value{
							"t_nested_field": cty.StringVal(""),
						}),
					}),
				}),
				"t_map_string_bool": cty.MapVal(map[string]cty.Value{
					"a": cty.True,
					"b": cty.False,
				}),
				"t_map_string_message": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"t_nested_field": cty.StringVal(""),
					}),
					"b": cty.ObjectVal(map[string]cty.Value{
						"t_nested_field": cty.StringVal(""),
					}),
				}),
				"t_message": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"t_nested_field": cty.StringVal(""),
					}),
				}),
				"t_strings": cty.ListVal([]cty.Value{
					cty.StringVal("hello"),
					cty.StringVal("world"),
				}),
			}),
		},
		"Any all unset": {
			Input: &testproto.WithAny{},
			Want: cty.ObjectVal(map[string]cty.Value{
				"t_any": cty.NullVal(cty.Object(map[string]cty.Type{
					"type_url": cty.String,
					"value":    cty.String,
				})),
				"t_any_list": cty.ListValEmpty(cty.Object(map[string]cty.Type{
					"type_url": cty.String,
					"value":    cty.String,
				})),
				"t_any_map_number": cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"key": cty.Number,
					"value": cty.Object(map[string]cty.Type{
						"type_url": cty.String,
						"value":    cty.String,
					}),
				})),
				"t_any_map_string": cty.MapValEmpty(cty.Object(map[string]cty.Type{
					"type_url": cty.String,
					"value":    cty.String,
				})),
				"t_string": cty.StringVal(""),
			}),
		},
		"Any all set": {
			Input: &testproto.WithAny{
				TString: "not an any",
				TAny:    mustAny(&testproto.Simple{Foo: &testproto.Empty{}}),
				TAnyList: []*anypb.Any{
					mustAny(&testproto.Empty{}),
				},
				TAnyMapString: map[string]*anypb.Any{
					"a": mustAny(&testproto.Empty{}),
				},
				TAnyMapNumber: map[int64]*anypb.Any{
					4: mustAny(&testproto.Empty{}),
				},
			},
			Want: cty.ObjectVal(map[string]cty.Value{
				"t_any": cty.ObjectVal(map[string]cty.Value{
					"type_url": cty.StringVal("type.googleapis.com/testproto.Simple"),
					"value":    cty.StringVal("CgA="), // base64-encoded Simple message
				}),
				"t_any_list": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"type_url": cty.StringVal("type.googleapis.com/testproto.Empty"),
						"value":    cty.StringVal(""),
					}),
				}),
				"t_any_map_number": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"key": cty.NumberIntVal(4),
						"value": cty.ObjectVal(map[string]cty.Value{
							"type_url": cty.StringVal("type.googleapis.com/testproto.Empty"),
							"value":    cty.StringVal(""),
						}),
					}),
				}),
				"t_any_map_string": cty.MapVal(map[string]cty.Value{
					"a": cty.ObjectVal(map[string]cty.Value{
						"type_url": cty.StringVal("type.googleapis.com/testproto.Empty"),
						"value":    cty.StringVal(""),
					}),
				}),
				"t_string": cty.StringVal("not an any"),
			}),
		},
		"Enum all unset": {
			Input: &testproto.WithEnum{},
			Want: cty.ObjectVal(map[string]cty.Value{
				"t_enum":   cty.StringVal("A"),
				"t_string": cty.StringVal(""),
			}),
		},
		"Enum all set": {
			Input: &testproto.WithEnum{
				TString: "hello",
				TEnum:   testproto.WithEnum_d,
			},
			Want: cty.ObjectVal(map[string]cty.Value{
				"t_enum":   cty.StringVal("d"),
				"t_string": cty.StringVal("hello"),
			}),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := FromProtobufMessage(test.Input.ProtoReflect())

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

			if !test.Want.RawEquals(got) {
				t.Errorf(
					"wrong result\ngot: %s\nwant: %s",
					ctydebug.ValueString(got),
					ctydebug.ValueString(test.Want),
				)
			}
		})
	}

}
