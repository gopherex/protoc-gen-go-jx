package jxpb

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-faster/jx"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// --- Timestamp (RFC3339 UTC, 0/3/6/9 fractional digits) ---

func EncTimestamp(e *jx.Encoder, t *timestamppb.Timestamp) {
	tm := time.Unix(t.GetSeconds(), int64(t.GetNanos())).UTC()
	s := tm.Format("2006-01-02T15:04:05")
	if n := t.GetNanos(); n != 0 {
		frac := fmt.Sprintf("%09d", n)
		switch {
		case n%1e6 == 0:
			frac = frac[:3]
		case n%1e3 == 0:
			frac = frac[:6]
		}
		s += "." + frac
	}
	e.Str(s + "Z")
}

func DecTimestamp(d *jx.Decoder, t *timestamppb.Timestamp) error {
	s, err := d.Str()
	if err != nil {
		return err
	}
	tm, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return err
	}
	t.Seconds = tm.Unix()
	t.Nanos = int32(tm.Nanosecond())
	return nil
}

// --- Duration ("<seconds>.<frac>s") ---

func EncDuration(e *jx.Encoder, d *durationpb.Duration) {
	sec, nanos := d.GetSeconds(), d.GetNanos()
	sign := ""
	if sec < 0 || nanos < 0 {
		sign = "-"
		if sec < 0 {
			sec = -sec
		}
		if nanos < 0 {
			nanos = -nanos
		}
	}
	s := sign + strconv.FormatInt(sec, 10)
	if nanos != 0 {
		frac := fmt.Sprintf("%09d", nanos)
		switch {
		case nanos%1e6 == 0:
			frac = frac[:3]
		case nanos%1e3 == 0:
			frac = frac[:6]
		}
		s += "." + frac
	}
	e.Str(s + "s")
}

func DecDuration(d *jx.Decoder, out *durationpb.Duration) error {
	s, err := d.Str()
	if err != nil {
		return err
	}
	if len(s) == 0 || s[len(s)-1] != 's' {
		return fmt.Errorf("invalid duration %q", s)
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	out.Seconds = int64(dur / time.Second)
	out.Nanos = int32(dur % time.Second)
	return nil
}

// --- Empty ---

func EncEmpty(e *jx.Encoder, _ *emptypb.Empty) {
	e.ObjStart()
	e.ObjEnd()
}

func DecEmpty(d *jx.Decoder, _ *emptypb.Empty) error {
	return d.Obj(func(d *jx.Decoder, key string) error {
		return fmt.Errorf("unexpected field %q in Empty", key)
	})
}

// --- FieldMask (comma-joined lowerCamel paths) ---

func EncFieldMask(e *jx.Encoder, m *fieldmaskpb.FieldMask) {
	out := make([]byte, 0, 32)
	for i, p := range m.GetPaths() {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, snakeToCamel(p)...)
	}
	e.Str(string(out))
}

func DecFieldMask(d *jx.Decoder, m *fieldmaskpb.FieldMask) error {
	s, err := d.Str()
	if err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			m.Paths = append(m.Paths, camelToSnake(s[start:i]))
			start = i + 1
		}
	}
	return nil
}

func snakeToCamel(s string) string {
	out := make([]byte, 0, len(s))
	up := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '_' {
			up = true
			continue
		}
		if up && c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		up = false
		out = append(out, c)
	}
	return string(out)
}

func camelToSnake(s string) string {
	out := make([]byte, 0, len(s)+4)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			out = append(out, '_', c+('a'-'A'))
		} else {
			out = append(out, c)
		}
	}
	return string(out)
}

// --- Wrappers (encode/decode the bare value) ---

func EncDoubleValue(e *jx.Encoder, w *wrapperspb.DoubleValue) { EncFloat64(e, w.GetValue()) }
func EncFloatValue(e *jx.Encoder, w *wrapperspb.FloatValue)   { EncFloat32(e, w.GetValue()) }
func EncInt32Value(e *jx.Encoder, w *wrapperspb.Int32Value)   { e.Int32(w.GetValue()) }
func EncUint32Value(e *jx.Encoder, w *wrapperspb.UInt32Value) { e.UInt32(w.GetValue()) }
func EncInt64Value(e *jx.Encoder, w *wrapperspb.Int64Value)   { EncInt64(e, w.GetValue()) }
func EncUint64Value(e *jx.Encoder, w *wrapperspb.UInt64Value) { EncUint64(e, w.GetValue()) }
func EncBoolValue(e *jx.Encoder, w *wrapperspb.BoolValue)     { e.Bool(w.GetValue()) }
func EncStringValue(e *jx.Encoder, w *wrapperspb.StringValue) { e.Str(w.GetValue()) }
func EncBytesValue(e *jx.Encoder, w *wrapperspb.BytesValue)   { EncBytes(e, w.GetValue()) }

func DecDoubleValue(d *jx.Decoder, w *wrapperspb.DoubleValue) error {
	v, err := DecFloat64(d)
	w.Value = v
	return err
}
func DecFloatValue(d *jx.Decoder, w *wrapperspb.FloatValue) error {
	v, err := DecFloat32(d)
	w.Value = v
	return err
}
func DecInt32Value(d *jx.Decoder, w *wrapperspb.Int32Value) error {
	v, err := DecInt32(d)
	w.Value = v
	return err
}
func DecUint32Value(d *jx.Decoder, w *wrapperspb.UInt32Value) error {
	v, err := DecUint32(d)
	w.Value = v
	return err
}
func DecInt64Value(d *jx.Decoder, w *wrapperspb.Int64Value) error {
	v, err := DecInt64(d)
	w.Value = v
	return err
}
func DecUint64Value(d *jx.Decoder, w *wrapperspb.UInt64Value) error {
	v, err := DecUint64(d)
	w.Value = v
	return err
}
func DecBoolValue(d *jx.Decoder, w *wrapperspb.BoolValue) error {
	v, err := d.Bool()
	w.Value = v
	return err
}
func DecStringValue(d *jx.Decoder, w *wrapperspb.StringValue) error {
	v, err := d.Str()
	w.Value = v
	return err
}
func DecBytesValue(d *jx.Decoder, w *wrapperspb.BytesValue) error {
	v, err := DecBytes(d)
	w.Value = v
	return err
}
