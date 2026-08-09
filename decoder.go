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

var decoders = &sync.Pool{New: func() any {
	return NewDecoder(newReader(nil))
}}

func Unmarshal(b []byte, v any) (err error) {
	d := decoders.Get().(*Decoder)
	d.reader.(*sliceReader).Reset(b) // Reset the reader
	err = d.Decode(v)
	d.arena = nil
	decoders.Put(d)
	return
}

type Decoder struct {
	reader  reader
	slice   *sliceReader
	arena   []byte
	scratch [10]byte
	last    reflect.Type
	codec   Codec
}

func NewDecoder(r io.Reader) *Decoder {
	reader := newReader(r)
	d := &Decoder{reader: reader}
	d.slice, _ = reader.(*sliceReader)
	return d
}

func (d *Decoder) Decode(v any) (err error) {
	d.arena = nil
	rv := reflect.Indirect(reflect.ValueOf(v))
	if !rv.CanAddr() {
		return errors.New("binary: can only decode to pointer type")
	}
	t := rv.Type()
	c := d.codec
	if t != d.last {
		if c, err = scan(t); err != nil {
			return
		}
		d.last = t
		d.codec = c
	}
	err = c.DecodeTo(d, rv)
	return
}

func (d *Decoder) Read(b []byte) (int, error) {
	if d.slice != nil {
		return d.slice.Read(b)
	}
	return d.reader.Read(b)
}

func (d *Decoder) ReadUvarint() (uint64, error) {
	if d.slice != nil {
		return d.slice.ReadUvarint()
	}
	return d.reader.ReadUvarint()
}

func (d *Decoder) ReadVarint() (int64, error) {
	if d.slice != nil {
		return d.slice.ReadVarint()
	}
	return d.reader.ReadVarint()
}

func (d *Decoder) ReadUint16() (out uint16, err error) {
	var b []byte
	if b, err = d.Slice(2); err == nil {
		out = binary.LittleEndian.Uint16(b)
	}
	return
}

func (d *Decoder) ReadUint32() (out uint32, err error) {
	var b []byte
	if b, err = d.Slice(4); err == nil {
		out = binary.LittleEndian.Uint32(b)
	}
	return
}

func (d *Decoder) ReadUint64() (out uint64, err error) {
	var b []byte
	if b, err = d.Slice(8); err == nil {
		out = binary.LittleEndian.Uint64(b)
	}
	return
}

func (d *Decoder) ReadFloat32() (out float32, err error) {
	var v uint32
	if v, err = d.ReadUint32(); err == nil {
		out = math.Float32frombits(v)
	}
	return
}

func (d *Decoder) ReadFloat64() (out float64, err error) {
	var v uint64
	if v, err = d.ReadUint64(); err == nil {
		out = math.Float64frombits(v)
	}
	return
}

func (d *Decoder) ReadBool() (bool, error) {
	if d.slice != nil {
		b, err := d.slice.ReadByte()
		return b == 1, err
	}
	b, err := d.reader.ReadByte()
	return b == 1, err
}

func (d *Decoder) ReadString() (out string, err error) {
	return d.readString("")
}

func (d *Decoder) readString(old string) (string, error) {
	b, err := d.ReadSlice()
	if err != nil {
		return "", err
	}
	if len(old) == len(b) && bytes.Equal(ToBytes(old), b) {
		return old, nil
	}
	return string(b), nil
}

func (d *Decoder) readComplex64() (out complex64, err error) {
	b, err := d.Slice(8)
	if err != nil {
		return 0, err
	}
	return complex(math.Float32frombits(binary.LittleEndian.Uint32(b)), math.Float32frombits(binary.LittleEndian.Uint32(b[4:]))), nil
}

func (d *Decoder) readComplex128() (out complex128, err error) {
	b, err := d.Slice(16)
	if err != nil {
		return 0, err
	}
	return complex(math.Float64frombits(binary.LittleEndian.Uint64(b)), math.Float64frombits(binary.LittleEndian.Uint64(b[8:]))), nil
}

func (d *Decoder) Slice(n int) ([]byte, error) {
	if n < 0 {
		return nil, io.ErrUnexpectedEOF
	}
	if d.slice != nil {
		return d.slice.Slice(n)
	}
	return d.reader.Slice(n)
}

// Available reports the remaining bytes for an in-memory decoder, or -1 for a stream.
func (d *Decoder) Available() int {
	if d.slice == nil {
		return -1
	}
	return d.slice.Len()
}

func decodeLength(n uint64) (int, error) {
	if n > uint64(^uint(0)>>1) {
		return 0, io.ErrUnexpectedEOF
	}
	return int(n), nil
}

func validateSliceLength(t reflect.Type, n int) error {
	if n < 0 {
		return io.ErrUnexpectedEOF
	}
	size := t.Elem().Size()
	maxInt := uint64(^uint(0) >> 1)
	if uint64(n) > maxInt/2 || (size != 0 && uint64(n) > maxInt/uint64(size)) {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func (d *Decoder) ensureAvailable(n int) error {
	if n < 0 {
		return io.ErrUnexpectedEOF
	}
	if available := d.Available(); available >= 0 && n > available {
		return io.EOF
	}
	return nil
}

func (d *Decoder) ensureElements(n, minBytes int) error {
	if n < 0 || minBytes < 0 {
		return io.ErrUnexpectedEOF
	}
	if available := d.Available(); available >= 0 && minBytes > 0 && n > available/minBytes {
		return io.EOF
	}
	return nil
}

func (d *Decoder) mapCapacity(n int) int {
	if d.Available() < 0 {
		return 0
	}
	return n
}

func (d *Decoder) readSlice(n uint64) ([]byte, error) {
	l, err := decodeLength(n)
	if err != nil {
		return nil, err
	}
	if d.slice != nil {
		return d.slice.Slice(l)
	}
	return d.reader.Slice(l)
}

func (d *Decoder) ReadSlice() (b []byte, err error) {
	if d.slice != nil {
		var l uint64
		if l, err = d.slice.ReadUvarint(); err != nil {
			return
		}
		return d.readSlice(l)
	}
	var l uint64
	if l, err = d.ReadUvarint(); err != nil {
		return
	}
	return d.readSlice(l)
}

func (d *Decoder) ReadTagged() (tag uint64, body []byte, err error) {
	if d.slice != nil {
		if tag, err = d.slice.ReadUvarint(); err == nil {
			body, err = d.ReadSlice()
		}
		return
	}
	if tag, err = d.ReadUvarint(); err != nil {
		return
	}
	var l uint64
	if l, err = d.ReadUvarint(); err != nil {
		return
	}
	body, err = d.readSlice(l)
	return
}
