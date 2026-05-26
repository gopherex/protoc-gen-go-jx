package jxpb_test

import (
	"math"
	"testing"

	"github.com/go-faster/jx"
	"github.com/gopherex/protoc-gen-go-jx/jxpb"
)

func TestInt64String(t *testing.T) {
	var e jx.Encoder
	jxpb.EncInt64(&e, 9000000000)
	if got := string(e.Bytes()); got != `"9000000000"` {
		t.Fatalf("EncInt64 = %s", got)
	}
	for _, in := range []string{`"42"`, `42`} {
		v, err := jxpb.DecInt64(jx.DecodeStr(in))
		if err != nil || v != 42 {
			t.Fatalf("DecInt64(%s) = %d, %v", in, v, err)
		}
	}
}

func TestFloatSpecials(t *testing.T) {
	cases := map[float64]string{math.NaN(): `"NaN"`, math.Inf(1): `"Infinity"`, math.Inf(-1): `"-Infinity"`, 1.5: `1.5`}
	for v, want := range cases {
		var e jx.Encoder
		jxpb.EncFloat64(&e, v)
		if got := string(e.Bytes()); got != want {
			t.Fatalf("EncFloat64(%v) = %s want %s", v, got, want)
		}
	}
	v, err := jxpb.DecFloat64(jx.DecodeStr(`"NaN"`))
	if err != nil || !math.IsNaN(v) {
		t.Fatalf("DecFloat64 NaN = %v, %v", v, err)
	}
}

func TestBytesStd(t *testing.T) {
	var e jx.Encoder
	jxpb.EncBytes(&e, []byte("hi"))
	if got := string(e.Bytes()); got != `"aGk="` {
		t.Fatalf("EncBytes = %s", got)
	}
	for _, in := range []string{`"aGk="`, `"aGk"`} {
		b, err := jxpb.DecBytes(jx.DecodeStr(in))
		if err != nil || string(b) != "hi" {
			t.Fatalf("DecBytes(%s) = %q, %v", in, b, err)
		}
	}
}
