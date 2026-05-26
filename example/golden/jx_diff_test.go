package golden_test

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	anypb "google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	structpb "google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

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
	func() jxMessage { return &pb.MapKeyTypes{} },
	func() jxMessage { return &pb.MapValueTypes{} },
	func() jxMessage { return &pb.OneofContainer{} },
	func() jxMessage { return &pb.TreeNode{} },
	func() jxMessage { return &pb.MutualA{} },
	func() jxMessage { return &pb.MutualB{} },
	func() jxMessage { return &pb.Reserved{} },
	func() jxMessage { return &pb.WellKnownTypes{} },
	func() jxMessage { return &pb.Everything{} },
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

// TestNegativeZeroEmitted: protojson keeps -0.0 (distinct from default 0), so we
// must emit it too. jsonEqual catches omission (want has the key, ours wouldn't).
func TestNegativeZeroEmitted(t *testing.T) {
	m := &pb.ScalarTypes{FieldDouble: math.Copysign(0, -1), FieldFloat: float32(math.Copysign(0, -1))}
	ours, err := m.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	want, err := protojson.Marshal(m)
	if err != nil {
		t.Fatalf("protojson.Marshal: %v", err)
	}
	if !jsonEqual(t, ours, want) {
		t.Fatalf("negative zero mismatch:\n ours: %s\n want: %s", ours, want)
	}
}

// #1: decode accepts original proto (snake_case) field names, like protojson.
func TestDecodeAcceptsProtoNames(t *testing.T) {
	var m pb.ScalarTypes
	if err := m.UnmarshalJSON([]byte(`{"field_int32":7,"field_string":"hi"}`)); err != nil {
		t.Fatalf("decode snake_case: %v", err)
	}
	if m.FieldInt32 != 7 || m.FieldString != "hi" {
		t.Fatalf("got %+v", &m)
	}
}

// #2: duplicate field keys are rejected (incl. via a camel/snake alias of the
// same field).
func TestDecodeRejectsDuplicateField(t *testing.T) {
	if err := (&pb.ScalarTypes{}).UnmarshalJSON([]byte(`{"fieldInt32":1,"fieldInt32":2}`)); err == nil {
		t.Fatal("expected duplicate-field error")
	}
	if err := (&pb.ScalarTypes{}).UnmarshalJSON([]byte(`{"fieldInt32":1,"field_int32":2}`)); err == nil {
		t.Fatal("expected duplicate-field error via alias")
	}
}

// #3: two keys for the same oneof are rejected; a single member decodes fine.
func TestDecodeRejectsOneofConflict(t *testing.T) {
	if err := (&pb.OneofContainer{}).UnmarshalJSON([]byte(`{"choiceBool":true,"choiceInt32":5}`)); err == nil {
		t.Fatal("expected oneof-conflict error")
	}
	var ok pb.OneofContainer
	if err := ok.UnmarshalJSON([]byte(`{"choiceBool":true,"otherA":"x"}`)); err != nil {
		t.Fatalf("distinct oneofs should decode: %v", err)
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
		if v, ok := sampleWKT(fd.Message()); ok {
			return v
		}
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

func sampleWKT(md protoreflect.MessageDescriptor) (protoreflect.Value, bool) {
	switch md.FullName() {
	case "google.protobuf.Timestamp":
		return wrapMsg(timestamppb.New(time.Unix(1700000000, 500000000).UTC())), true
	case "google.protobuf.Duration":
		return wrapMsg(&durationpb.Duration{Seconds: 1, Nanos: 500000000}), true
	case "google.protobuf.Any":
		a, _ := anypb.New(&durationpb.Duration{Seconds: 3})
		return wrapMsg(a), true
	case "google.protobuf.Empty":
		return wrapMsg(&emptypb.Empty{}), true
	case "google.protobuf.FieldMask":
		return wrapMsg(&fieldmaskpb.FieldMask{Paths: []string{"a.b", "c"}}), true
	case "google.protobuf.Struct":
		s, _ := structpb.NewStruct(map[string]any{"k": "v", "n": 1.0})
		return wrapMsg(s), true
	case "google.protobuf.Value":
		return wrapMsg(structpb.NewStringValue("x")), true
	case "google.protobuf.ListValue":
		l, _ := structpb.NewList([]any{1.0, "two"})
		return wrapMsg(l), true
	case "google.protobuf.DoubleValue":
		return wrapMsg(wrapperspb.Double(1.5)), true
	case "google.protobuf.FloatValue":
		return wrapMsg(wrapperspb.Float(2.5)), true
	case "google.protobuf.Int32Value":
		return wrapMsg(wrapperspb.Int32(7)), true
	case "google.protobuf.UInt32Value":
		return wrapMsg(wrapperspb.UInt32(8)), true
	case "google.protobuf.Int64Value":
		return wrapMsg(wrapperspb.Int64(9000000000)), true
	case "google.protobuf.UInt64Value":
		return wrapMsg(wrapperspb.UInt64(9000000001)), true
	case "google.protobuf.BoolValue":
		return wrapMsg(wrapperspb.Bool(true)), true
	case "google.protobuf.StringValue":
		return wrapMsg(wrapperspb.String("s")), true
	case "google.protobuf.BytesValue":
		return wrapMsg(wrapperspb.Bytes([]byte("b"))), true
	}
	return protoreflect.Value{}, false
}

func wrapMsg(m proto.Message) protoreflect.Value {
	return protoreflect.ValueOfMessage(m.ProtoReflect())
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
