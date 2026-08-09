// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"reflect"
	"sync"
)

var errNilWriter = errors.New("binary: nil writer")

var encoders = &sync.Pool{New: func() any {
	return new(Encoder)
}}

type marshalState struct {
	bytes.Buffer
	encoder Encoder
}

var marshalBuffers = &sync.Pool{New: func() any {
	return new(marshalState)
}}

func marshalCapacity(rv reflect.Value) int {
	const limit = 1 << 20
	size := rv.Type().Size()
	if rv.Kind() == reflect.Array {
		if size <= limit {
			return max(64, int(size))
		}
		return 64
	}
	if rv.Kind() == reflect.Slice {
		size = rv.Type().Elem().Size()
	}
	if size == 0 || size > limit || uintptr(rv.Len()) > limit/size {
		return 64
	}
	return max(64, rv.Len()*int(size))
}

func Marshal(v any) (output []byte, err error) {
	state := marshalBuffers.Get().(*marshalState)
	state.Reset()
	state.Grow(64)
	state.encoder.Reset(&state.Buffer)
	err = state.encoder.Encode(v)
	if err == nil {
		output = state.Bytes()
	}
	state.Buffer = bytes.Buffer{}
	marshalBuffers.Put(state)
	return
}

func MarshalTo(v any, dst io.Writer) (err error) {
	e := encoders.Get().(*Encoder)
	e.Reset(dst)
	err = e.Encode(v)
	encoders.Put(e)
	return
}

type Encoder struct {
	scratch [10]byte
	last    reflect.Type
	codec   Codec
	out     io.Writer
	err     error
}

func NewEncoder(out io.Writer) *Encoder {
	e := new(Encoder)
	e.Reset(out)
	return e
}

func (e *Encoder) Reset(out io.Writer) {
	e.out = out
	e.err = nil
	if out == nil {
		e.err = errNilWriter
		return
	}
	if buffer, ok := out.(*bytes.Buffer); ok {
		if buffer == nil {
			e.err = errNilWriter
		}
		return
	}
	if isNilInterface(out) {
		e.err = errNilWriter
	}
}

func (e *Encoder) Buffer() io.Writer {
	return e.out
}

func (e *Encoder) Encode(v any) (err error) {
	rv := reflect.Indirect(reflect.ValueOf(v))
	if !rv.IsValid() {
		return errors.New("binary: cannot encode nil value")
	}
	t := rv.Type()
	c := e.codec
	if t != e.last {
		if c, err = scan(t); err != nil {
			return
		}
		e.last = t
		e.codec = c
	}
	if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil && (rv.Kind() == reflect.Array || rv.Kind() == reflect.Slice) {
		out.Grow(marshalCapacity(rv))
	}
	if err = c.EncodeTo(e, rv); err == nil {
		err = e.err
	}
	return
}

func (e *Encoder) Write(p []byte) {
	if e.err != nil {
		return
	}
	_, e.err = e.out.Write(p)
}

func (e *Encoder) WriteVarint(v int64) {
	x := uint64(v) << 1
	if v < 0 {
		x = ^x
	}
	i := 0
	for x >= 0x80 {
		e.scratch[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	e.scratch[i] = byte(x)
	e.Write(e.scratch[:(i + 1)])
}

func (e *Encoder) WriteUvarint(x uint64) {
	if x < 0x80 {
		e.scratch[0] = byte(x)
		e.Write(e.scratch[:1])
		return
	}
	i := 0
	for x >= 0x80 {
		e.scratch[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	e.scratch[i] = byte(x)
	e.Write(e.scratch[:(i + 1)])
}

func (e *Encoder) WriteUint16(v uint16) {
	binary.LittleEndian.PutUint16(e.scratch[:2], v)
	e.Write(e.scratch[:2])
}

func (e *Encoder) WriteUint32(v uint32) {
	binary.LittleEndian.PutUint32(e.scratch[:4], v)
	e.Write(e.scratch[:4])
}

func (e *Encoder) WriteUint64(v uint64) {
	binary.LittleEndian.PutUint64(e.scratch[:8], v)
	e.Write(e.scratch[:8])
}

func (e *Encoder) WriteFloat32(v float32) {
	e.WriteUint32(math.Float32bits(v))
}

func (e *Encoder) WriteFloat64(v float64) {
	e.WriteUint64(math.Float64bits(v))
}

func (e *Encoder) writeBool(v bool) {
	e.scratch[0] = 0
	if v {
		e.scratch[0] = 1
	}
	e.Write(e.scratch[:1])
}

func (e *Encoder) writeComplex64(v complex64) {
	var b [8]byte
	binary.LittleEndian.PutUint32(b[:4], math.Float32bits(real(v)))
	binary.LittleEndian.PutUint32(b[4:], math.Float32bits(imag(v)))
	e.Write(b[:])
}

func (e *Encoder) writeComplex128(v complex128) {
	var b [16]byte
	binary.LittleEndian.PutUint64(b[:8], math.Float64bits(real(v)))
	binary.LittleEndian.PutUint64(b[8:], math.Float64bits(imag(v)))
	e.Write(b[:])
}

func (e *Encoder) WriteString(v string) {
	e.WriteUvarint(uint64(len(v)))
	e.Write(ToBytes(v))
}

func (e *Encoder) WriteTagged(tag uint64, body []byte) {
	e.WriteUvarint(tag)
	e.WriteUvarint(uint64(len(body)))
	if len(body) > 0 {
		e.Write(body)
	}
}
