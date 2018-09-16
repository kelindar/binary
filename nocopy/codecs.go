// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package nocopy

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
			src := reflect.New(c.sliceType)
			out := (*reflect.SliceHeader)(unsafe.Pointer(src.Pointer()))

			out.Data = reflect.ValueOf(b).Pointer()
			out.Len = int(l) / c.sizeOfInt
			out.Cap = int(l) / c.sizeOfInt
			rv.Set(reflect.Indirect(src))
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
			rv.Set(reflect.ValueOf(b))
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
	e.Write(stringToBinary(v))
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *stringCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) (err error) {
	var l uint64
	var v []byte

	if l, err = d.ReadUvarint(); err == nil {
		if v, err = d.Slice(int(l)); err == nil {
			rv.SetString(binaryToString(&v))
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
