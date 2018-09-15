// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"reflect"
)

// Reader represents the interface a reader should implement
type Reader interface {
	io.Reader
	io.ByteReader
}

// Unmarshal decodes the payload from the binary format.
func Unmarshal(b []byte, v interface{}) error {
	return NewDecoder(bytes.NewReader(b)).Decode(v)
}

// Decoder represents a binary decoder.
type Decoder struct {
	Order   binary.ByteOrder
	r       Reader
	scratch [10]byte
}

// NewDecoder creates a binary decoder.
func NewDecoder(r Reader) *Decoder {
	return &Decoder{
		Order: DefaultEndian,
		r:     r,
	}
}

// Decode decodes a value by reading from the underlying io.Reader.
func (d *Decoder) Decode(v interface{}) (err error) {
	rv := reflect.Indirect(reflect.ValueOf(v))
	if !rv.CanAddr() {
		return errors.New("binary: can only Decode to pointer type")
	}

	// Scan the type (this will load from cache)
	var c Codec
	if c, err = scan(rv.Type()); err == nil {
		err = c.DecodeTo(d, rv)
	}

	return
}

// ReadUvarint reads a variable-length Uint64 from the buffer.
func (d *Decoder) ReadUvarint() (uint64, error) {
	return binary.ReadUvarint(d.r)
}

// ReadVarint reads a variable-length Int64 from the buffer.
func (d *Decoder) ReadVarint() (int64, error) {
	return binary.ReadVarint(d.r)
}

// Read reads a set of bytes
func (d *Decoder) Read(b []byte) (int, error) {
	return d.r.Read(b)
}

// ReadUint16 reads a uint16
func (d *Decoder) ReadUint16() (out uint16, err error) {
	if _, err = d.r.Read(d.scratch[:2]); err == nil {
		out = d.Order.Uint16(d.scratch[:2])
	}
	return
}

// ReadUint32 reads a uint32
func (d *Decoder) ReadUint32() (out uint32, err error) {
	if _, err = d.r.Read(d.scratch[:4]); err == nil {
		out = d.Order.Uint32(d.scratch[:4])
	}
	return
}

// ReadUint64 reads a uint64
func (d *Decoder) ReadUint64() (out uint64, err error) {
	if _, err = d.r.Read(d.scratch[:8]); err == nil {
		out = d.Order.Uint64(d.scratch[:8])
	}
	return
}
