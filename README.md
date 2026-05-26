# protoc-gen-go-jx

A `protoc` plugin that generates fast, reflection-free JSON codecs for protobuf
messages using [`github.com/go-faster/jx`](https://github.com/go-faster/jx).

For every message it generates:

```go
func (m *T) Encode(e *jx.Encoder)
func (m *T) Decode(d *jx.Decoder) error
func (m *T) MarshalJSON() ([]byte, error)
func (m *T) UnmarshalJSON(b []byte) error
```

The output is compatible with
`google.golang.org/protobuf/encoding/protojson` at its **default** options —
but without runtime reflection on the encode/decode path.

## Compatibility

Generated JSON matches protojson defaults:

- field names in lowerCamelCase (`jsonName`);
- 64-bit integers (`int64`/`uint64`/`fixed64`/…) as JSON **strings**;
- `float`/`double` as numbers, with `"NaN"`/`"Infinity"`/`"-Infinity"` for specials;
- `bytes` as standard padded base64;
- enums as their string name (unknown numbers as a JSON number);
- default/empty fields omitted; `optional` presence respected;
- maps as JSON objects (keys always strings); `oneof` flattened;
- **all** well-known types: Timestamp, Duration, the 9 wrappers, Empty, Struct,
  Value, ListValue, FieldMask, and Any.

Decoding is strict: an unknown JSON field is an error. It also accepts the
lenient inputs protojson accepts (enum as name or number, 64-bit ints as string
or number, std or URL-safe base64).

`Encode` is infallible by contract; `Any` resolution failures degrade to a
best-effort `{"@type": …}` rather than panicking.

## Layout

- `main.go`, `generator/` — the plugin (walks the proto file, emits
  `<prefix>.pb.jx.go` into the same Go package as the `.pb.go`).
- `jxpb/` — the small runtime imported by generated code: scalar helpers and
  the hand-written well-known-type codecs.
- `example/golden/` — `golden.proto` exercising every proto feature, plus a
  differential test (`jx_diff_test.go`) that asserts byte-for-byte parity with
  `protojson` and a decode round-trip for every message.

## Usage

```bash
make build   # builds bin/protoc-gen-go-jx
make gen     # regenerates example/golden via easyp
make test    # runs the differential tests vs protojson
```

With `protoc` directly:

```bash
protoc --go-jx_out=. --go-jx_opt=paths=source_relative your.proto
```

Or via [easyp](https://github.com/easyp-tech/easyp), pointing at the built binary
(see `example/easyp.yaml`):

```yaml
generate:
  plugins:
    - path: ../bin/protoc-gen-go-jx
      out: .
      opts:
        paths: source_relative
```

Generated code imports the runtime package
`github.com/gopherex/protoc-gen-go-jx/jxpb`.

## Scope

protojson's configurable options (`UseProtoNames`, `UseEnumNumbers`,
`EmitUnpopulated`, …) are baked to their defaults; they are not yet exposed as
generation flags. proto2, extensions, and groups are out of scope.
