// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package unsafe

import (
	"reflect"
	"unsafe"

	"github.com/kelindar/binary"
)

// ------------------------------------------------------------------------------

type byteSliceCodec struct{}

// EncodeTo encodes a value into the encoder.
func (c *byteSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	e.WriteUint64(uint64(rv.Len()))
	e.Write(*(*[]byte)(unsafe.Pointer(rv.Addr().Pointer())))
	return
}

// DecodeTo decodes into a reflect value from the decoder.
func (c *byteSliceCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUint64(); err == nil {
		data := make([]byte, int(l), int(l))
		if _, err = d.Read(data); err == nil {
			rv.Set(reflect.ValueOf(data))
		}
	}
	return
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
