// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"reflect"
	"sync"
)

// Reusable long-lived encoder pool.
var encoders = &sync.Pool{New: func() interface{} {
	return &Encoder{
		Order: DefaultEndian,
	}
}}

// Marshal encodes the payload into binary format.
func Marshal(v interface{}) (output []byte, err error) {
	var buffer bytes.Buffer

	// Get the encoder from the pool, reset it
	e := encoders.Get().(*Encoder)
	e.out = &buffer
	e.err = nil

	// Encode and set the buffer if successful
	if err = e.Encode(v); err == nil {
		output = buffer.Bytes()
	}

	// Put the encoder back when we're finished
	encoders.Put(e)
	return
}

// Encoder represents a binary encoder.
type Encoder struct {
	scratch [10]byte
	out     io.Writer
	err     error
	Order   binary.ByteOrder
}

// NewEncoder creates a new encoder.
func NewEncoder(out io.Writer) *Encoder {
	return &Encoder{
		Order: DefaultEndian,
		out:   out,
	}
}

// Encode encodes the value to the binary format.
func (e *Encoder) Encode(v interface{}) (err error) {

	// Scan the type (this will load from cache)
	rv := reflect.Indirect(reflect.ValueOf(v))
	var c Codec
	if c, err = scan(rv.Type()); err != nil {
		return
	}

	// Encode the value
	if err = c.EncodeTo(e, rv); err == nil {
		err = e.err
	}
	return
}

// Write writes the contents of p into the buffer.
func (e *Encoder) Write(p []byte) {
	if e.err == nil {
		_, e.err = e.out.Write(p)
	}
}

// WriteVarint writes a variable size integer
func (e *Encoder) WriteVarint(v int64) {
	n := binary.PutVarint(e.scratch[:], v)
	e.Write(e.scratch[:n])
}

// WriteUvarint writes a variable size unsigned integer
func (e *Encoder) WriteUvarint(v uint64) {
	n := binary.PutUvarint(e.scratch[:], v)
	e.Write(e.scratch[:n])
}

// WriteUint16 writes a Uint16
func (e *Encoder) WriteUint16(v uint16) {
	e.Order.PutUint16(e.scratch[0:2], v)
	e.Write(e.scratch[:2])
}

// WriteUint32 writes a Uint32
func (e *Encoder) WriteUint32(v uint32) {
	e.Order.PutUint32(e.scratch[0:4], v)
	e.Write(e.scratch[:4])
}

// WriteUint64 writes a Uint64
func (e *Encoder) WriteUint64(v uint64) {
	e.Order.PutUint64(e.scratch[0:8], v)
	e.Write(e.scratch[:8])
}

// WriteFloat32 a 32-bit floating point number
func (e *Encoder) WriteFloat32(v float32) {
	e.Order.PutUint32(e.scratch[0:4], math.Float32bits(v))
	e.Write(e.scratch[:4])
}

// WriteFloat64 a 64-bit floating point number
func (e *Encoder) WriteFloat64(v float64) {
	e.Order.PutUint64(e.scratch[0:8], math.Float64bits(v))
	e.Write(e.scratch[:8])
}

// Writes a boolean value
func (e *Encoder) writeBool(v bool) {
	e.scratch[0] = 0
	if v {
		e.scratch[0] = 1
	}
	e.Write(e.scratch[:1])
}

// Writes a complex number
func (e *Encoder) writeComplex(v complex128) {
	e.err = binary.Write(e.out, e.Order, v)
}

// Writes a string
func (e *Encoder) writeString(v string) {
	e.WriteUvarint(uint64(len(v)))
	e.Write(stringToBinary(v))
}
