package jxpb

import (
	"testing"

	"github.com/go-faster/jx"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// wrapperspb.StringValue is a proto.Message without a generated jx codec, so it
// exercises the protojson fallback in EncMessage/DecMessage — the case of a
// message from a package outside our jurisdiction.

func TestEncMessageFallback(t *testing.T) {
	var e jx.Encoder
	EncMessage(&e, wrapperspb.String("hi"))
	// protojson renders a StringValue wrapper as a bare JSON string.
	if got := string(e.Bytes()); got != `"hi"` {
		t.Fatalf("EncMessage fallback = %s, want \"hi\"", got)
	}
}

func TestDecMessageFallback(t *testing.T) {
	var got wrapperspb.StringValue
	if err := DecMessage(jx.DecodeStr(`"hi"`), &got); err != nil {
		t.Fatalf("DecMessage fallback: %v", err)
	}
	if got.GetValue() != "hi" {
		t.Fatalf("DecMessage fallback = %q, want hi", got.GetValue())
	}
}

// jxMsg implements Marshaler/Unmarshaler to confirm the direct path is taken
// (no protojson) when a generated codec is present.
type jxMsg struct {
	proto.Message
	encoded bool
	decoded bool
}

func (m *jxMsg) Encode(e *jx.Encoder) { m.encoded = true; e.Bool(true) }
func (m *jxMsg) Decode(d *jx.Decoder) error {
	m.decoded = true
	_, err := d.Bool()
	return err
}

func TestEncMessageUsesGeneratedCodec(t *testing.T) {
	m := &jxMsg{}
	var e jx.Encoder
	EncMessage(&e, m)
	if !m.encoded {
		t.Fatal("EncMessage did not use generated Encode")
	}
	if got := string(e.Bytes()); got != "true" {
		t.Fatalf("got %s", got)
	}
}

func TestDecMessageUsesGeneratedCodec(t *testing.T) {
	m := &jxMsg{}
	if err := DecMessage(jx.DecodeStr(`true`), m); err != nil {
		t.Fatal(err)
	}
	if !m.decoded {
		t.Fatal("DecMessage did not use generated Decode")
	}
}
