# protoc-gen-go-jx

A `protoc` plugin that generates fast, **reflection-free** JSON codecs for
protobuf messages using [`github.com/go-faster/jx`](https://github.com/go-faster/jx).
Output is compatible with `google.golang.org/protobuf/encoding/protojson` at its
default options, but ~2–3× faster (see [Benchmarks](#benchmarks)).

## Install

```bash
go install github.com/gopherex/protoc-gen-go-jx@latest
```

This puts `protoc-gen-go-jx` on your `$PATH` (in `$(go env GOBIN)` or
`$(go env GOPATH)/bin` — make sure that directory is on `PATH`).

Add the runtime module to your project (generated code imports it):

```bash
go get github.com/gopherex/protoc-gen-go-jx
```

From source instead:

```bash
git clone https://github.com/gopherex/protoc-gen-go-jx
cd protoc-gen-go-jx
make build          # -> bin/protoc-gen-go-jx
```

## Usage

The plugin runs alongside `protoc-gen-go`: generate the standard `*.pb.go`
first, then `protoc-gen-go-jx` writes a `*.pb.jx.go` next to it in the same Go
package. For every message it generates four methods:

```go
func (m *T) Encode(e *jx.Encoder)            // streaming encode
func (m *T) Decode(d *jx.Decoder) error      // streaming decode
func (m *T) MarshalJSON() ([]byte, error)    // wraps Encode
func (m *T) UnmarshalJSON(b []byte) error    // wraps Decode
```

### With protoc

```bash
protoc \
  --go_out=. --go_opt=paths=source_relative \
  --go-jx_out=. --go-jx_opt=paths=source_relative \
  your.proto
```

### With buf

```yaml
# buf.gen.yaml
version: v2
plugins:
  - local: protoc-gen-go
    out: .
    opt: paths=source_relative
  - local: protoc-gen-go-jx
    out: .
    opt: paths=source_relative
```

### With easyp

Point at the built binary via `path` (see `example/easyp.yaml`):

```yaml
generate:
  plugins:
    - name: go
      out: .
      opts: { paths: source_relative }
    - path: ./bin/protoc-gen-go-jx
      out: .
      opts: { paths: source_relative }
```

### In Go

```go
import "github.com/go-faster/jx"

// Marshal
b, err := msg.MarshalJSON()            // or: encoding/json.Marshal(msg)

// Streaming encode
var e jx.Encoder
msg.Encode(&e)
out := e.Bytes()

// Unmarshal
err = msg.UnmarshalJSON(b)             // or: encoding/json.Unmarshal(b, msg)

// Streaming decode
if err := msg.Decode(jx.DecodeBytes(b)); err != nil { /* ... */ }
```

`Encode` is infallible by contract; an `Any` whose type cannot be resolved
degrades to a best-effort `{"@type": …}` rather than panicking.

## Why

`protojson` walks the message descriptor with reflection on every call.
`protoc-gen-go-jx` does that work once, at generation time, and emits direct
`jx` encode/decode code. You get protojson-shaped JSON, `encoding/json`-compatible
types, and a zero-reflection hot path.

## Compatibility

Encoded JSON matches protojson defaults:

| proto | JSON |
|---|---|
| `int32`/`uint32`/`fixed32`/`sfixed32`/`sint32` | number |
| `int64`/`uint64`/`fixed64`/`sfixed64`/`sint64` | **string** |
| `float`/`double` | number; `"NaN"`/`"Infinity"`/`"-Infinity"`; `-0` preserved |
| `bool` | `true`/`false` |
| `string` | string |
| `bytes` | standard padded base64 |
| `enum` | string name (unknown number → JSON number) |
| `message` | object |
| `repeated` | array · `map` | object (string keys) · `oneof` | flattened |
| well-known types | Timestamp (RFC3339), Duration (`"1.5s"`), wrappers (bare value), Struct/Value/ListValue (native JSON), Empty (`{}`), FieldMask (camel CSV), Any (`{"@type":…}`) |

Default-valued and unset fields are omitted; proto3 `optional` presence is
respected.

Decoding matches protojson defaults too:

- accepts both the lowerCamel JSON name **and** the original proto field name;
- accepts lenient scalars (enum as name or number, 64-bit ints as string or
  number, std or URL-safe base64);
- **errors** on an unknown field, a duplicate field key, or two keys for the
  same `oneof`.

Cross-package message fields are supported when that package is also
`protoc-gen-go-jx`-generated.

## Benchmarks

Generated `jx` codecs vs `protojson` on the same messages
(`go test -bench=. -benchmem ./example/golden/`, Go 1.26, Intel i5-14600K).
`ScalarTypes` is a flat scalar message; `Everything` nests every feature
(scalars, optionals, repeated, maps, oneof, all WKTs, recursion). Numbers vary
by machine — run it yourself.

**Marshal**

| message | codec | ns/op | B/op | allocs/op | speedup |
|---|---|--:|--:|--:|--:|
| ScalarTypes | jx | 837 | 1104 | 14 | **2.3×** |
| ScalarTypes | protojson | 1901 | 1474 | 26 | — |
| Everything | jx | 49 193 | 66 810 | 244 | **3.2×** |
| Everything | protojson | 158 605 | 122 210 | 1919 | — |

**Unmarshal**

| message | codec | ns/op | B/op | allocs/op | speedup |
|---|---|--:|--:|--:|--:|
| ScalarTypes | jx | 1760 | 1960 | 30 | **2.1×** |
| ScalarTypes | protojson | 3749 | 984 | 63 | — |
| Everything | jx | 106 720 | 101 880 | 1890 | **2.4×** |
| Everything | protojson | 259 432 | 90 513 | 3643 | — |

~2–3× faster with far fewer allocations (up to ~8× fewer on encode), since there
is no per-call descriptor reflection. The benchmark is
`example/golden/jx_bench_test.go`.

## Development

```bash
make build   # build bin/protoc-gen-go-jx
make gen     # regenerate example/golden via easyp
make test    # differential tests vs protojson
```

Repository layout:

- `main.go`, `generator/` — the plugin. `generator/{encode,decode,maps,wkt}.go`
  emit the per-message methods; `fieldinfo.go` classifies fields.
- `jxpb/` — the runtime imported by generated code: scalar helpers and the
  hand-written well-known-type codecs.
- `example/golden/` — `golden.proto` exercising every proto3 feature, plus
  `jx_diff_test.go`: a differential test asserting byte-for-byte parity with
  `protojson` and a decode round-trip for every message.

## Scope

protojson's configurable options (`UseProtoNames`, `UseEnumNumbers`,
`EmitUnpopulated`, …) are baked to their defaults and not yet exposed as
generation flags. proto2 semantics, extensions, and groups are out of scope.

## License

See [LICENSE](LICENSE).
