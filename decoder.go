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
	err = binary.Read(d.reader, binary.LittleEndian, &out)
	return
}

func (d *Decoder) readComplex128() (out complex128, err error) {
	err = binary.Read(d.reader, binary.LittleEndian, &out)
	return
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

func (d *Decoder) readSlice(n uint64) ([]byte, error) {
	if n > uint64(^uint(0)>>1) {
		return nil, io.ErrUnexpectedEOF
	}
	if d.slice != nil {
		return d.slice.Slice(int(n))
	}
	return d.reader.Slice(int(n))
}

func (d *Decoder) ReadSlice() (b []byte, err error) {
	if d.slice != nil {
		var l uint64
		if l, err = d.slice.ReadUvarint(); err != nil {
			return
		}
		if l > uint64(^uint(0)>>1) {
			return nil, io.ErrUnexpectedEOF
		}
		return d.slice.Slice(int(l))
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
