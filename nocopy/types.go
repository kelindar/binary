// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package nocopy

import (
	"bytes"
	bin "encoding/binary"
	"encoding/json"
	"github.com/kelindar/binary"
	"reflect"
	"unsafe"
)

type String string

func (s *String) GetBinaryCodec() binary.Codec { return new(stringCodec) }

type Bytes []byte

func (s *Bytes) GetBinaryCodec() binary.Codec { return new(byteSliceCodec) }

type JSON json.RawMessage

func (j JSON) MarshalJSON() ([]byte, error) {
	return json.RawMessage(j).MarshalJSON()
}
func (j *JSON) UnmarshalJSON(data []byte) error {
	return (*json.RawMessage)(j).UnmarshalJSON(data)
}
func (j *JSON) GetBinaryCodec() binary.Codec { return new(byteSliceCodec) }

type Bools []bool

func (s *Bools) GetBinaryCodec() binary.Codec { return new(boolSliceCodec) }

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

type Dictionary map[string]string

func (d *Dictionary) GetBinaryCodec() binary.Codec { return new(dictionaryCodec) }

type ByteMap map[string][]byte

func (d *ByteMap) GetBinaryCodec() binary.Codec { return new(byteMapCodec) }

type HashMap map[uint64][]byte

func (d *HashMap) GetBinaryCodec() binary.Codec { return new(hashMapCodec) }

type integerSliceCodec struct {
	sizeOfInt int
}

func integerCodec[T any](size int) binary.Codec {
	return &integerSliceCodec{sizeOfInt: size}
}
func (c *integerSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	n := rv.Len() * c.sizeOfInt
	e.WriteUint64(uint64(n))
	e.Write(unsafe.Slice((*byte)(rv.UnsafePointer()), n))
	return
}
func (c *integerSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var l uint64
	var b []byte
	if l, err = d.ReadUint64(); err == nil && l > 0 {
		if b, err = d.Slice(int(l)); err == nil {
			setSlice(rv, unsafe.Pointer(unsafe.SliceData(b)), int(l)/c.sizeOfInt)
		}
	}
	return
}

type sliceHeader struct {
	Data     unsafe.Pointer
	Len, Cap int
}

func setSlice(rv reflect.Value, data unsafe.Pointer, n int) {
	*(*sliceHeader)(unsafe.Pointer(rv.UnsafeAddr())) = sliceHeader{data, n, n}
}

type byteSliceCodec struct{}

func (c *byteSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	e.WriteUvarint(uint64(rv.Len()))
	e.Write(rv.Bytes())
	return
}
func (c *byteSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var b []byte
	if b, err = d.ReadSlice(); err == nil && len(b) > 0 {
		setSlice(rv, unsafe.Pointer(unsafe.SliceData(b)), len(b))
	}
	return
}

type stringCodec struct{}

func (c *stringCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) error {
	v := rv.String()
	e.WriteUvarint(uint64(len(v)))
	e.Write(binary.ToBytes(v))
	return nil
}
func (c *stringCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var v []byte
	if v, err = d.ReadSlice(); err == nil {
		*(*string)(unsafe.Pointer(rv.UnsafeAddr())) = binary.ToString(&v)
	}
	return
}

type boolSliceCodec struct{}

func (c *boolSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	if l > 0 {
		v := rv.Interface().(Bools)
		e.Write(boolsToBinary(&v))
	}
	return
}
func (c *boolSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var l uint64
	var v []byte
	if l, err = d.ReadUvarint(); err == nil && l > 0 {
		if v, err = d.Slice(int(l)); err == nil {
			b := binaryToBools(&v)
			setSlice(rv, unsafe.Pointer(unsafe.SliceData(b)), len(b))
		}
	}
	return
}

type byteMapCodec struct{}

