// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package unsafe

import (
	"io"
	"reflect"
	"unsafe"

	"github.com/kelindar/binary"
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

func decodeLength(n uint64) (int, error) {
	if n > uint64(^uint(0)>>1) {
		return 0, io.ErrUnexpectedEOF
	}
	return int(n), nil
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
	if l, err = d.ReadUint64(); err != nil {
		return
	}
	if l == 0 {
		rv.SetZero()
		return nil
	}
	n, err := decodeLength(l)
	if err != nil {
		return err
	}
	if n > int(^uint(0)>>1)/c.sizeOfInt {
		return io.ErrUnexpectedEOF
	}
	size := n * c.sizeOfInt
	src := reflect.MakeSlice(c.sliceType, n, n)
	if _, err = io.ReadFull(d, unsafe.Slice((*byte)(src.UnsafePointer()), size)); err != nil {
		return err
	}
	rv.Set(src)
	return
}
