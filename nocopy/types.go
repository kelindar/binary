// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package nocopy

import (
	"reflect"
	"unsafe"

	"github.com/kelindar/binary"
)

// ------------------------------------------------------------------------------

// String represents a type serialized in an unsafe, non portable manner. Moreover, when
// decoding it simply reuses the underlying byte array to store the data and does not
// perform a memory copy. This can be dangerous in many cases, be careful how this is used.
type String string

// GetBinaryCodec retrieves a custom binary codec.
func (s *String) GetBinaryCodec() binary.Codec {
	return new(stringCodec)
}

// ------------------------------------------------------------------------------

// Bytes represents a type serialized in an unsafe, non portable manner. Moreover, when
// decoding it simply reuses the underlying byte array to store the data and does not
// perform a memory copy. This can be dangerous in many cases, be careful how this is used.
type Bytes []byte

// GetBinaryCodec retrieves a custom binary codec.
func (s *Bytes) GetBinaryCodec() binary.Codec {
	return new(byteSliceCodec)
}

// ------------------------------------------------------------------------------

// Bools represents a type serialized in an unsafe, non portable manner. Moreover, when
// decoding it simply reuses the underlying byte array to store the data and does not
// perform a memory copy. This can be dangerous in many cases, be careful how this is used.
type Bools []bool

// GetBinaryCodec retrieves a custom binary codec.
func (s *Bools) GetBinaryCodec() binary.Codec {
	return new(boolSliceCodec)
}

// ------------------------------------------------------------------------------

// Uint16s represents a slice serialized in an unsafe, non portable manner.
type Uint16s []uint16

func (s Uint16s) Len() int           { return len(s) }
func (s Uint16s) Less(i, j int) bool { return s[i] < s[j] }
func (s Uint16s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint16s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeFor[Uint16s](),
		sizeOfInt: 2,
	}
}

// ------------------------------------------------------------------------------

// Int16s represents a slice serialized in an unsafe, non portable manner.
type Int16s []int16

func (s Int16s) Len() int           { return len(s) }
func (s Int16s) Less(i, j int) bool { return s[i] < s[j] }
func (s Int16s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int16s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeFor[Int16s](),
		sizeOfInt: 2,
	}
}

// ------------------------------------------------------------------------------

// Uint32s represents a slice serialized in an unsafe, non portable manner.
type Uint32s []uint32

func (s Uint32s) Len() int           { return len(s) }
func (s Uint32s) Less(i, j int) bool { return s[i] < s[j] }
func (s Uint32s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint32s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeFor[Uint32s](),
		sizeOfInt: 4,
	}
}

// ------------------------------------------------------------------------------

// Int32s represents a slice serialized in an unsafe, non portable manner.
type Int32s []int32

func (s Int32s) Len() int           { return len(s) }
func (s Int32s) Less(i, j int) bool { return s[i] < s[j] }
func (s Int32s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int32s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeFor[Int32s](),
		sizeOfInt: 4,
	}
}

// ------------------------------------------------------------------------------

// Uint64s represents a slice serialized in an unsafe, non portable manner.
type Uint64s []uint64

func (s Uint64s) Len() int           { return len(s) }
func (s Uint64s) Less(i, j int) bool { return s[i] < s[j] }
func (s Uint64s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint64s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeFor[Uint64s](),
		sizeOfInt: 8,
	}
}

// ------------------------------------------------------------------------------

// Int64s represents a slice serialized in an unsafe, non portable manner.
type Int64s []int64

func (s Int64s) Len() int           { return len(s) }
func (s Int64s) Less(i, j int) bool { return s[i] < s[j] }
func (s Int64s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int64s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeFor[Int64s](),
		sizeOfInt: 8,
	}
}

// ------------------------------------------------------------------------------

// Float32s represents a slice serialized in an unsafe, non portable manner.
type Float32s []float32

func (s Float32s) Len() int           { return len(s) }
func (s Float32s) Less(i, j int) bool { return s[i] < s[j] }
func (s Float32s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Float32s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeFor[Float32s](),
		sizeOfInt: 4,
	}
}

// ------------------------------------------------------------------------------

// Float64s represents a slice serialized in an unsafe, non portable manner.
type Float64s []float64

func (s Float64s) Len() int           { return len(s) }
func (s Float64s) Less(i, j int) bool { return s[i] < s[j] }
func (s Float64s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Float64s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeFor[Float64s](),
		sizeOfInt: 8,
	}
}

// ------------------------------------------------------------------------------

// Dictionary represents a map where both keys and values are strings. It is
// serialized in an unsafe, non portable manner.
type Dictionary map[string]string

// GetBinaryCodec retrieves a custom binary codec.
func (d *Dictionary) GetBinaryCodec() binary.Codec {
	return new(dictionaryCodec)
}

// ------------------------------------------------------------------------------

// ByteMap represents a map where keys are strings but the values are slices of
// bytes. It is encoded in an unsafe, non portable mapper.
type ByteMap map[string][]byte

// GetBinaryCodec retrieves a custom binary codec.
func (d *ByteMap) GetBinaryCodec() binary.Codec {
	return new(byteMapCodec)
}

// ------------------------------------------------------------------------------

// HashMap represents a map where keys are uint64 but the values are slices of
// bytes. It is encoded in an unsafe, non portable mapper.
type HashMap map[uint64][]byte

// GetBinaryCodec retrieves a custom binary codec.
func (d *HashMap) GetBinaryCodec() binary.Codec {
	return new(hashMapCodec)
}

// ------------------------------------------------------------------------------

