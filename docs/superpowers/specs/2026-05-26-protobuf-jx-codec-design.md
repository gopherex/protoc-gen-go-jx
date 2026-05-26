# protoc-gen-go-jx — protobuf ↔ JSON codec via go-faster/jx

Date: 2026-05-26
Status: approved (design)

## Goal

A protoc plugin that, for every protobuf message, generates static Go methods
that encode/decode the message to/from JSON using `github.com/go-faster/jx`,
with **no runtime reflection**. The generated JSON is byte-for-byte compatible
with `google.golang.org/protobuf/encoding/protojson` at its **default** options.

Required generated surface per message:

```go
func (m *T) Encode(e *jx.Encoder)            // infallible
func (m *T) Decode(d *jx.Decoder) error
func (m *T) MarshalJSON() ([]byte, error)    // wraps Encode
func (m *T) UnmarshalJSON(b []byte) error    // wraps Decode
```

Reference layout: `github.com/yaroher/protoc-gen-ogen` (main.go + `generator`
package + protoc/easyp wiring + `example/golden`).

## Why static codegen

`protojson` works entirely at runtime: it walks `m.ProtoReflect()` over the
descriptor metadata embedded in `.pb.go` and tunes behaviour via a
`MarshalOptions`/`UnmarshalOptions` struct passed at call time. We do the same
work at **generation time**, reading the same descriptor info through
`protogen` (`field.Desc.JSONName()`, `.Kind()`, `.Cardinality()`, …) and
emitting straight-line `jx` calls. No reflection at runtime → fast, allocation-light.

Cost: protojson's options are taken per-call; our methods take no options
argument, so the policy is **baked at generation time**. For v1 we bake the
protojson defaults and expose no knobs (deferred).

## Baked policy (protojson defaults)

| Option | Baked value |
|---|---|
| UseProtoNames | false → camelCase `JSONName()` keys |
| UseEnumNumbers | false → enum as string name |
| EmitUnpopulated / EmitDefaultValues | false → omit default/empty fields |
| AllowPartial | n/a (proto3 has no required fields) |
| DiscardUnknown (decode) | false → **unknown JSON key is an error** |

## Architecture (Approach B: thin generated code + runtime helper package)

```
protoc-gen-go-jx/
├── main.go                 # protogen.Options.Run, FEATURE_PROTO3_OPTIONAL
├── generator/
│   ├── generator.go        # File→Message→Field walk, emits <prefix>.pb.jx.go
│   ├── encode.go           # Encode body generation
│   ├── decode.go           # Decode body generation
│   └── types.go            # kind→jx mapping, WKT detection, JSONName helpers
├── jxpb/                   # hand-written runtime, imported by generated code
│   ├── scalars.go          # int64-as-string, base64, float NaN/Inf
│   ├── enum.go             # name<->number lookup helpers
│   └── wkt.go              # well-known-type codecs
├── example/golden/         # golden.proto + generated golden.pb.jx.go (test target)
└── Makefile, easyp.yaml
```

- One generated file per `.proto`: `<prefix>.pb.jx.go`, in the **same package**
  as the `.pb.go`, `paths=source_relative`. Methods are on the pb structs.
- Generated code imports `github.com/gopherex/protoc-gen-go-jx/jxpb`. `jxpb`
  does not import generated packages → no cycle.

### Why a runtime package

The hard, reusable logic (WKT codecs, base64, int64-string, float specials,
enum lookup) is written and tested **once** in `jxpb` instead of being inlined
into every generated file. Keeps generated output small.

## Format mapping (protojson defaults)

| proto type | JSON | omitted when |
|---|---|---|
| int32 / uint32 / fixed32 / sfixed32 / sint32 | number | == 0 |
| **int64 / uint64 / fixed64 / sfixed64 / sint64** | **string** | == 0 |
| double / float | number; NaN/Inf → `"NaN"` / `"Infinity"` / `"-Infinity"` | == 0 |
| bool | `true` / `false` | false |
| string | string | "" |
| bytes | base64 **standard** (padded) string | len == 0 |
| enum | string name; unknown number → JSON number | == 0 |
| message | object | nil |
| optional scalar (explicit presence) | as scalar | nil pointer (unset) |
| repeated | array | empty |
| map | object — key always serialized as string | empty |
| oneof | flattened; only the set member emitted | unset |

- Object keys use `JSONName()` (camelCase). Fields emitted in declaration order.
- Map keys: int/bool keys rendered as JSON strings (`"true"`, `"123"`).
- 64-bit map values follow the int64-string rule; enum map values use names.

### Well-known types (all in v1)

