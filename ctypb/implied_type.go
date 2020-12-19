package ctypb

import (
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ImpliedTypeForMessageDesc returns a cty.Type which corresponds to the given
// protocol buffers message descriptor.
//
// The result will always be an object type, whose attributes each correspond
// to fields of the message descriptor. The types of those attributes will
// depend on the definitions of each field.
//
// The conversion from protobuf schema to cty is lossy, because cty and
// protobuf do not have all concepts in common. In particular, the conversion
// will treat "oneOf" definitions as a set of normal fields where only one
// can be non-null by convention, and all of the specific protocol buffers
// numeric types will be generalized to cty.Number.
//
// Protocol buffers compatibility rules do not necessarily translate directly
// to cty: adding new fields to an existing message type will cause the
// resulting object type to be non-equal to the previous object type. Whether
// that is important will depend on what the calling application intends to
// do with the resulting type.
//
// If ImpliedTypeForMessageDesc returns an error then it might be a
// cty.PathError referring to a specific sub-path within the generated type.
func ImpliedTypeForMessageDesc(desc protoreflect.MessageDescriptor) (cty.Type, error) {
	path := make(cty.Path, 0, 4) // four levels deep without further allocation
	ty, err := impliedTypeForMessageDesc(desc, path)
	return ty, err
}

func impliedTypeForMessageDesc(desc protoreflect.MessageDescriptor, path cty.Path) (ty cty.Type, err error) {
	fields := desc.Fields()
	atys := make(map[string]cty.Type, fields.Len())
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		name := string(field.Name())

		// Temporarily extend path with new attribute name
		path := append(path, cty.GetAttrStep{Name: name})
		aty, err := impliedTypeForFieldDesc(field, path)
		if err != nil {
			return cty.NilType, err
		}
		atys[name] = aty
	}
	return cty.Object(atys), nil
}

func impliedTypeForFieldDesc(field protoreflect.FieldDescriptor, path cty.Path) (ty cty.Type, err error) {
	isRepeated := field.Cardinality() == protoreflect.Repeated

	if isRepeated {
		// Protocol buffers has a special case for repeated messages
		// representing entries in a map, which we'll respect by
		// representing it as a cty map as long as the map keys
		// are strings. Otherwise, we'll produce a set of map
		// entry objects with "key" and "value" attributes, because
		// that's the closest approximation of that intent which we
		// can achieve in cty.
		if kind, sub := field.Kind(), field.Message(); kind == protoreflect.MessageKind && sub.IsMapEntry() {
			subFields := sub.Fields()
			keyField := subFields.ByNumber(1)
			valField := subFields.ByNumber(2)
			if keyField.Kind() == protoreflect.StringKind {
				// Temporarily extend path with placeholder for indexing.
				path := append(path, cty.IndexStep{Key: cty.UnknownVal(cty.String)})
				valTy, err := impliedTypeForFieldDesc(valField, path)
				if err != nil {
					return cty.NilType, err
				}
				return cty.Map(valTy), nil
			} else {
				keyTy, err := impliedTypeForFieldDesc(keyField, path)
				if err != nil {
					return cty.NilType, err
				}
				// Temporarily extend path with placeholder for indexing.
				path := append(path, cty.IndexStep{Key: cty.UnknownVal(keyTy)})
				valTy, err := impliedTypeForFieldDesc(valField, path)
				if err != nil {
					return cty.NilType, err
				}
				return cty.Set(cty.Object(map[string]cty.Type{
					"key":   keyTy,
					"value": valTy,
				})), nil
			}
		}
	}

	// Determine the base type, ignoring cardinality for now.
	aty, err := impliedTypeForFieldKind(field, path)
	if err != nil {
		return cty.NilType, err
	}

	// If the field is "repeated" and it didn't match our special case for
	// maps above then the result is a list of the base type we already
	// determined.
	if isRepeated {
		return cty.List(aty), nil
	}
	return aty, nil
}

// impliedTypeForFieldKind determines a corresponding type for the given
// field's kind (and optionally, nested message type) while disregarding
// the cardinality.
func impliedTypeForFieldKind(field protoreflect.FieldDescriptor, path cty.Path) (ty cty.Type, err error) {
	switch kind := field.Kind(); kind {
	case protoreflect.BoolKind:
		return cty.Bool, nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Uint64Kind, protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind, protoreflect.FloatKind, protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind, protoreflect.DoubleKind:
		return cty.Number, nil
	case protoreflect.StringKind, protoreflect.BytesKind, protoreflect.EnumKind:
		return cty.String, nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		// The type is that of the nested message descriptor.
		return impliedTypeForMessageDesc(field.Message(), path)
	default:
		return cty.NilType, path.NewErrorf("no cty equivalent for protobuf kind %s", kind.String())
	}
}
