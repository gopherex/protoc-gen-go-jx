package jxpb_test

import (
	"testing"
	"time"

	"github.com/go-faster/jx"
	"github.com/gopherex/protoc-gen-go-jx/jxpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestTimestamp(t *testing.T) {
	ts := timestamppb.New(time.Unix(1700000000, 500000000).UTC())
	var e jx.Encoder
	jxpb.EncTimestamp(&e, ts)
	want := `"2023-11-14T22:13:20.500Z"`
	if got := string(e.Bytes()); got != want {
		t.Fatalf("EncTimestamp = %s want %s", got, want)
	}
	out := &timestamppb.Timestamp{}
	if err := jxpb.DecTimestamp(jx.DecodeStr(string(e.Bytes())), out); err != nil {
		t.Fatal(err)
	}
	if out.Seconds != ts.Seconds || out.Nanos != ts.Nanos {
		t.Fatalf("DecTimestamp = %v want %v", out, ts)
	}
}

func TestDuration(t *testing.T) {
	d := &durationpb.Duration{Seconds: 1, Nanos: 500000000}
	var e jx.Encoder
	jxpb.EncDuration(&e, d)
	if got := string(e.Bytes()); got != `"1.500s"` {
		t.Fatalf("EncDuration = %s", got)
	}
	out := &durationpb.Duration{}
	if err := jxpb.DecDuration(jx.DecodeStr(`"1.5s"`), out); err != nil {
		t.Fatal(err)
	}
	if out.Seconds != 1 || out.Nanos != 500000000 {
		t.Fatalf("DecDuration = %v", out)
	}
}

func TestInt64Wrapper(t *testing.T) {
	var e jx.Encoder
	jxpb.EncInt64Value(&e, &wrapperspb.Int64Value{Value: 42})
	if got := string(e.Bytes()); got != `"42"` {
		t.Fatalf("EncInt64Value = %s", got)
	}
}
