package jxpb

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-faster/jx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	structpb "google.golang.org/protobuf/types/known/structpb"
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
	// Parse "[-]<seconds>[.<frac>]s" manually: time.ParseDuration overflows
	// above ~292 years, but protojson Duration spans ~10000 years.
	body := s[:len(s)-1]
	neg := false
	if len(body) > 0 && (body[0] == '-' || body[0] == '+') {
		neg = body[0] == '-'
		body = body[1:]
	}
	secPart, fracPart := body, ""
	if i := strings.IndexByte(body, '.'); i >= 0 {
		secPart, fracPart = body[:i], body[i+1:]
	}
	sec, err := strconv.ParseInt(secPart, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	var nanos int32
	if fracPart != "" {
		if len(fracPart) > 9 {
			return fmt.Errorf("invalid duration %q: too many fractional digits", s)
		}
		for len(fracPart) < 9 {
			fracPart += "0"
		}
		n, err := strconv.ParseInt(fracPart, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", s, err)
		}
		nanos = int32(n)
	}
	if neg {
		sec, nanos = -sec, -nanos
	}
	out.Seconds, out.Nanos = sec, nanos
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

// --- Struct / Value / ListValue ---

func EncValue(e *jx.Encoder, v *structpb.Value) {
	switch k := v.GetKind().(type) {
	case *structpb.Value_NullValue:
		e.Null()
	case *structpb.Value_NumberValue:
		EncFloat64(e, k.NumberValue)
	case *structpb.Value_StringValue:
		e.Str(k.StringValue)
	case *structpb.Value_BoolValue:
		e.Bool(k.BoolValue)
	case *structpb.Value_StructValue:
		EncStruct(e, k.StructValue)
	case *structpb.Value_ListValue:
		EncListValue(e, k.ListValue)
	default:
		e.Null()
	}
}

func EncStruct(e *jx.Encoder, s *structpb.Struct) {
	e.ObjStart()
	for k, v := range s.GetFields() {
		e.FieldStart(k)
		EncValue(e, v)
	}
	e.ObjEnd()
}

func EncListValue(e *jx.Encoder, l *structpb.ListValue) {
	e.ArrStart()
	for _, v := range l.GetValues() {
		EncValue(e, v)
	}
	e.ArrEnd()
}

func DecValue(d *jx.Decoder, v *structpb.Value) error {
	switch d.Next() {
	case jx.Null:
		if err := d.Null(); err != nil {
			return err
		}
		v.Kind = &structpb.Value_NullValue{}
	case jx.Number:
		n, err := d.Float64()
		if err != nil {
			return err
		}
		v.Kind = &structpb.Value_NumberValue{NumberValue: n}
	case jx.String:
		s, err := d.Str()
		if err != nil {
			return err
		}
		v.Kind = &structpb.Value_StringValue{StringValue: s}
	case jx.Bool:
		b, err := d.Bool()
		if err != nil {
			return err
		}
		v.Kind = &structpb.Value_BoolValue{BoolValue: b}
	case jx.Object:
		s := &structpb.Struct{}
		if err := DecStruct(d, s); err != nil {
			return err
		}
		v.Kind = &structpb.Value_StructValue{StructValue: s}
	case jx.Array:
		l := &structpb.ListValue{}
		if err := DecListValue(d, l); err != nil {
			return err
		}
		v.Kind = &structpb.Value_ListValue{ListValue: l}
	default:
		return fmt.Errorf("invalid Value token %s", d.Next())
	}
	return nil
}

func DecStruct(d *jx.Decoder, s *structpb.Struct) error {
	return d.Obj(func(d *jx.Decoder, key string) error {
		if s.Fields == nil {
			s.Fields = map[string]*structpb.Value{}
		}
		val := &structpb.Value{}
		if err := DecValue(d, val); err != nil {
			return err
		}
		s.Fields[key] = val
		return nil
	})
}

func DecListValue(d *jx.Decoder, l *structpb.ListValue) error {
	return d.Arr(func(d *jx.Decoder) error {
		val := &structpb.Value{}
		if err := DecValue(d, val); err != nil {
			return err
		}
		l.Values = append(l.Values, val)
		return nil
	})
}

// EncAny renders google.protobuf.Any. protojson already implements the exact
// "@type" expansion (including WKT-valued Any -> {"@type":..,"value":..}); we
// delegate the bytes and splice them in via Raw to stay infallible.
func EncAny(e *jx.Encoder, a *anypb.Any) {
	if a == nil || a.GetTypeUrl() == "" {
		e.ObjStart()
		e.ObjEnd()
		return
	}
	b, err := protojson.Marshal(a)
	if err != nil {
		// best-effort: unresolved type -> {"@type": url}
		e.ObjStart()
		e.FieldStart("@type")
		e.Str(a.GetTypeUrl())
		e.ObjEnd()
		return
	}
	e.Raw(b)
}

func DecAny(d *jx.Decoder, a *anypb.Any) error {
	raw, err := d.Raw()
	if err != nil {
		return err
	}
	return protojson.Unmarshal(raw, a)
}
