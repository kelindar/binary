// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"reflect"
	"sync"
)

// Reusable long-lived decoder pool.
var decoders = &sync.Pool{New: func() interface{} {
	return NewDecoder(newReader(nil))
}}

// Unmarshal decodes the payload from the binary format.
func Unmarshal(b []byte, v interface{}) (err error) {

	// Get the decoder from the pool, reset it
	d := decoders.Get().(*Decoder)
	d.reader.(*sliceReader).Reset(b) // Reset the reader

	// Decode and set the buffer if successful and free the decoder
	err = d.Decode(v)
	decoders.Put(d)
	return
}

// Decoder represents a binary decoder.
type Decoder struct {
	reader  reader
	scratch [10]byte
	schemas map[reflect.Type]Codec
}

// NewDecoder creates a binary decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		reader:  newReader(r),
		schemas: make(map[reflect.Type]Codec),
	}
}

// Decode decodes a value by reading from the underlying io.Reader.
func (d *Decoder) Decode(v interface{}) (err error) {
	rv := reflect.Indirect(reflect.ValueOf(v))
	if !rv.CanAddr() {
		return errors.New("binary: can only decode to pointer type")
	}

	// Scan the type (this will load from cache)
	var c Codec
	if c, err = scanToCache(rv.Type(), d.schemas); err == nil {
		err = c.DecodeTo(d, rv)
	}

	return
}

// Read reads a set of bytes
func (d *Decoder) Read(b []byte) (int, error) {
	return d.reader.Read(b)
}

// ReadUvarint reads a variable-length Uint64 from the buffer.
func (d *Decoder) ReadUvarint() (uint64, error) {
	return d.reader.ReadUvarint()
}

// ReadVarint reads a variable-length Int64 from the buffer.
func (d *Decoder) ReadVarint() (int64, error) {
	return d.reader.ReadVarint()
}

// ReadUint16 reads a uint16
func (d *Decoder) ReadUint16() (out uint16, err error) {
	var b []byte
	if b, err = d.reader.Slice(2); err == nil {
		_ = b[1] // bounds check hint to compiler
		out = (uint16(b[0]) | uint16(b[1])<<8)
	}
	return
}

// ReadUint32 reads a uint32
func (d *Decoder) ReadUint32() (out uint32, err error) {
	var b []byte
	if b, err = d.reader.Slice(4); err == nil {
		_ = b[3] // bounds check hint to compiler
		out = (uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
	}
	return
}

// ReadUint64 reads a uint64
func (d *Decoder) ReadUint64() (out uint64, err error) {
	var b []byte
	if b, err = d.reader.Slice(8); err == nil {
		_ = b[7] // bounds check hint to compiler
		out = (uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
			uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56)
	}
	return
}

// ReadFloat32 reads a float32
func (d *Decoder) ReadFloat32() (out float32, err error) {
	var v uint32
	if v, err = d.ReadUint32(); err == nil {
		out = math.Float32frombits(v)
	}
	return
}

// ReadFloat64 reads a float64
func (d *Decoder) ReadFloat64() (out float64, err error) {
	var v uint64
	if v, err = d.ReadUint64(); err == nil {
		out = math.Float64frombits(v)
	}
	return
}

// ReadBool reads a single boolean value from the slice.
func (d *Decoder) ReadBool() (bool, error) {
	b, err := d.reader.ReadByte()
	return b == 1, err
}

// ReadString a string prefixed with a variable-size integer size.
func (d *Decoder) ReadString() (out string, err error) {
	var b []byte
	if b, err = d.ReadSlice(); err == nil {
		out = string(b)
	}
	return
}

// ReadComplex reads a complex64
func (d *Decoder) readComplex64() (out complex64, err error) {
	err = binary.Read(d.reader, binary.LittleEndian, &out)
	return
}

// ReadComplex reads a complex128
func (d *Decoder) readComplex128() (out complex128, err error) {
	err = binary.Read(d.reader, binary.LittleEndian, &out)
	return
}

// Slice selects a sub-slice of next bytes. This is similar to Read() but does not
// actually perform a copy, but simply uses the underlying slice (if available) and
// returns a sub-slice pointing to the same array. Since this requires access
// to the underlying data, this is only available for a slice reader.
func (d *Decoder) Slice(n int) ([]byte, error) {
	return d.reader.Slice(n)
}

// ReadSlice reads a varint prefixed sub-slice without copying and returns the underlying
// byte slice.
func (d *Decoder) ReadSlice() (b []byte, err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil {
		b, err = d.Slice(int(l))
	}
	return
}
