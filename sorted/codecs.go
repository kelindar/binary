// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	"bytes"
	bin "encoding/binary"
	"reflect"
	"sort"

	"github.com/kelindar/binary"
)

type intSliceCodec struct {
	sliceType reflect.Type
	sizeOfInt int
}

// EncodeTo encodes a value into the encoder.
func (c *intSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	sort.Sort(rv.Interface().(sort.Interface))

	prev := int64(0)
	temp := make([]byte, 10)
	bytes := make([]byte, 0, rv.Len()+2)

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

	if l, err = d.ReadUvarint(); err == nil && l > 0 {
		if b, err = d.Slice(int(l)); err == nil {

			// Create a new slice and figure out its element type
			elemType := c.sliceType.Elem()
			slice := reflect.MakeSlice(c.sliceType, 0, 16)

			// Using old implementation, can be optimised
			read := bytes.NewReader(b)
			prev := int64(0)
			for read.Len() > 0 {
				diff, _ := bin.ReadVarint(read)
				prev = prev + diff
				slice = reflect.Append(slice, reflect.ValueOf(prev).Convert(elemType))
			}

			rv.Set(slice)
		}
	}
	return
}

// ------------------------------------------------------------------------------

type uintSliceCodec struct {
	sliceType reflect.Type
	sizeOfInt int
}

// EncodeTo encodes a value into the encoder.
func (c *uintSliceCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	sort.Sort(rv.Interface().(sort.Interface))

	prev := uint64(0)
	temp := make([]byte, 10)
	bytes := make([]byte, 0, rv.Len()+2)

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

	if l, err = d.ReadUvarint(); err == nil && l > 0 {
		if b, err = d.Slice(int(l)); err == nil {

			// Create a new slice and figure out its element type
			elemType := c.sliceType.Elem()
			slice := reflect.MakeSlice(c.sliceType, 0, 16)

			// Using old implementation, can be optimised
			read := bytes.NewReader(b)
			prev := uint64(0)
			for read.Len() > 0 {
				diff, _ := bin.ReadUvarint(read)
				prev = prev + diff
				slice = reflect.Append(slice, reflect.ValueOf(prev).Convert(elemType))
			}

			rv.Set(slice)
		}
	}
	return
}
