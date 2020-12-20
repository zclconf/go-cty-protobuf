package ctypb

import (
	"encoding/base64"

	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// FromProtobufMessage converts the given message to an equivalent cty.Value,
// which will always be of an object type.
//
// Specifically, the result is guaranteed to conform to the type that
// ImpliedTypeForMessageDesc would've returned if given the message descriptor
// that's associated with the given Message.
//
// Note that FromProtobufMessage takes a protoreflect.Message rather than
// a proto.Message value directly. You can obtain a protoreflect.Message
// value from a proto.Message value by calling its ProtoReflect method.
func FromProtobufMessage(msg protoreflect.Message) (cty.Value, error) {
	path := make(cty.Path, 0, 4) // some capacity to avoid further allocs for shallow structures
	return fromProtobufMessage(msg, path)
}

func fromProtobufMessage(msg protoreflect.Message, path cty.Path) (cty.Value, error) {
	desc := msg.Descriptor()
	fields := desc.Fields()
	attrs := make(map[string]cty.Value, fields.Len())

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := string(field.Name())

		// Temporarily extend path with new attribute name
		path := append(path, cty.GetAttrStep{Name: name})

		if field.HasPresence() && !msg.Has(field) {
			// For presence-tracking fields that are absent, the cty
			// representation is a null value of the field's implied
			// type.
			aty, err := impliedTypeForFieldDesc(field, path)
			if err != nil {
				return cty.NilVal, err
			}
			attrs[name] = cty.NullVal(aty)
			continue
		}

		rawV := msg.Get(field)
		v, err := fromProtobufFieldValue(rawV, field, path)
		if err != nil {
			return cty.NilVal, err
		}
		attrs[name] = v
	}

	return cty.ObjectVal(attrs), nil
}

func fromProtobufFieldValue(rawV protoreflect.Value, field protoreflect.FieldDescriptor, path cty.Path) (cty.Value, error) {
	// This should generally follow the same structure as in
	// impliedTypeForFieldDesc, because we must always produce
	// a value of the same type that impliedTypeForFieldDesc
	// would've returned for our FieldDescriptor.

	switch {
	case field.IsMap():
		subDesc := field.Message()
		subFields := subDesc.Fields()
		keyField := subFields.ByNumber(1)
		valField := subFields.ByNumber(2)
		rawMap := rawV.Map()
		switch {
		case keyField.Kind() == protoreflect.StringKind:
			elems := make(map[string]cty.Value, rawMap.Len())
			var err error
			rawMap.Range(func(rawK protoreflect.MapKey, rawV protoreflect.Value) bool {
				key := rawK.String()

				// Temporarily extend path with placeholder for indexing.
				path := append(path, cty.IndexStep{Key: cty.StringVal(key)})

				ev, thisErr := fromProtobufFieldValue(rawV, valField, path)
				if thisErr != nil {
					err = thisErr
					return false
				}
				elems[key] = ev
				return true
			})
			if err != nil {
				return cty.NilVal, err
			}
			if len(elems) == 0 {
				path := append(path, cty.IndexStep{Key: cty.UnknownVal(cty.String)})
				ety, err := impliedTypeForFieldDesc(valField, path)
				if err != nil {
					return cty.NilVal, err
				}
				return cty.MapValEmpty(ety), nil
			}
			return cty.MapVal(elems), nil
		default:
			elems := make([]cty.Value, 0, rawMap.Len())
			var err error
			rawMap.Range(func(rawK protoreflect.MapKey, rawV protoreflect.Value) bool {
				// Temporarily extend path with placeholder for indexing.
				// We don't actually know the "index" here because we're
				// building a set element and so the element itself would
				// be the "index", and we've not finished building it.
				path := append(path, cty.IndexStep{Key: cty.DynamicVal})

				rawKV := rawK.Value()
				ek, thisErr := fromProtobufFieldValue(rawKV, keyField, path)
				if thisErr != nil {
					err = thisErr
					return false
				}

				ev, thisErr := fromProtobufFieldValue(rawV, valField, path)
				if thisErr != nil {
					err = thisErr
					return false
				}

				elems = append(elems, cty.ObjectVal(map[string]cty.Value{
					"key":   ek,
					"value": ev,
				}))
				return true
			})
			if err != nil {
				return cty.NilVal, err
			}
			if len(elems) == 0 {
				path := append(path, cty.IndexStep{Key: cty.DynamicVal})
				keyTy, err := impliedTypeForFieldDesc(keyField, path)
				if err != nil {
					return cty.NilVal, err
				}
				valTy, err := impliedTypeForFieldDesc(valField, path)
				if err != nil {
					return cty.NilVal, err
				}
				return cty.SetValEmpty(cty.Object(map[string]cty.Type{
					"key":   keyTy,
					"value": valTy,
				})), nil
			}
			return cty.SetVal(elems), nil
		}
	case field.IsList():
		rawList := rawV.List()
		elems := make([]cty.Value, rawList.Len())
		for i := 0; i < rawList.Len(); i++ {
			// Temporarily extend path with placeholder for indexing.
			path := append(path, cty.IndexStep{Key: cty.NumberIntVal(int64(i))})

			rawEV := rawList.Get(i)
			ev, err := fromProtobufFieldKindValue(rawEV, field, path)
			if err != nil {
				return cty.NilVal, err
			}
			elems[i] = ev
		}
		if len(elems) == 0 {
			path := append(path, cty.IndexStep{Key: cty.UnknownVal(cty.Number)})
			ety, err := impliedTypeForFieldKind(field, path)
			if err != nil {
				return cty.NilVal, err
			}
			return cty.ListValEmpty(ety), nil
		}
		return cty.ListVal(elems), nil
	default:
		return fromProtobufFieldKindValue(rawV, field, path)
	}
}

func fromProtobufFieldKindValue(rawV protoreflect.Value, field protoreflect.FieldDescriptor, path cty.Path) (cty.Value, error) {
	switch kind := field.Kind(); kind {
	case protoreflect.BoolKind:
		if rawV.Bool() {
			return cty.True, nil
		}
		return cty.False, nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return cty.NumberIntVal(rawV.Int()), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return cty.NumberUIntVal(rawV.Uint()), nil
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return cty.NumberFloatVal(rawV.Float()), nil
	case protoreflect.StringKind:
		return cty.StringVal(rawV.String()), nil
	case protoreflect.BytesKind:
		// cty strings are sequences of unicode characters rather than of
		// bytes, so our convention is to Base64-encode the bytes to
		// represent them in cty without loss.
		return cty.StringVal(base64.StdEncoding.EncodeToString(rawV.Bytes())), nil
	case protoreflect.EnumKind:
		// cty doesn't have a sense of enums, so for usability we translate
		// these to strings based on the enum field names. That means we
		// need to translate the stored number into a name to return.
		num := rawV.Enum()
		desc := field.Enum().Values().ByNumber(rawV.Enum())
		if desc == nil {
			// Invalid enum member, then
			return cty.NilVal, path.NewErrorf("value %d is not part of the enumeration", num)
		}
		return cty.StringVal(string(desc.Name())), nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		sub := rawV.Message()
		return fromProtobufMessage(sub, path)
	default:
		return cty.NilVal, path.NewErrorf("no cty equivalent for protobuf kind %s", kind.String())
	}
}
