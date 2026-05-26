package golden_test

import (
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/gopherex/protoc-gen-go-jx/example/golden"
)

// benchCases: a flat scalar message and the deep "everything" message.
var benchCases = []func() jxMessage{
	func() jxMessage { return &pb.ScalarTypes{} },
	func() jxMessage { return &pb.Everything{} },
}

func BenchmarkMarshal(b *testing.B) {
	for _, mk := range benchCases {
		msg := mk()
		populate(msg.ProtoReflect(), 4)
		name := string(msg.ProtoReflect().Descriptor().Name())

		b.Run(name+"/jx", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := msg.MarshalJSON(); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(name+"/protojson", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := protojson.Marshal(msg); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	for _, mk := range benchCases {
		msg := mk()
		populate(msg.ProtoReflect(), 4)
		name := string(msg.ProtoReflect().Descriptor().Name())
		data, err := protojson.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}

		b.Run(name+"/jx", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := mk().UnmarshalJSON(data); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(name+"/protojson", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := protojson.Unmarshal(data, mk()); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
