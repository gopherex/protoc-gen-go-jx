// Package jxpb provides the runtime helpers used by protoc-gen-go-jx generated code.
package jxpb

import (
	"encoding/base64"
	"math"
	"strconv"

	"github.com/go-faster/jx"
)

// 64-bit integers are encoded as JSON strings (protojson rule) and decoded
// from either a JSON string or a JSON number.

func EncInt64(e *jx.Encoder, v int64)   { e.Str(strconv.FormatInt(v, 10)) }
func EncUint64(e *jx.Encoder, v uint64) { e.Str(strconv.FormatUint(v, 10)) }

func DecInt64(d *jx.Decoder) (int64, error) {
	if d.Next() == jx.String {
		s, err := d.Str()
		if err != nil {
			return 0, err
		}
		return strconv.ParseInt(s, 10, 64)
	}
	return d.Int64()
}

func DecUint64(d *jx.Decoder) (uint64, error) {
	if d.Next() == jx.String {
		s, err := d.Str()
		if err != nil {
			return 0, err
		}
		return strconv.ParseUint(s, 10, 64)
	}
	return d.UInt64()
}

// 32-bit integers are JSON numbers but protojson also accepts quoted strings.

func DecInt32(d *jx.Decoder) (int32, error) {
	if d.Next() == jx.String {
		s, err := d.Str()
		if err != nil {
			return 0, err
		}
		n, err := strconv.ParseInt(s, 10, 32)
		return int32(n), err
	}
	return d.Int32()
}

func DecUint32(d *jx.Decoder) (uint32, error) {
	if d.Next() == jx.String {
		s, err := d.Str()
		if err != nil {
			return 0, err
		}
		n, err := strconv.ParseUint(s, 10, 32)
		return uint32(n), err
	}
	return d.UInt32()
}

// Floats: NaN/Inf are emitted as the protojson string sentinels.

func EncFloat64(e *jx.Encoder, v float64) {
	switch {
	case math.IsNaN(v):
		e.Str("NaN")
	case math.IsInf(v, 1):
		e.Str("Infinity")
	case math.IsInf(v, -1):
		e.Str("-Infinity")
	default:
		e.Float64(v)
	}
}

func EncFloat32(e *jx.Encoder, v float32) { EncFloat64(e, float64(v)) }

func DecFloat64(d *jx.Decoder) (float64, error) {
	if d.Next() == jx.String {
		s, err := d.Str()
		if err != nil {
			return 0, err
		}
		switch s {
		case "NaN":
			return math.NaN(), nil
		case "Infinity":
			return math.Inf(1), nil
		case "-Infinity":
			return math.Inf(-1), nil
		default:
			return strconv.ParseFloat(s, 64)
		}
	}
	return d.Float64()
}

func DecFloat32(d *jx.Decoder) (float32, error) {
	v, err := DecFloat64(d)
	return float32(v), err
}

// Bytes: encoded as standard padded base64 (protojson output); decoded from
// standard or URL-safe alphabet, padded or not.

func EncBytes(e *jx.Encoder, v []byte) { e.Str(base64.StdEncoding.EncodeToString(v)) }

func DecBytes(d *jx.Decoder) ([]byte, error) {
	s, err := d.Str()
	if err != nil {
		return nil, err
	}
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, base64.CorruptInputError(0)
}
