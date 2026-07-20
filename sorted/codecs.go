// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	bin "encoding/binary"
	"errors"
	"reflect"
	"sort"

	"github.com/kelindar/binary"
)

var errInvalidVarint = errors.New("sorted: invalid varint")

// IntsCodecAs returns an int slice codec with the specified precision and type.
func IntsCodecAs(sliceType reflect.Type, sizeOfInt int) binary.Codec {
	return &intSliceCodec{
		sliceType: sliceType,
		sizeOfInt: sizeOfInt,
	}
}

type intSliceCodec struct {
	sliceType reflect.Type
	sizeOfInt int
}

// EncodeTo encodes a value into the encoder.
func (c *intSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	sort.Sort(rv.Interface().(sort.Interface))

	prev := int64(0)
	temp := make([]byte, 10)
	bytes := make([]byte, 0, c.sizeOfInt*rv.Len())

	for i := 0; i < rv.Len(); i++ {
		curr := rv.Index(i).Int()
		diff := curr - prev
		bytes = append(bytes, temp[:bin.PutVarint(temp, diff)]...)
		prev = curr
	}

	e.WriteUvarint(uint64(len(bytes)))
	e.Write(bytes)
	return
}

// DecodeTo decodes into a reflect value from the decoder.
func (c *intSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
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

			// Iterate through and uncompress
			prev := int64(0)
			for i, j := 0, 0; i < len(b); j++ {
				diff, n := bin.Varint(b[i:])
				if n <= 0 {
					return errInvalidVarint
				}
				prev = prev + diff
				rv.Index(j).SetInt(prev)
				i += n
			}
		}
	}
	return
}

// ------------------------------------------------------------------------------

// UintsCodecAs returns an uint slice codec with the specified precision and type.
func UintsCodecAs(sliceType reflect.Type, sizeOfInt int) binary.Codec {
	return &uintSliceCodec{
		sliceType: sliceType,
		sizeOfInt: sizeOfInt,
	}
}

type uintSliceCodec struct {
	sliceType reflect.Type
	sizeOfInt int
}

// EncodeTo encodes a value into the encoder.
func (c *uintSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	sort.Sort(rv.Interface().(sort.Interface))

	prev := uint64(0)
	temp := make([]byte, 10)
	bytes := make([]byte, 0, c.sizeOfInt*rv.Len())

	for i := 0; i < rv.Len(); i++ {
		curr := rv.Index(i).Uint()
		diff := curr - prev
		bytes = append(bytes, temp[:bin.PutUvarint(temp, diff)]...)
		prev = curr
	}

	e.WriteUvarint(uint64(len(bytes)))
	e.Write(bytes)
	return
}

// DecodeTo decodes into a reflect value from the decoder.
func (c *uintSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
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

			// Iterate through and uncompress
			prev := uint64(0)
			for i, j := 0, 0; i < len(b); j++ {
				diff, n := bin.Uvarint(b[i:])
				if n <= 0 {
					return errInvalidVarint
				}
				prev = prev + diff
				rv.Index(j).SetUint(prev)
				i += n
			}
		}
	}
	return
}

func countVarints(b []byte) (count int) {
	for _, v := range b {
		if v < 0x80 {
			count++
		}
	}
	return
}

// ------------------------------------------------------------------------------

type timestampCodec struct{}

// EncodeTo encodes a value into the encoder.
func (c timestampCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	data := rv.Interface().(Timestamps)
	if !sort.IsSorted(Uint64s(data)) {
		sort.Sort(Uint64s(data))
	}

	temp := make([]byte, 10)
	buffer := make([]byte, 0, 2*len(data)) // ~1-2 bytes per timestamp
	prev := uint64(0)
	for _, curr := range data {
		diff := curr - prev
		prev = curr
		buffer = append(buffer, temp[:bin.PutUvarint(temp, uint64(diff))]...)
	}

	// Writhe the size and the buffer
	e.WriteUvarint(uint64(len(data)))
	e.WriteUvarint(uint64(len(buffer)))
	e.Write(buffer)
	return
}

// DecodeTo decodes into a reflect value from the decoder.
func (timestampCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) error {

	// Read the number of timestamps
	count, err := d.ReadUvarint()
	if err != nil {
		return err
	}

	// Read the size in bytes
	size, err := d.ReadUvarint()
	if err != nil {
		return err
	}

	// Read the timestamp buffer
	buffer, err := d.Slice(int(size))
	if err != nil {
		return err
	}

	// Read the timestamps
	slice := make(Timestamps, 0, count)
	prev := uint64(0)
	for i := 0; i < int(size); {
		diff, n := bin.Uvarint(buffer[i:])
		prev = prev + diff
		slice = append(slice, uint64(prev))
		i += n
	}

	rv.Set(reflect.ValueOf(slice))
	return nil
}