type integerSliceCodec struct {
	sliceType reflect.Type
	sizeOfInt int
}

// EncodeTo encodes a value into the encoder.
func (c *integerSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	var out reflect.SliceHeader
	out.Data = rv.Pointer()
	out.Len = rv.Len() * c.sizeOfInt
	out.Cap = out.Len

	e.WriteUint64(uint64(rv.Len() * c.sizeOfInt))
	e.Write(*(*[]byte)(unsafe.Pointer(&out)))
	return
}

// DecodeTo decodes into a reflect value from the decoder.
func (c *integerSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var l uint64
	var b []byte

	if l, err = d.ReadUint64(); err == nil && l > 0 {
		if b, err = d.Slice(int(l)); err == nil {
			out := (*reflect.SliceHeader)(unsafe.Pointer(rv.UnsafeAddr()))
			out.Data = (*reflect.SliceHeader)(unsafe.Pointer(&b)).Data
			out.Len = int(l) / c.sizeOfInt
			out.Cap = int(l) / c.sizeOfInt
		}
	}
	return
}

// ------------------------------------------------------------------------------

type byteSliceCodec struct{}

// Encode encodes a value into the encoder.
func (c *byteSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	e.WriteUvarint(uint64(rv.Len()))
	e.Write(rv.Bytes())
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *byteSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var l uint64
	var b []byte

	if l, err = d.ReadUvarint(); err == nil && l > 0 {
		if b, err = d.Slice(int(l)); err == nil {
			rv.SetBytes(b)
		}
	}
	return
}

// ------------------------------------------------------------------------------

type stringCodec struct{}

// Encode encodes a value into the encoder.
func (c *stringCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) error {
	v := rv.String()
	e.WriteUvarint(uint64(len(v)))
	e.Write(binary.ToBytes(v))
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *stringCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var l uint64
	var v []byte

	if l, err = d.ReadUvarint(); err == nil {
		if v, err = d.Slice(int(l)); err == nil {
			rv.SetString(binary.ToString(&v))
		}
	}
	return
}

// ------------------------------------------------------------------------------

type boolSliceCodec struct{}

// Encode encodes a value into the encoder.
func (c *boolSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	if l > 0 {
		v := rv.Interface().(Bools)
		e.Write(boolsToBinary(&v))
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *boolSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var l uint64
	var v []byte

	if l, err = d.ReadUvarint(); err == nil && l > 0 {
		if v, err = d.Slice(int(l)); err == nil {
			rv.Set(reflect.ValueOf(binaryToBools(&v)))
		}
	}
	return
}

// -----------------------------------------------------------------------------

// The codec to use for marshaling the properties
type byteMapCodec struct{}

// Encode encodes a value into the encoder.
func (c *byteMapCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	dict := rv.Interface().(ByteMap)
	e.WriteUint16(uint16(len(dict)))
	for k, v := range dict {
		encodeString(e, k)
		e.WriteUvarint(uint64(len(v)))
		e.Write(v)
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *byteMapCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var size uint16
	if size, err = d.ReadUint16(); err == nil {
		dict := make(ByteMap, int(size))
		rv.Set(reflect.ValueOf(dict))
		for i := 0; i < int(size); i++ {
			k, _ := decodeString(d)
			var l uint64
			var b []byte
			if l, err = d.ReadUvarint(); err == nil && l > 0 {
				if b, err = d.Slice(int(l)); err == nil {
					dict[k] = b
				}
			}
		}
	}
	return
}

// -----------------------------------------------------------------------------

// The codec to use for marshaling the pre-hashed hash maps
type hashMapCodec struct{}

// Encode encodes a value into the encoder.
func (c *hashMapCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	dict := rv.Interface().(HashMap)
	e.WriteUint32(uint32(len(dict)))
	for k, v := range dict {
		e.WriteUint64(k)
		e.WriteUint32(uint32(len(v)))
		e.Write(v)
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *hashMapCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var size uint32
	if size, err = d.ReadUint32(); err == nil {
		dict := make(HashMap, int(size))
		rv.Set(reflect.ValueOf(dict))
		for i := 0; i < int(size); i++ {
			k, _ := d.ReadUint64()
			var l uint32
			var b []byte
			if l, err = d.ReadUint32(); err == nil && l > 0 {
				if b, err = d.Slice(int(l)); err == nil {
					dict[k] = b
				}
			}
		}
	}
	return
}

// -----------------------------------------------------------------------------

// The codec to use for marshaling the properties
type dictionaryCodec struct{}

// Encode encodes a value into the encoder.
func (c *dictionaryCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	dict := rv.Interface().(Dictionary)
	e.WriteUint16(uint16(len(dict)))
	for k, v := range dict {
		encodeString(e, k)
		encodeString(e, v)
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *dictionaryCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var size uint16
	if size, err = d.ReadUint16(); err == nil {
		dict := make(Dictionary)
		rv.Set(reflect.ValueOf(dict))
		for i := 0; i < int(size); i++ {
			k, _ := decodeString(d)
			v, _ := decodeString(d)
			dict[k] = v
		}
	}
	return
}

// encodeString writes a string to the encoder
func encodeString(e *binary.Encoder, v string) {
	e.WriteUvarint(uint64(len(v)))
	e.Write(binary.ToBytes(v))
}

// decodeString reads a string from the decoder
func decodeString(d *binary.Decoder) (v string, err error) {
	var l uint64
	var b []byte
	if l, err = d.ReadUvarint(); err == nil {
		if b, err = d.Slice(int(l)); err == nil {
			v = binary.ToString(&b)
		}
	}
	return
}
