// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package unsafe

import (
	"reflect"
	"unsafe"

	"github.com/kelindar/binary"
)

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

	e.WriteUint64(uint64(rv.Len()))
	e.Write(*(*[]byte)(unsafe.Pointer(&out)))
	return
}

// DecodeTo decodes into a reflect value from the decoder.
func (c *integerSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUint64(); err == nil {
		src := reflect.MakeSlice(c.sliceType, int(l), int(l))

		var out reflect.SliceHeader
		out.Data = src.Pointer()
		out.Len = int(l) * c.sizeOfInt
		out.Cap = int(l) * c.sizeOfInt
		data := *(*[]byte)(unsafe.Pointer(&out))
		if _, err = d.Read(data); err == nil {
			rv.Set(src)
		}
	}
	return
}

// ------------------------------------------------------------------------------

// Bools represents a slice serialized in an unsafe, non portable manner.
type Bools []bool

// GetBinaryCodec retrieves a custom binary codec.
func (s *Bools) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Bools{}),
		sizeOfInt: 1,
	}
}

// ------------------------------------------------------------------------------

// Uint16s represents a slice serialized in an unsafe, non portable manner.
type Uint16s []uint16

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint16s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Uint16s{}),
		sizeOfInt: 2,
	}
}

// ------------------------------------------------------------------------------

// Int16s represents a slice serialized in an unsafe, non portable manner.
type Int16s []int16

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int16s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Int16s{}),
		sizeOfInt: 2,
	}
}

// ------------------------------------------------------------------------------

// Uint32s represents a slice serialized in an unsafe, non portable manner.
type Uint32s []uint32

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint32s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Uint32s{}),
		sizeOfInt: 4,
	}
}

// ------------------------------------------------------------------------------

// Int32s represents a slice serialized in an unsafe, non portable manner.
type Int32s []int32

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int32s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Int32s{}),
		sizeOfInt: 4,
	}
}

// ------------------------------------------------------------------------------

// Uint64s represents a slice serialized in an unsafe, non portable manner.
type Uint64s []uint64

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint64s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Uint64s{}),
		sizeOfInt: 8,
	}
}

// ------------------------------------------------------------------------------

// Int64s represents a slice serialized in an unsafe, non portable manner.
type Int64s []int64

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int64s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Int64s{}),
		sizeOfInt: 8,
	}
}

// ------------------------------------------------------------------------------

// Float32s represents a slice serialized in an unsafe, non portable manner.
type Float32s []float32

// GetBinaryCodec retrieves a custom binary codec.
func (s *Float32s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Float32s{}),
		sizeOfInt: 4,
	}
}

// ------------------------------------------------------------------------------

// Float64s represents a slice serialized in an unsafe, non portable manner.
type Float64s []float64

// GetBinaryCodec retrieves a custom binary codec.
func (s *Float64s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Float64s{}),
		sizeOfInt: 8,
	}
}
