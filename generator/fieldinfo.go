package generator

import (
	"strconv"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// scalarKind classifies a field by how its JSON value is written/read.
type scalarKind int

const (
	kindOther scalarKind = iota
	kindInt32
	kindUint32
	kindInt64  // emitted as JSON string
	kindUint64 // emitted as JSON string
	kindFloat32
	kindFloat64
	kindBool
	kindString
	kindBytes
	kindEnum
	kindMessage
)

func classify(fd protoreflect.FieldDescriptor) scalarKind {
	switch fd.Kind() {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return kindInt32
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return kindUint32
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return kindInt64
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return kindUint64
	case protoreflect.FloatKind:
		return kindFloat32
	case protoreflect.DoubleKind:
		return kindFloat64
	case protoreflect.BoolKind:
		return kindBool
	case protoreflect.StringKind:
		return kindString
	case protoreflect.BytesKind:
		return kindBytes
	case protoreflect.EnumKind:
		return kindEnum
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return kindMessage
	}
	return kindOther
}

func strconvQuote(s string) string { return strconv.Quote(s) }

// zeroLit returns the zero-value literal used in the omit-default check.
func zeroLit(f *protogen.Field) string {
	switch classify(f.Desc) {
	case kindBool:
		return "false"
	default:
		return "0"
	}
}