| WKT | JSON form |
|---|---|
| Timestamp | RFC3339 string, UTC `Z`, 0/3/6/9 fractional digits |
| Duration | seconds string with fraction + `s`, e.g. `"1.5s"`, `"-3s"` |
| DoubleValue … BytesValue (9 wrappers) | bare scalar value (Int64Value → string, BytesValue → base64, …) |
| Empty | `{}` |
| Struct | JSON object (string → Value) |
| Value | any JSON value (null/number/string/bool/object/array) |
| ListValue | JSON array of Value |
| Any | `{"@type": <typeURL>, …inlined fields}`; WKT-valued Any → `{"@type":…, "value": <wkt-json>}` |
| FieldMask | single comma-joined string, each path snake→lowerCamel |

WKT fields in user messages appear as Go types from
`google.golang.org/protobuf/types/known/*pb`. Generated code detects them by
full proto name and calls the matching `jxpb` codec keyed on that concrete Go
type.

## Generated method behaviour

- `Encode(*jx.Encoder)` is **infallible** by contract (jx buffers, no error
  return). The only failable path is `Any` resolution; on unresolved type it
  emits a best-effort `{"@type": <url>}` with no value — never panics.
- `Decode(*jx.Decoder) error`:
  - unknown JSON key → error (protojson default).
  - enum accepts string name **or** number.
  - 64-bit ints accept JSON string **or** number.
  - bytes accept standard **or** URL-safe base64, padded or not.
  - JSON `null` → field left at default (skipped).
  - errors wrapped with `fmt.Errorf("...%w", err)` including the field path.
- `MarshalJSON` = `var e jx.Encoder; m.Encode(&e); return e.Bytes(), nil`.
- `UnmarshalJSON` = decode bytes via `jx` then `m.Decode(d)`.

### Recursion / enum reuse

- Recursive messages (TreeNode, MutualA/B, Everything.recursive) work naturally:
  generated methods call each other.
- Enums reuse the `<Enum>_name` / `<Enum>_value` maps already generated in
  `.pb.go`; we do not emit duplicate maps.

## Any handling

`Any` carries a dynamic embedded type, the one case that genuinely needs a
resolver. Strategy:

1. Resolve the type URL via `protoregistry.GlobalTypes`.
2. Encode: if resolved message has our jx codec, inline it under `@type`;
   else fall back to `protojson` for the value; if unresolved, best-effort
   `{"@type": url}`.
3. Decode: read `@type`, resolve, decode the rest into the resolved message.

This is the one place a `protojson`/reflection fallback is acceptable.

## Dependencies

- `github.com/go-faster/jx`
- `google.golang.org/protobuf` (already present) — protogen, known types,
  protoregistry, protojson (Any fallback only)
- stdlib only for error wrapping (`fmt`, `errors`).

## Testing

Primary check is **differential parity with protojson**:

1. A test-only, reflection-based populator fills every field of `Everything`
   (and each top-level message) with non-default values across all branches.
2. Assert our `MarshalJSON` output ≡ `protojson.Marshal` output, compared
   semantically (parse both to `any`, deep-equal — independent of key order).
3. Decode our JSON back and assert `proto.Equal(original, decoded)`.
4. Per-WKT unit tests in `jxpb` (boundary values: epoch, negative duration,
   NaN/Inf, empty bytes, unknown enum number, nested Any).

`example/golden/golden.proto` already exercises scalars, optionals, repeated,
maps (all key types), oneof, nested, recursion, reserved, and every WKT.

## Build / wiring

- `Makefile`: build `protoc-gen-go-jx` and `protoc-gen-go` into `bin/`, then
  run generation. Primary path mirrors the ogen Makefile's direct `protoc`
  invocation (proven); easyp wiring via local plugin `path:` kept consistent
  with the existing `example/easyp.yaml`.
- Plugin protoc name: `go-jx` (`--go-jx_out`, `--go-jx_opt=paths=source_relative`).
- `example/easyp.yaml`: add a `go-jx` plugin entry pointing at the built binary.

## Out of scope (v1)

- Configurable options (UseProtoNames / UseEnumNumbers / Emit* / DiscardUnknown).
  Deferred; would map protojson option names to gen-time plugin params.
- proto2 semantics, extensions, groups, MessageSet.
- Multiline/indented output, deterministic map ordering.

### Known decode-leniency gaps vs protojson (v1)

The encode side is byte-parity with protojson defaults (verified by the
differential suite). These input-side behaviours differ from protojson and are
deferred — they do not affect encoding, nor round-tripping our own/standard
camelCase output:

- **Original (snake_case) field names are rejected on decode.** protojson
  accepts both the lowerCamel JSON name and the original proto name; we only
  accept the JSON name (unknown key → error). Fix would emit a second `case`
  per field.
- **Duplicate JSON keys** are last-wins; protojson errors.
- **Two members of the same oneof set in one object** silently overwrite;
  protojson errors.
- **Cross-package, non-WKT message fields are silently omitted** (the
  `localPath` guard). No such field exists in `golden.proto`; supporting them
  would require detecting whether the other package's type has generated
  `Encode`/`Decode` methods.
- **Negative zero** (`-0.0`) is omitted on encode (treated as default `0`);
  protojson emits `-0`. Round-trips correctly under `proto.Equal`.
