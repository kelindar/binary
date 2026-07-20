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
- Optional subpackages for **sorted**, **unsafe**, and **nocopy** typed slices when you need smaller payloads or lower decode cost.

## Documentation

Variable-sized values are prefixed with a varint-encoded size and encoded recursively. The format is intentionally not versioned or cross-language — this is for efficient exchange of known Go types between systems you control.

- [Quick Start](#quick-start)
- [Streaming Encode and Decode](#streaming-encode-and-decode)
- [Skipping Fields](#skipping-fields)
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
binary/enc           139.2 ns     7.2M         2
binary/enc-to        104.7 ns     9.5M         0
binary/dec           175.6 ns     5.7M         5
binary/map-enc       9.5 µs       105.2K       210
binary/map-dec       14.4 µs      69.4K        511
binary/slice-enc     9.4 µs       106.5K       9
binary/slice-dec     14.9 µs      67.2K        502
binary/nest-enc      5.0 µs       201.9K       14
binary/nest-dec      8.3 µs       120.8K       271
binary/bytes-enc     835.2 ns     1.2M         3
binary/bytes-dec     821.2 ns     1.2M         2
binary/u64-enc       75.3 µs      13.3K        11
binary/u64-dec       58.4 µs      17.1K        2
binary/reuse-enc     113.2 ns     8.8M         0
binary/stream-dec    240.8 ns     4.2M         5
nocopy/str-enc       109.7 ns     9.1M         3
nocopy/str-dec       39.2 ns      25.5M        0
nocopy/dict-enc      185.7 ns     5.4M         2
nocopy/dict-dec      154.5 ns     6.5M         2
nocopy/bmap-enc      313.3 ns     3.2M         5
nocopy/bmap-dec      160.9 ns     6.2M         2
nocopy/hmap-enc      307.6 ns     3.3M         5
nocopy/hmap-dec      146.6 ns     6.8M         2
nocopy/bytes-enc     862.0 ns     1.2M         3
nocopy/bytes-dec     41.6 ns      24.0M        0
nocopy/u64-enc       5.3 µs       187.6K       3
nocopy/u64-dec       39.8 ns      25.1M        0
nocopy/col-enc       524.3 ns     1.9M         9
nocopy/col-dec       532.2 ns     1.9M         8
nocopy/struct-enc    162.9 ns     6.1M         3
nocopy/struct-dec    82.7 ns      12.1M        0
sorted/i32-enc       78.9 µs      12.7K        5
sorted/i32-dec       551.8 µs     1.8K         29.8K
sorted/u32-enc       76.2 µs      13.1K        5
sorted/u32-dec       543.5 µs     1.8K         29.8K
sorted/ts-enc        52.5 µs      19.0K        6
sorted/ts-dec        21.8 µs      45.8K        2
sorted/tsz-enc       126.4 µs     7.9K         7
sorted/tsz-dec       119.3 µs     8.4K         3
sorted/tcz-enc       85.9 µs      11.6K        6
sorted/tcz-dec       76.1 µs      13.1K        3
unsafe/u64-enc       459.8 ns     2.2M         3
unsafe/u64-dec       450.1 ns     2.2M         2
```

## Disclaimer

This is **not** a replacement for JSON, protobuf, or other versioned interchange formats. The codec does not maintain schema evolution or cross-language compatibility. Use it to exchange binary data of a known format between Go services where you control both ends.

## Contributing

Contributions are welcome — open a pull request and we will review it as quickly as we can. This library is maintained by [Roman Atachiants](https://www.linkedin.com/in/atachiants/).

## License

Binary is licensed under the [MIT License](LICENSE).
