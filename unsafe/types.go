// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package unsafe

import (
	"github.com/kelindar/binary"
	"reflect"
	"unsafe"
)

type Bools []bool

func (s *Bools) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{sliceType: reflect.TypeFor[Bools](), sizeOfInt: 1}
}

type Uint16s []uint16

func (s Uint16s) Len() int                      { return len(s) }
func (s Uint16s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Uint16s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Uint16s) GetBinaryCodec() binary.Codec { return integerCodec[Uint16s](2) }

type Int16s []int16

func (s Int16s) Len() int                      { return len(s) }
func (s Int16s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Int16s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Int16s) GetBinaryCodec() binary.Codec { return integerCodec[Int16s](2) }

type Uint32s []uint32

func (s Uint32s) Len() int                      { return len(s) }
func (s Uint32s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Uint32s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Uint32s) GetBinaryCodec() binary.Codec { return integerCodec[Uint32s](4) }

type Int32s []int32

func (s Int32s) Len() int                      { return len(s) }
func (s Int32s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Int32s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Int32s) GetBinaryCodec() binary.Codec { return integerCodec[Int32s](4) }

type Uint64s []uint64

func (s Uint64s) Len() int                      { return len(s) }
func (s Uint64s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Uint64s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Uint64s) GetBinaryCodec() binary.Codec { return integerCodec[Uint64s](8) }

type Int64s []int64

func (s Int64s) Len() int                      { return len(s) }
func (s Int64s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Int64s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Int64s) GetBinaryCodec() binary.Codec { return integerCodec[Int64s](8) }

type Float32s []float32

func (s Float32s) Len() int                      { return len(s) }
func (s Float32s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Float32s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Float32s) GetBinaryCodec() binary.Codec { return integerCodec[Float32s](4) }

type Float64s []float64

func (s Float64s) Len() int                      { return len(s) }
func (s Float64s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Float64s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Float64s) GetBinaryCodec() binary.Codec { return integerCodec[Float64s](8) }

type integerSliceCodec struct {
	sliceType reflect.Type
	sizeOfInt int
}

func integerCodec[T any](size int) binary.Codec {
	return &integerSliceCodec{sliceType: reflect.TypeFor[T](), sizeOfInt: size}
}
func (c *integerSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	n := rv.Len() * c.sizeOfInt
	e.WriteUint64(uint64(rv.Len()))
	e.Write(unsafe.Slice((*byte)(rv.UnsafePointer()), n))
	return
}
func (c *integerSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUint64(); err == nil && l > 0 {
		src := reflect.MakeSlice(c.sliceType, int(l), int(l))
		data := unsafe.Slice((*byte)(src.UnsafePointer()), int(l)*c.sizeOfInt)
		if _, err = d.Read(data); err == nil {
			rv.Set(src)
		}
	}
	return
}