func (c *byteMapCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	dict := rv.Interface().(ByteMap)
	if len(dict) >= 8 {
		if out, ok := e.Buffer().(*bytes.Buffer); ok {
			size := 2
			for key, value := range dict {
				size += uvarintSize(uint64(len(key))) + len(key) + uvarintSize(uint64(len(value))) + len(value)
			}
			out.Grow(size)
			buffer := out.AvailableBuffer()
			buffer = append(buffer, byte(len(dict)), byte(len(dict)>>8))
			for key, value := range dict {
				buffer = bin.AppendUvarint(buffer, uint64(len(key)))
				buffer = append(buffer, key...)
				buffer = bin.AppendUvarint(buffer, uint64(len(value)))
				buffer = append(buffer, value...)
			}
			e.Write(buffer)
			return
		}
	}
	e.WriteUint16(uint16(len(dict)))
	for k, v := range dict {
		e.WriteString(k)
		e.WriteUvarint(uint64(len(v)))
		e.Write(v)
	}
	return
}
func (c *byteMapCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var size uint16
	if size, err = d.ReadUint16(); err == nil {
		dict := rv.Interface().(ByteMap)
		if dict == nil {
			dict = make(ByteMap, int(size))
			rv.Set(reflect.ValueOf(dict))
		} else {
			clear(dict)
		}
		for i := 0; i < int(size); i++ {
			k, err := decodeString(d)
			if err != nil {
				return err
			}
			var l uint64
			if l, err = d.ReadUvarint(); err != nil {
				return err
			}
			var b []byte
			if l > 0 {
				if b, err = d.Slice(int(l)); err != nil {
					return err
				}
			}
			dict[k] = b
		}
	}
	return
}

type hashMapCodec struct{}

func (c *hashMapCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	dict := rv.Interface().(HashMap)
	if len(dict) >= 8 {
		if out, ok := e.Buffer().(*bytes.Buffer); ok {
			size := 4
			for _, value := range dict {
				size += 12 + len(value)
			}
			out.Grow(size)
			buffer := out.AvailableBuffer()
			buffer = bin.LittleEndian.AppendUint32(buffer, uint32(len(dict)))
			for key, value := range dict {
				buffer = bin.LittleEndian.AppendUint64(buffer, key)
				buffer = bin.LittleEndian.AppendUint32(buffer, uint32(len(value)))
				buffer = append(buffer, value...)
			}
			e.Write(buffer)
			return
		}
	}
	e.WriteUint32(uint32(len(dict)))
	for k, v := range dict {
		e.WriteUint64(k)
		e.WriteUint32(uint32(len(v)))
		e.Write(v)
	}
	return
}
func (c *hashMapCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var size uint32
	if size, err = d.ReadUint32(); err == nil {
		dict := rv.Interface().(HashMap)
		if dict == nil {
			dict = make(HashMap, int(size))
			rv.Set(reflect.ValueOf(dict))
		} else {
			clear(dict)
		}
		for i := 0; i < int(size); i++ {
			k, err := d.ReadUint64()
			if err != nil {
				return err
			}
			var l uint32
			var b []byte
			if l, err = d.ReadUint32(); err != nil {
				return err
			}
			if l > 0 {
				if b, err = d.Slice(int(l)); err != nil {
					return err
				}
			}
			dict[k] = b
		}
	}
	return
}

type dictionaryCodec struct{}

func (c *dictionaryCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	dict := rv.Interface().(Dictionary)
	e.WriteUint16(uint16(len(dict)))
	for k, v := range dict {
		e.WriteString(k)
		e.WriteString(v)
	}
	return
}
func (c *dictionaryCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var size uint16
	if size, err = d.ReadUint16(); err == nil {
		dict := rv.Interface().(Dictionary)
		if dict == nil {
			dict = make(Dictionary, int(size))
			rv.Set(reflect.ValueOf(dict))
		} else {
			clear(dict)
		}
		for i := 0; i < int(size); i++ {
			k, err := decodeString(d)
			if err != nil {
				return err
			}
			v, err := decodeString(d)
			if err != nil {
				return err
			}
			dict[k] = v
		}
	}
	return
}
func uvarintSize(x uint64) int {
	size := 1
	for x >= 0x80 {
		x >>= 7
		size++
	}
	return size
}
func decodeString(d *binary.Decoder) (v string, err error) {
	var b []byte
	if b, err = d.ReadSlice(); err == nil {
		v = binary.ToString(&b)
	}
	return
}
