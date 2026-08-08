// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"bytes"
	"errors"
	"reflect"
	"sync"
	"unsafe"
)

var ErrMultipleArms = errors.New("binary: multiple union arms set")

type tagState struct {
	bytes.Buffer
	encoder Encoder
}

var tagBuffers = sync.Pool{New: func() any {
	return new(tagState)
}}

type unionArm struct {
	tag    uint64
	index  int
	offset uintptr
	elem   reflect.Type
	codec  Codec
}

type reflectUnionCodec struct {
	arms  []unionArm
	byTag []int // tag → index into arms; -1 = unknown; sized maxTag+1
}

func (c *reflectUnionCodec) lookup(tag uint64) *unionArm {
	if tag >= uint64(len(c.byTag)) || c.byTag[tag] < 0 {
		return nil
	}
	return &c.arms[c.byTag[tag]]
}
func (c *reflectUnionCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	selected, elem := c.findArm(rv)
	switch selected {
	case &errArm:
		return ErrMultipleArms
	case nil:
		e.WriteTagged(0, nil)
		return e.err
	}
	state := tagBuffers.Get().(*tagState)
	state.Buffer.Reset()
	state.encoder.Reset(&state.Buffer)
	err = selected.codec.EncodeTo(&state.encoder, elem)
	if err == nil {
		err = state.encoder.err
	}
	if err != nil {
		tagBuffers.Put(state)
		return err
	}
	e.WriteTagged(selected.tag, state.Bytes())
	tagBuffers.Put(state)
	return e.err
}
func (c *reflectUnionCodec) findArm(rv reflect.Value) (*unionArm, reflect.Value) {
	if rv.CanAddr() {
		return c.findArmUnsafe(rv)
	}
	return c.findArmReflect(rv)
}
func (c *reflectUnionCodec) findArmUnsafe(rv reflect.Value) (*unionArm, reflect.Value) {
	base := unsafe.Pointer(rv.UnsafeAddr())
	var (
		selected *unionArm
		elem     reflect.Value
	)
	for i := range c.arms {
		arm := &c.arms[i]
		p := *(*unsafe.Pointer)(unsafe.Add(base, arm.offset))
		if p == nil {
			continue
		}
		if selected != nil {
			return &errArm, reflect.Value{}
		}
		selected = arm
		elem = reflect.NewAt(arm.elem, p).Elem()
	}
	return selected, elem
}

func (c *reflectUnionCodec) findArmReflect(rv reflect.Value) (*unionArm, reflect.Value) {
	var (
		selected *unionArm
		elem     reflect.Value
	)
	for i := range c.arms {
		arm := &c.arms[i]
		f := rv.Field(arm.index)
		if f.IsNil() {
			continue
		}
		if selected != nil {
			return &errArm, reflect.Value{}
		}
		selected = arm
		elem = f.Elem()
	}
	return selected, elem
}

var errArm = unionArm{}

func (c *reflectUnionCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	tag, body, err := d.ReadTagged()
	if err != nil {
		return err
	}
	arm := c.lookup(tag)
	if arm == nil {
		c.clearValue(rv)
		return nil
	}
	var ptr reflect.Value
	if rv.CanAddr() {
		base := unsafe.Pointer(rv.UnsafeAddr())
		if p := *(*unsafe.Pointer)(unsafe.Add(base, arm.offset)); p != nil {
			ptr = reflect.NewAt(arm.elem, p)
		}
		c.clearUnsafe(base)
	} else {
		c.clearValue(rv)
	}
	if !ptr.IsValid() {
		ptr = reflect.New(arm.elem)
	}
	if err = c.decodeArm(d, arm.codec, body, ptr.Elem()); err != nil {
		return err
	}
	if rv.CanAddr() {
		*(*unsafe.Pointer)(unsafe.Add(unsafe.Pointer(rv.UnsafeAddr()), arm.offset)) = ptr.UnsafePointer()
	} else {
		rv.Field(arm.index).Set(ptr)
	}
	return nil
}

func (c *reflectUnionCodec) clearUnsafe(base unsafe.Pointer) {
	for i := range c.arms {
		*(*unsafe.Pointer)(unsafe.Add(base, c.arms[i].offset)) = nil
	}
}

func (c *reflectUnionCodec) clearValue(rv reflect.Value) {
	for i := range c.arms {
		rv.Field(c.arms[i].index).Set(reflect.Zero(rv.Field(c.arms[i].index).Type()))
	}
}

func (c *reflectUnionCodec) decodeArm(d *Decoder, codec Codec, body []byte, elem reflect.Value) error {
	if d.slice != nil {
		r := d.slice
		buffer, offset := r.buffer, r.offset
		r.buffer, r.offset = body, 0
		err := codec.DecodeTo(d, elem)
		r.buffer, r.offset = buffer, offset
		return err
	}
	dec := decoders.Get().(*Decoder)
	dec.reader.(*sliceReader).Reset(body)
	dec.last = nil
	dec.codec = nil
	dec.arena = nil
	err := codec.DecodeTo(dec, elem)
	dec.arena = nil
	decoders.Put(dec)
	return err
}
