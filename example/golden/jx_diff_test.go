package golden_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	pb "github.com/gopherex/protoc-gen-go-jx/example/golden"
)

// jxMessage is satisfied by every generated message.
type jxMessage interface {
	proto.Message
	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error
}

// diffCases lists message factories whose every field kind is implemented.
// Append to this as tasks land.
var diffCases = []func() jxMessage{
	func() jxMessage { return &pb.ScalarTypes{} },
	func() jxMessage { return &pb.OptionalScalarTypes{} },
	func() jxMessage { return &pb.RepeatedScalarTypes{} },
	func() jxMessage { return &pb.EnumAndMessageFields{} },
	func() jxMessage { return &pb.Outer{} },
	func() jxMessage { return &pb.Outer_Inner{} },
}

func TestDiffAgainstProtojson(t *testing.T) {
	for _, mk := range diffCases {
		msg := mk()
		name := string(msg.ProtoReflect().Descriptor().Name())
		t.Run(name, func(t *testing.T) {
			populate(msg.ProtoReflect(), 3)

			ours, err := msg.MarshalJSON()
			if err != nil {
				t.Fatalf("our MarshalJSON: %v", err)
			}
			want, err := protojson.Marshal(msg)
			if err != nil {
				t.Fatalf("protojson.Marshal: %v", err)
			}
			if !jsonEqual(t, ours, want) {
				t.Fatalf("JSON mismatch:\n ours: %s\n want: %s", ours, want)
			}

			dec := mk()
			if err := dec.UnmarshalJSON(ours); err != nil {
				t.Fatalf("our UnmarshalJSON: %v", err)
			}
			if !proto.Equal(msg, dec) {
				t.Fatalf("round-trip mismatch:\n in:  %v\n out: %v", msg, dec)
			}
		})
	}
}

func jsonEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var ua, ub any
	if err := json.Unmarshal(a, &ua); err != nil {
		t.Fatalf("parse ours: %v (%s)", err, a)
	}
	if err := json.Unmarshal(b, &ub); err != nil {
		t.Fatalf("parse want: %v (%s)", err, b)
	}
	return reflect.DeepEqual(ua, ub)
}

func populate(m protoreflect.Message, depth int) {
	fds := m.Descriptor().Fields()
	for i := 0; i < fds.Len(); i++ {
		fd := fds.Get(i)
		if fd.ContainingOneof() != nil {
			oo := fd.ContainingOneof()
			if fd.Index() != oo.Fields().Get(0).Index() {
				continue
			}
		}
		switch {
		case fd.IsMap():
			if depth <= 0 {
				continue
			}
			mp := m.NewField(fd)
			k := sampleMapKey(fd.MapKey())
			v := sampleValue(fd.MapValue(), depth-1)
			mp.Map().Set(k.MapKey(), v)
			m.Set(fd, mp)
		case fd.IsList():
			if fd.Message() != nil && depth <= 0 {
				continue
			}
			lst := m.NewField(fd)
			lst.List().Append(sampleValue(fd, depth-1))
			m.Set(fd, lst)
		case fd.Message() != nil:
			if depth <= 0 {
				continue
			}
			m.Set(fd, sampleValue(fd, depth-1))
		default:
			m.Set(fd, sampleValue(fd, depth-1))
		}
	}
}

func sampleValue(fd protoreflect.FieldDescriptor, depth int) protoreflect.Value {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(true)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(11)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(12)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(9000000000)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(9000000001)
	case protoreflect.FloatKind:
		return protoreflect.ValueOfFloat32(1.5)
	case protoreflect.DoubleKind:
		return protoreflect.ValueOfFloat64(2.5)
	case protoreflect.StringKind:
		return protoreflect.ValueOfString("hello")
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte("hi"))
	case protoreflect.EnumKind:
		vals := fd.Enum().Values()
		return protoreflect.ValueOfEnum(vals.Get(vals.Len() - 1).Number())
	case protoreflect.MessageKind, protoreflect.GroupKind:
		sub := newMessageValue(fd)
		populate(sub.Message(), depth)
		return sub
	}
	panic("unhandled kind")
}

func newMessageValue(fd protoreflect.FieldDescriptor) protoreflect.Value {
	return protoreflect.ValueOfMessage(dynamicNew(fd.Message()))
}

func dynamicNew(md protoreflect.MessageDescriptor) protoreflect.Message {
	mt, err := protoregistry.GlobalTypes.FindMessageByName(md.FullName())
	if err != nil {
		panic(err)
	}
	return mt.New()
}

func sampleMapKey(fd protoreflect.FieldDescriptor) protoreflect.Value {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return protoreflect.ValueOfBool(true)
	case protoreflect.StringKind:
		return protoreflect.ValueOfString("k")
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(7)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(7)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return protoreflect.ValueOfInt64(7)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return protoreflect.ValueOfUint64(7)
	}
	panic("unhandled map key kind")
}
