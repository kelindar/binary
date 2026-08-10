// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

// Copyright 2012 The Go Authors. All rights reserved.

package binary

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

var overflow = errors.New("binary: varint overflows a 64-bit integer")

type reader interface {
	io.Reader
	io.ByteReader
	Slice(n int) (buffer []byte, err error)
	ReadUvarint() (uint64, error)
	ReadVarint() (int64, error)
}

func newReader(r io.Reader) reader {
	if r != nil && isNilInterface(r) {
		return newSliceReader(nil)
	}
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

type sliceReader struct {
	buffer []byte
	offset int // current reading index
}

func newSliceReader(b []byte) *sliceReader { return &sliceReader{b, 0} }

func (r *sliceReader) Len() int {
	if n := len(r.buffer) - r.offset; n > 0 {
		return n
	}
	return 0
}

func (r *sliceReader) Size() int64 { return int64(len(r.buffer)) }

func (r *sliceReader) Read(b []byte) (n int, err error) {
	if r.offset >= len(r.buffer) {
		return 0, io.EOF
	}
	n = copy(b, r.buffer[r.offset:])
	r.offset += n
	return
}

func (r *sliceReader) ReadByte() (byte, error) {
	if r.offset >= len(r.buffer) {
		return 0, io.EOF
	}
	b := r.buffer[r.offset]
	r.offset++
	return b, nil
}

func (r *sliceReader) Slice(n int) ([]byte, error) {
	if n < 0 || n > len(r.buffer)-r.offset {
		return nil, io.EOF
	}
	cur := r.offset
	r.offset += n
	return r.buffer[cur:r.offset], nil
}

func (r *sliceReader) ReadUvarint() (uint64, error) {
	var value uint64
	for shift := uint(0); shift < 64; shift += 7 {
		if r.offset >= len(r.buffer) {
			return 0, io.EOF
		}
		b := r.buffer[r.offset]
		r.offset++
		if b < 0x80 {
			if shift == 63 && b > 1 {
				return value, overflow
			}
			return value | uint64(b)<<shift, nil
		}
		value |= uint64(b&0x7f) << shift
	}
	return value, overflow
}

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

func readVarints[T integer](r *sliceReader, values []T, signed bool) error {
	buffer := r.buffer
	offset := r.offset
	for i := range values {
		if offset >= len(buffer) {
			r.offset = offset
			return io.EOF
		}
		b := buffer[offset]
		offset++
		if b < 0x80 {
			values[i] = varintValue[T](uint64(b), signed)
			continue
		}
		if offset < len(buffer) && buffer[offset] < 0x80 {
			x := uint64(b&0x7f) | uint64(buffer[offset])<<7
			offset++
			values[i] = varintValue[T](x, signed)
			continue
		}
		x := uint64(b & 0x7f)
		for s := 7; s < binary.MaxVarintLen64*7; s += 7 {
			if offset >= len(buffer) {
				r.offset = offset
				return io.EOF
			}
			b = buffer[offset]
			offset++
			if b < 0x80 {
				if s == binary.MaxVarintLen64*7-7 && b > 1 {
					r.offset = offset
					return overflow
				}
				x |= uint64(b) << s
				values[i] = varintValue[T](x, signed)
				goto next
			}
			x |= uint64(b&0x7f) << s
		}
		r.offset = offset
		return overflow
	next:
	}
	r.offset = offset
	return nil
}

func varintValue[T integer](value uint64, signed bool) T {
	if signed {
		return T(decodeVarint(value))
	}
	return T(value)
}

func decodeVarint(x uint64) int64 { return int64(x>>1) ^ -int64(x&1) }

func (r *sliceReader) ReadVarint() (int64, error) {
	ux, err := r.ReadUvarint() // ok to continue in presence of error
	return decodeVarint(ux), err
}

func (r *sliceReader) Reset(b []byte) { r.buffer, r.offset = b, 0 }

// --------------------------------------- Stream Reader ---------------------------------------

type streamReader struct {
	Reader
}

type Reader interface {
	io.Reader
	io.ByteReader
}

func newStreamReader(r io.Reader) *streamReader {
	rdr, ok := r.(Reader)
	if !ok {
		rdr = bufio.NewReader(r)
	}
	return &streamReader{
		Reader: rdr,
	}
}

func (r *streamReader) Slice(n int) (buffer []byte, err error) {
	if n < 0 || uint64(n) > uint64(^uint(0)>>1)/2 {
		return nil, io.ErrUnexpectedEOF
	}
	if n > 64<<10 {
		buffer := bytes.NewBuffer(make([]byte, 0, 64<<10))
		_, err = buffer.ReadFrom(io.LimitReader(r, int64(n)))
		if err == nil && buffer.Len() != n {
			err = io.EOF
		}
		return buffer.Bytes(), err
	}
	buffer = make([]byte, n)
	_, err = io.ReadFull(r, buffer)
	return
}

func (r *streamReader) ReadUvarint() (uint64, error) {
	return binary.ReadUvarint(r)
}

func (r *streamReader) ReadVarint() (int64, error) {
	return binary.ReadVarint(r)
}
