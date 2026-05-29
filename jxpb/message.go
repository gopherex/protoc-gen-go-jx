package jxpb

import (
	"github.com/go-faster/jx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Marshaler is implemented by messages that have a generated jx codec.
type Marshaler interface {
	Encode(e *jx.Encoder)
}

// Unmarshaler is implemented by messages that have a generated jx codec.
type Unmarshaler interface {
	Decode(d *jx.Decoder) error
}

// EncMessage encodes a nested message. Messages whose package has a generated
// jx codec use it directly; messages from packages without generated jx code
// (e.g. third-party protos outside our jurisdiction) fall back to protojson.
func EncMessage(e *jx.Encoder, m proto.Message) {
	if jm, ok := m.(Marshaler); ok {
		jm.Encode(e)
		return
	}
	b, err := protojson.Marshal(m)
	if err != nil {
		e.Null()
		return
	}
	e.Raw(b)
}

// DecMessage decodes a nested message, mirroring EncMessage: generated jx codec
// when present, protojson fallback otherwise.
func DecMessage(d *jx.Decoder, m proto.Message) error {
	if jm, ok := m.(Unmarshaler); ok {
		return jm.Decode(d)
	}
	raw, err := d.Raw()
	if err != nil {
		return err
	}
	return protojson.Unmarshal(raw, m)
}
