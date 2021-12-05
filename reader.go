// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

// This is a fork of bytes.Reader, originally licensed under
// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package binary

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// MaxVarintLenN is the maximum length of a varint-encoded N-bit integer.
const (
	maxVarintLen64 = 10 * 7
)

var overflow = errors.New("binary: varint overflows a 64-bit integer")

// reader represents a required contract for a decoder to work properly
type reader interface {
	io.Reader
	io.ByteReader
	Slice(n int) (buffer []byte, err error)
	ReadUvarint() (uint64, error)
	ReadVarint() (int64, error)
}

// newReader figures out the most efficient reader to use for the provided type
func newReader(r io.Reader) reader {
	switch v := r.(type) {
	case nil:
		return newSliceReader(nil)
	case *bytes.Buffer:
		return newSliceReader(v.Bytes())
	case *sliceReader:
		return v
	default:
		rdr, ok := r.(reader)
		if !ok {
			rdr = newStreamReader(r)
		}
		return rdr
	}
}

// --------------------------------------- Slice Reader ---------------------------------------

// sliceReader implements a reader that only reads from a slice
type sliceReader struct {
	buffer []byte
	offset int64 // current reading index
}

// newSliceReader returns a new Reader reading from b.
func newSliceReader(b []byte) *sliceReader {
	return &sliceReader{b, 0}
}

// Len returns the number of bytes of the unread portion of the
// slice.
func (r *sliceReader) Len() int {
	if r.offset >= int64(len(r.buffer)) {
		return 0
	}
	return int(int64(len(r.buffer)) - r.offset)
}

// Size returns the original length of the underlying byte slice.
// Size is the number of bytes available for reading via ReadAt.
// The returned value is always the same and is not affected by calls
// to any other method.
func (r *sliceReader) Size() int64 { return int64(len(r.buffer)) }

// Read implements the io.Reader interface.
func (r *sliceReader) Read(b []byte) (n int, err error) {
	if r.offset >= int64(len(r.buffer)) {
		return 0, io.EOF
	}

	n = copy(b, r.buffer[r.offset:])
	r.offset += int64(n)
	return
}

// ReadByte implements the io.ByteReader interface.
func (r *sliceReader) ReadByte() (byte, error) {
	if r.offset >= int64(len(r.buffer)) {
		return 0, io.EOF
	}

	b := r.buffer[r.offset]
	r.offset++
	return b, nil
}

// Slice selects a sub-slice of next bytes. This is similar to Read() but does not
// actually perform a copy, but simply uses the underlying slice (if available) and
// returns a sub-slice pointing to the same array. Since this requires access
// to the underlying data, this is only available for our default reader.
func (r *sliceReader) Slice(n int) ([]byte, error) {
	if r.offset+int64(n) > int64(len(r.buffer)) {
		return nil, io.EOF
	}

	cur := r.offset
	r.offset += int64(n)
	return r.buffer[cur:r.offset], nil
}

// ReadUvarint reads an encoded unsigned integer from r and returns it as a uint64.
func (r *sliceReader) ReadUvarint() (uint64, error) {
	var x uint64
	for s := 0; s < maxVarintLen64; s += 7 {
		if r.offset >= int64(len(r.buffer)) {
			return 0, io.EOF
		}

		b := r.buffer[r.offset]
		r.offset++
		if b < 0x80 {
			if s == maxVarintLen64-7 && b > 1 {
				return x, overflow
			}
			return x | uint64(b)<<s, nil
		}
		x |= uint64(b&0x7f) << s
	}
	return x, overflow
}

// ReadVarint reads an encoded signed integer from r and returns it as an int64.
func (r *sliceReader) ReadVarint() (int64, error) {
	ux, err := r.ReadUvarint() // ok to continue in presence of error
	x := int64(ux >> 1)
	if ux&1 != 0 {
		x = ^x
	}
	return x, err
}

// Reset resets the Reader to be reading from b.
func (r *sliceReader) Reset(b []byte) {
	r.buffer = b
	r.offset = 0
}

// --------------------------------------- Stream Reader ---------------------------------------

// streamReader represents a reader implementation for a generic reader (i.e. streams)
type streamReader struct {
	Reader
	scratch [10]byte
}

// Reader represents the interface a reader should implement.
type Reader interface {
	io.Reader
	io.ByteReader
}

// newStreamReader returns a new stream reader
func newStreamReader(r io.Reader) *streamReader {
	rdr, ok := r.(Reader)
	if !ok {
		rdr = bufio.NewReader(r)
	}

	return &streamReader{
		Reader: rdr,
	}
}

// Slice selects a sub-slice of next bytes.
func (r *streamReader) Slice(n int) (buffer []byte, err error) {
	if n <= 10 {
		buffer = r.scratch[:n]
	} else {
		buffer = make([]byte, n, n)
	}

	_, err = r.Read(buffer)
	return
}

// ReadUvarint reads an encoded unsigned integer from r and returns it as a uint64.
func (r *streamReader) ReadUvarint() (uint64, error) {
	return binary.ReadUvarint(r)
}

// ReadVarint reads a variable-length Int64 from the buffer.
func (r *streamReader) ReadVarint() (int64, error) {
	return binary.ReadVarint(r)
}
