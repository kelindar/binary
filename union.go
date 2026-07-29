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

// ErrMultipleArms is returned when encoding a union with more than one non-nil arm.
var ErrMultipleArms = errors.New("binary: multiple union arms set")

// Pooled scratch buffers for framing union arm payloads.
var tagBuffers = sync.Pool{New: func() any {
	return new(bytes.Buffer)
}}

// unionArm is a scan-time arm descriptor for a tagged union struct.
type unionArm struct {
	tag    uint64
	index  int
	offset uintptr
	elem   reflect.Type
	codec  Codec
}

// reflectUnionCodec encodes a pointer-arm struct as uvarint(tag)+uvarint(len)+body.
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
	var (
		selected *unionArm
		elem     reflect.Value
	)

	if rv.CanAddr() {
		base := unsafe.Pointer(rv.UnsafeAddr())
		for i := range c.arms {
			arm := &c.arms[i]
			p := *(*unsafe.Pointer)(unsafe.Add(base, arm.offset))
			if p == nil {
				continue
			}
			if selected != nil {
				return ErrMultipleArms
			}
			selected = arm
			elem = reflect.NewAt(arm.elem, p).Elem()
		}
	} else {
		for i := range c.arms {
			arm := &c.arms[i]
			f := rv.Field(arm.index)
			if f.IsNil() {
				continue
			}
			if selected != nil {
				return ErrMultipleArms
			}
			selected = arm
			elem = f.Elem()
		}
	}

	if selected == nil {
		e.WriteTagged(0, nil)
		return e.err
	}

	buf := tagBuffers.Get().(*bytes.Buffer)
	buf.Reset()
	tmp := encoders.Get().(*Encoder)
	tmp.Reset(buf)
	err = selected.codec.EncodeTo(tmp, elem)
	if err == nil {
		err = tmp.err
	}
	if err != nil {
		encoders.Put(tmp)
		tagBuffers.Put(buf)
		return err
	}
	e.WriteTagged(selected.tag, buf.Bytes())
	encoders.Put(tmp)
	tagBuffers.Put(buf)
	return e.err
}

func (c *reflectUnionCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	tag, body, err := d.ReadTagged()
	if err != nil {
		return err
	}

	arm := c.lookup(tag)
	// tag 0 (none) and unknown versions both leave all arms nil
	if arm == nil {
		c.clear(rv)
		return nil
	}

	var ptr reflect.Value
	if rv.CanAddr() {
		base := unsafe.Pointer(rv.UnsafeAddr())
		p := *(*unsafe.Pointer)(unsafe.Add(base, arm.offset))
		c.clear(rv)
		if p != nil {
			ptr = reflect.NewAt(arm.elem, p)
			*(*unsafe.Pointer)(unsafe.Add(base, arm.offset)) = p
		}
	} else {
		field := rv.Field(arm.index)
		if !field.IsNil() {
			ptr = field
		}
		c.clear(rv)
		if ptr.IsValid() {
			field.Set(ptr)
		}
	}

	if !ptr.IsValid() {
		ptr = reflect.New(arm.elem)
	}
	if err = c.decodeArm(arm.codec, body, ptr.Elem()); err != nil {
		c.clear(rv)
		return err
	}

	if rv.CanAddr() {
		base := unsafe.Pointer(rv.UnsafeAddr())
		*(*unsafe.Pointer)(unsafe.Add(base, arm.offset)) = ptr.UnsafePointer()
		return nil
	}
	rv.Field(arm.index).Set(ptr)
	return nil
}

func (c *reflectUnionCodec) clear(rv reflect.Value) {
	if rv.CanAddr() {
		base := unsafe.Pointer(rv.UnsafeAddr())
		for i := range c.arms {
			*(*unsafe.Pointer)(unsafe.Add(base, c.arms[i].offset)) = nil
		}
		return
	}
	for i := range c.arms {
		rv.Field(c.arms[i].index).Set(reflect.Zero(rv.Field(c.arms[i].index).Type()))
	}
}

func (c *reflectUnionCodec) decodeArm(codec Codec, body []byte, elem reflect.Value) error {
	dec := decoders.Get().(*Decoder)
	dec.reader.(*sliceReader).Reset(body)
	dec.last = nil
	dec.codec = nil
	err := codec.DecodeTo(dec, elem)
	decoders.Put(dec)
	return err
}
