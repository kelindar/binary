// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	bin "encoding/binary"
	"errors"
	"reflect"
	"sort"
	"unsafe"

	"github.com/kelindar/binary"
)

var errInvalidVarint = errors.New("sorted: invalid varint")

func IntsCodecAs(sliceType reflect.Type, sizeOfInt int) binary.Codec {
	return &deltaSliceCodec{sliceType: sliceType, sizeOfInt: sizeOfInt}
}

type deltaSliceCodec struct {
	sliceType reflect.Type
	sizeOfInt int
	unsigned  bool
}
type signedInteger interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}
type unsignedInteger interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func appendIntDeltas[T signedInteger](dst []byte, data []T) []byte {
	prev := int64(0)
	for _, curr := range data {
		value := int64(curr)
		dst = bin.AppendVarint(dst, value-prev)
		prev = value
	}
	return dst
}
func appendUintDeltas[T unsignedInteger](dst []byte, data []T) []byte {
	prev := uint64(0)
	for _, curr := range data {
		value := uint64(curr)
		dst = bin.AppendUvarint(dst, value-prev)
		prev = value
	}
	return dst
}
func decodeIntDeltas[T signedInteger](dst []T, data []byte) error {
	prev := int64(0)
	for i, j := 0, 0; i < len(data); j++ {
		diff, n := bin.Varint(data[i:])
		if n <= 0 {
			return errInvalidVarint
		}
		prev += diff
		dst[j] = T(prev)
		i += n
	}
	return nil
}
func decodeUintDeltas[T unsignedInteger](dst []T, data []byte) error {
	prev := uint64(0)
	for i, j := 0, 0; i < len(data); j++ {
		diff, n := bin.Uvarint(data[i:])
		if n <= 0 {
			return errInvalidVarint
		}
		prev += diff
		dst[j] = T(prev)
		i += n
	}
	return nil
}

func (c *deltaSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	data := rv.Interface().(sort.Interface)
	if !sort.IsSorted(data) {
		sort.Sort(data)
	}
	bytes := make([]byte, 0, c.sizeOfInt*rv.Len())
	base := rv.UnsafePointer()
	if c.unsigned {
		switch c.sizeOfInt {
		case 1:
			bytes = appendUintDeltas(bytes, unsafe.Slice((*uint8)(base), rv.Len()))
		case 2:
			bytes = appendUintDeltas(bytes, unsafe.Slice((*uint16)(base), rv.Len()))
		case 4:
			bytes = appendUintDeltas(bytes, unsafe.Slice((*uint32)(base), rv.Len()))
		case 8:
			bytes = appendUintDeltas(bytes, unsafe.Slice((*uint64)(base), rv.Len()))
		}
	} else {
		switch c.sizeOfInt {
		case 1:
			bytes = appendIntDeltas(bytes, unsafe.Slice((*int8)(base), rv.Len()))
		case 2:
			bytes = appendIntDeltas(bytes, unsafe.Slice((*int16)(base), rv.Len()))
		case 4:
			bytes = appendIntDeltas(bytes, unsafe.Slice((*int32)(base), rv.Len()))
		case 8:
			bytes = appendIntDeltas(bytes, unsafe.Slice((*int64)(base), rv.Len()))
		}
	}
	e.WriteUvarint(uint64(len(bytes)))
	e.Write(bytes)
	return
}
func (c *deltaSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var l uint64
	var b []byte
	if l, err = d.ReadUvarint(); err == nil {
		if l == 0 {
			rv.SetLen(0)
			return nil
		}
		if b, err = d.Slice(int(l)); err == nil {
			count := countVarints(b)
			if rv.Cap() < count {
				rv.Set(reflect.MakeSlice(c.sliceType, count, count))
			} else {
				rv.SetLen(count)
			}
			base := rv.UnsafePointer()
			if c.unsigned {
				switch c.sizeOfInt {
				case 1:
					err = decodeUintDeltas(unsafe.Slice((*uint8)(base), count), b)
				case 2:
					err = decodeUintDeltas(unsafe.Slice((*uint16)(base), count), b)
				case 4:
					err = decodeUintDeltas(unsafe.Slice((*uint32)(base), count), b)
				case 8:
					err = decodeUintDeltas(unsafe.Slice((*uint64)(base), count), b)
				}
			} else {
				switch c.sizeOfInt {
				case 1:
					err = decodeIntDeltas(unsafe.Slice((*int8)(base), count), b)
				case 2:
					err = decodeIntDeltas(unsafe.Slice((*int16)(base), count), b)
				case 4:
					err = decodeIntDeltas(unsafe.Slice((*int32)(base), count), b)
				case 8:
					err = decodeIntDeltas(unsafe.Slice((*int64)(base), count), b)
				}
			}
		}
	}
	return
}

func UintsCodecAs(sliceType reflect.Type, sizeOfInt int) binary.Codec {
	return &deltaSliceCodec{sliceType: sliceType, sizeOfInt: sizeOfInt, unsigned: true}
}
func countVarints(b []byte) (count int) {
	for _, v := range b {
		if v < 0x80 {
			count++
		}
	}
	return
}

func isSorted(data []uint64) bool {
	for i := 1; i < len(data); i++ {
		if data[i] < data[i-1] {
			return false
		}
	}
	return true
}

type timestampCodec struct{}

func (c timestampCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	data := rv.Interface().(Timestamps)
	if !isSorted(data) {
		sort.Sort(Uint64s(data))
	}
	buffer := appendDelta(make([]byte, 0, 2*len(data)), []uint64(data)) // ~1-2 bytes per timestamp
	e.WriteUvarint(uint64(len(data)))
	e.WriteUvarint(uint64(len(buffer)))
	e.Write(buffer)
	return
}
func (timestampCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) error {
	count, err := d.ReadUvarint()
	if err != nil {
		return err
	}
	size, err := d.ReadUvarint()
	if err != nil {
		return err
	}
	buffer, err := d.Slice(int(size))
	if err != nil {
		return err
	}
	n := int(count)
	slice := rv.Interface().(Timestamps)
	if slice == nil || cap(slice) < n {
		slice = make(Timestamps, n)
	} else {
		slice = slice[:n]
	}
	if _, err = readDelta([]uint64(slice), buffer); err != nil {
		return err
	}
	rv.Set(reflect.ValueOf(slice))
	return nil
}
