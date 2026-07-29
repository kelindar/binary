<p align="center">
<img src="https://img.shields.io/github/go-mod/go-version/kelindar/binary" alt="Go Version">
<a href="https://pkg.go.dev/github.com/kelindar/binary"><img src="https://pkg.go.dev/badge/github.com/kelindar/binary" alt="PkgGoDev"></a>
<a href="https://goreportcard.com/report/github.com/kelindar/binary"><img src="https://goreportcard.com/badge/github.com/kelindar/binary" alt="Go Report Card"></a>
<a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
</p>

## Fast Binary Serializer for Go

This package contains a **high-performance binary serializer** for Go that encodes and decodes arbitrary data structures of variable size. It is designed as a simple `json.Marshal` / `json.Unmarshal` drop-in for cases where you control both ends and want compact, fast payloads — for example inter-broker message encoding in [emitter](https://github.com/emitter-io/emitter).

## Features

- **Simple API** that mirrors `encoding/json` with `Marshal`, `Unmarshal`, and `MarshalTo`.
- **Compact payloads** using varint encoding for integers and size-prefixed variable-length values.
- **Zero-allocation encoding** path via `MarshalTo` writing directly to an `io.Writer`.
- **Reflect-based** support for structs, maps, slices, arrays, pointers, and nested types.
- **Fast paths** for `[]byte` and other common slice types.
- **Custom serialization** via `encoding.BinaryMarshaler` / `BinaryUnmarshaler` or a full `Codec` through `GetBinaryCodec`.
- **Field skipping** with the `binary:"-"` struct tag.
- **Tagged unions** (oneof / versioning) via `binary:"N,union"` tags on pointer fields.
- Optional subpackages for **sorted**, **unsafe**, and **nocopy** typed slices when you need smaller payloads or lower decode cost.

## Documentation

Variable-sized values are prefixed with a varint-encoded size and encoded recursively. The default sequential format is not self-describing or automatically versioned; tagged unions provide explicit, user-managed versioning. The format is Go-specific and intended for systems you control.

- [Quick Start](#quick-start)
- [Streaming Encode and Decode](#streaming-encode-and-decode)
- [Skipping Fields](#skipping-fields)
- [Tagged Unions](#tagged-unions)
- [Custom Serialization](#custom-serialization)
- [Typed Slice Subpackages](#typed-slice-subpackages)
- [Benchmarks](#benchmarks)
- [Disclaimer](#disclaimer)
- [Contributing](#contributing)
- [License](#license)

## Quick Start

Define a message and marshal it the same way you would with JSON:

```go
type message struct {
	Name      string
	Timestamp int64
	Payload   []byte
	Ssid      []uint32
}

v := &message{
	Name:      "Roman",
	Timestamp: 1242345235,
	Payload:   []byte("hi"),
	Ssid:      []uint32{1, 2, 3},
}

encoded, err := binary.Marshal(v)
if err != nil {
	panic(err)
}

var out message
err = binary.Unmarshal(encoded, &out)
```

## Streaming Encode and Decode

For hot paths, write into a reused buffer with `MarshalTo`, or use `Encoder` / `Decoder` directly against an `io.Writer` / `io.Reader`:

```go
var buf bytes.Buffer
if err := binary.MarshalTo(v, &buf); err != nil {
	panic(err)
}

dec := binary.NewDecoder(&buf)
var out message
if err := dec.Decode(&out); err != nil {
	panic(err)
}
```

## Skipping Fields

Fields tagged with `binary:"-"` are ignored during encode and decode. Useful for locks, caches, or derived state:

```go
type Cache struct {
	mu    sync.Mutex `binary:"-"`
	Key   string
	Value []byte
}
```

## Tagged Unions

A struct whose included fields all use `binary:"N,union"` tags (`N` in `1..255`) is encoded as a **tagged union** (oneof / versioning):

```text
uvarint(tag) + uvarint(len) + body
```

Arms must be pointers and tags must be unique. Exactly one may be non-nil on encode; all nil writes tag `0` with an empty body. Unknown tags are skipped on decode (forward-compatible versioning).

### Oneof

```go
type Payload struct {
	Text  *TextPayload  `binary:"1,union"`
	Image *ImagePayload `binary:"2,union"`
}

encoded, err := binary.Marshal(Payload{Text: &TextPayload{Msg: "hi"}})

var out Payload
err = binary.Unmarshal(encoded, &out)
// out.Text != nil
```

### Versioning

Use one arm per schema version. Older readers ignore newer tags; newer readers still decode older payloads:

```go
type DocV1 struct {
	Title string
}

type DocV2 struct {
	Title string
	Body  string
}

type Doc struct {
	V1 *DocV1 `binary:"1,union"`
	V2 *DocV2 `binary:"2,union"`
}

var ErrUnknownVersion = errors.New("unknown document version")

// Current returns the latest representation, migrating older versions explicitly.
func (d Doc) Current() (DocV2, error) {
	switch {
	case d.V2 != nil:
		return *d.V2, nil
	case d.V1 != nil:
		return DocV2{Title: d.V1.Title}, nil
	default:
		return DocV2{}, ErrUnknownVersion
	}
}

// Writer on the new version
encoded, err := binary.Marshal(Doc{V2: &DocV2{Title: "hi", Body: "world"}})

// Reader that only knows V1 still succeeds: unknown tag 2 is skipped
var legacy struct {
	V1 *DocV1 `binary:"1,union"`
}
err = binary.Unmarshal(encoded, &legacy)
// legacy.V1 == nil — body was skipped, no error

// Reader that knows both picks the matching arm
var current Doc
err = binary.Unmarshal(encoded, &current)
latest, err := current.Current()
```

Nest a union inside a normal sequential struct as a single field. For hand-rolled codecs, use `Encoder.WriteTagged` / `Decoder.ReadTagged` with the same framing.

## Custom Serialization

By default, values are encoded through reflection. You can override that for a type in two ways, checked in this order:

1. **`GetBinaryCodec()`** — return a `binary.Codec` for full control over the wire format (no extra length prefix).
2. **`MarshalBinary` / `UnmarshalBinary`** — the standard `encoding.BinaryMarshaler` / `BinaryUnmarshaler` pair; the package length-prefixes the returned bytes for you.

Use `MarshalBinary` when you already have a `[]byte` representation. Use `GetBinaryCodec` when you want to stream fields through the encoder without an intermediate buffer.

### Option 1: `MarshalBinary` and `UnmarshalBinary`

This matches `encoding.BinaryMarshaler` and `encoding.BinaryUnmarshaler`. On encode, the returned slice is written as `uvarint(length) + bytes`. On decode, that framed blob is passed to `UnmarshalBinary`:

```go
type CompactHeader struct {
	Code uint8
}

func (h CompactHeader) MarshalBinary() ([]byte, error) {
	return []byte{h.Code}, nil
}

func (h *CompactHeader) UnmarshalBinary(data []byte) error {
	if len(data) != 1 {
		return fmt.Errorf("CompactHeader: want 1 byte, got %d", len(data))
	}
	h.Code = data[0]
	return nil
}

encoded, err := binary.Marshal(CompactHeader{Code: 0x13})
// encoded == []byte{0x01, 0x13}  // length + payload

var out CompactHeader
err = binary.Unmarshal(encoded, &out)
```

This is enough for most custom types, including anything that already implements the standard library interfaces (for example `time.Time`).

### Option 2: `GetBinaryCodec`

For tighter packing or to avoid the intermediate `[]byte`, implement `GetBinaryCodec` on a **pointer receiver**. It must return a `binary.Codec`:

```go
type Codec interface {
	EncodeTo(*Encoder, reflect.Value) error
	DecodeTo(*Decoder, reflect.Value) error
}
```

Example: encode a 2D point as two raw `float64` values with no struct field metadata:

```go
type Point struct {
	X float64
	Y float64
}

func (p *Point) GetBinaryCodec() binary.Codec {
	return pointCodec{}
}

type pointCodec struct{}

func (pointCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) error {
	p := rv.Interface().(Point)
	e.WriteFloat64(p.X)
	e.WriteFloat64(p.Y)
	return nil
}

func (pointCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) error {
	x, err := d.ReadFloat64()
	if err != nil {
		return err
	}
	y, err := d.ReadFloat64()
	if err != nil {
		return err
	}
	rv.Set(reflect.ValueOf(Point{X: x, Y: y}))
	return nil
}
```

Because `GetBinaryCodec` takes priority, a type should not also rely on `MarshalBinary` for the same purpose — pick one approach per type.

Nested use works automatically: if a struct field's type implements either customization, that field uses the custom path while the rest of the struct uses the default reflect codecs.

```go
type Packet struct {
	ID   uint64
	At   time.Time // uses time.Time's BinaryMarshaler
	Pos  Point     // uses GetBinaryCodec above
	Raw  []byte
}
```

## Typed Slice Subpackages

Optional helpers live in subpackages when the default reflect path is not enough:

| Package | Purpose |
|---------|---------|
| [`sorted`](./sorted) | Delta-encoded sorted integer / timestamp slices for smaller wire size |
| [`unsafe`](./unsafe) | Memory-cast numeric slices (faster, not portable across endianness) |
| [`nocopy`](./nocopy) | Like `unsafe`, but decode reuses the input buffer (zero-copy; lifetime tied to the buffer) |

```go
import (
	"github.com/kelindar/binary"
	"github.com/kelindar/binary/sorted"
)

v := sorted.Int32s{4, 5, 6, 1, 2, 3}
encoded, err := binary.Marshal(&v)

var out sorted.Int32s
err = binary.Unmarshal(encoded, &out)
```

See each subpackage README for trade-offs and warnings.

## Benchmarks

Numbers from `bench/`. Run locally with:

```bash
cd bench && go run .
```

```
name                 time/op      ops/s        allocs/op
-------------------- ------------ ------------ ------------
binary/enc           122.6 ns     8.2M         2
binary/enc-to        89.0 ns      11.2M        0
binary/dec           90.1 ns      11.1M        1
binary/map-enc       8.5 µs       117.4K       209
binary/map-dec       12.8 µs      78.1K        500
binary/slice-enc     8.0 µs       124.3K       9
binary/slice-dec     6.6 µs       151.4K       100
binary/nest-enc      4.2 µs       236.1K       13
binary/nest-dec      3.7 µs       270.0K       63
binary/bytes-enc     812.2 ns     1.2M         3
binary/bytes-dec     172.5 ns     5.8M         0
binary/u64-enc       42.4 µs      23.6K        3
binary/u64-dec       57.6 µs      17.4K        0
binary/reuse-enc     95.7 ns      10.5M        0
binary/stream-dec    147.4 ns     6.8M         2
binary/trace-enc     12.6 µs      79.2K        9
binary/trace-dec     13.0 µs      76.7K        640
union/enc            196.6 ns     5.1M         3
union/dec            100.7 ns     9.9M         0
union/nest-enc       213.3 ns     4.7M         3
union/nest-dec       110.4 ns     9.1M         0
union/reuse-enc      98.3 ns      10.2M        0
union/stream-dec     132.9 ns     7.5M         1
nocopy/str-enc       107.6 ns     9.3M         3
nocopy/str-dec       34.8 ns      28.8M        0
nocopy/dict-enc      183.7 ns     5.4M         2
nocopy/dict-dec      155.5 ns     6.4M         2
nocopy/bmap-enc      322.4 ns     3.1M         5
nocopy/bmap-dec      161.9 ns     6.2M         2
nocopy/hmap-enc      320.1 ns     3.1M         5
nocopy/hmap-dec      143.9 ns     7.0M         2
nocopy/bytes-enc     841.1 ns     1.2M         3
nocopy/bytes-dec     35.8 ns      28.0M        0
nocopy/u64-enc       5.4 µs       186.9K       3
nocopy/u64-dec       32.0 ns      31.2M        0
nocopy/col-enc       504.2 ns     2.0M         8
nocopy/col-dec       359.0 ns     2.8M         6
nocopy/struct-enc    161.0 ns     6.2M         3
nocopy/struct-dec    72.4 ns      13.8M        0
sorted/i32-enc       81.7 µs      12.2K        5
sorted/i32-dec       57.6 µs      17.4K        0
sorted/u32-enc       78.8 µs      12.7K        5
sorted/u32-dec       51.3 µs      19.5K        0
sorted/ts-enc        51.4 µs      19.4K        6
sorted/ts-dec        22.8 µs      43.9K        2
sorted/tsz-enc       129.7 µs     7.7K         7
sorted/tsz-dec       132.0 µs     7.6K         3
sorted/tcz-enc       86.9 µs      11.5K        6
sorted/tcz-dec       79.2 µs      12.6K        3
unsafe/u64-enc       483.4 ns     2.1M         3
unsafe/u64-dec       463.0 ns     2.2M         2
```

## Disclaimer

This is **not** a replacement for JSON, protobuf, or other versioned interchange formats. The codec does not maintain schema evolution or cross-language compatibility. Use it to exchange binary data of a known format between Go services where you control both ends.

## Contributing

Contributions are welcome — open a pull request and we will review it as quickly as we can. This library is maintained by [Roman Atachiants](https://www.linkedin.com/in/atachiants/).

## License

Binary is licensed under the [MIT License](LICENSE).
