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
	"unsafe"
)

// Constants
var (
	LittleEndian = binary.LittleEndian
	BigEndian    = binary.BigEndian
)

// Codec represents a single part Codec, which can encode and decode something.
type Codec interface {
	EncodeTo(*Encoder, reflect.Value) error
	DecodeTo(*Decoder, reflect.Value) error
}

// ------------------------------------------------------------------------------

type reflectArrayCodec struct {
	elemCodec Codec // The codec of the array's elements
}

// Encode encodes a value into the encoder.
func (c *reflectArrayCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Type().Len()
	for i := range l {
		v := rv.Index(i)
		if err = c.elemCodec.EncodeTo(e, v); err != nil {
			return
		}
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *reflectArrayCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	l := rv.Type().Len()
	for i := range l {
		v := rv.Index(i)
		if err = c.elemCodec.DecodeTo(d, v); err != nil {
			return
		}
	}
	return
}

// ------------------------------------------------------------------------------

type reflectSliceCodec struct {
	elemCodec Codec // The codec of the slice's elements
}

func resizeSlice(rv reflect.Value, n int) {
	if rv.Cap() >= n {
		rv.SetLen(n)
		return
	}
	rv.Set(reflect.MakeSlice(rv.Type(), n, n))
}

func sliceData(p unsafe.Pointer) unsafe.Pointer {
	return *(*unsafe.Pointer)(p)
}

func sliceCap(p unsafe.Pointer) int {
	return *(*int)(unsafe.Add(p, 2*unsafe.Sizeof(uintptr(0))))
}

func sliceLen(p unsafe.Pointer) int {
	return *(*int)(unsafe.Add(p, unsafe.Sizeof(uintptr(0))))
}

// Encode encodes a value into the encoder.
func (c *reflectSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	for i := range l {
		v := rv.Index(i)
		if err = c.elemCodec.EncodeTo(e, v); err != nil {
			return
		}
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *reflectSliceCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil {
		n := int(l)
		resizeSlice(rv, n)
		for i := range n {
			v := rv.Index(i)
			if err = c.elemCodec.DecodeTo(d, v); err != nil {
				return
			}
		}
	}
	return
}

// ------------------------------------------------------------------------------

type reflectSliceOfPtrCodec struct {
	elemCodec Codec        // The codec of the slice's elements
	elemType  reflect.Type // The type of the element
}

// Encode encodes a value into the encoder.
func (c *reflectSliceOfPtrCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	for i := range l {
		v := rv.Index(i)
		isNil := v.IsNil()
		e.writeBool(isNil)
		if !isNil {
			if err = c.elemCodec.EncodeTo(e, reflect.Indirect(v)); err != nil {
				return
			}
		}
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *reflectSliceOfPtrCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	var isNil bool
	if l, err = d.ReadUvarint(); err != nil {
		return
	}
	resizeSlice(rv, int(l))
	for i := 0; i < int(l); i++ {
		ptr := rv.Index(i)
		isNil, err = d.ReadBool()
		switch {
		case err != nil:
			return
		case isNil:
			ptr.SetZero()
			continue
		}
		if ptr.IsNil() {
			ptr.Set(reflect.New(c.elemType))
		}
		if err = c.elemCodec.DecodeTo(d, ptr.Elem()); err != nil {
			return
		}
	}
	return
}

// ------------------------------------------------------------------------------

type byteSliceCodec struct{}

// Encode encodes a value into the encoder.
func (c *byteSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	b := rv.Bytes()
	e.WriteUvarint(uint64(len(b)))
	e.Write(b)
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *byteSliceCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil {
		resizeSlice(rv, int(l))
		if l > 0 {
			_, err = d.Read(rv.Bytes())
		}
	}
	return
}

// ------------------------------------------------------------------------------

type stringSliceCodec struct {
	array bool
}

// Encode encodes a value into the encoder.
func (c *stringSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	if l >= 8 {
		if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil {
			size := 0
			if !c.array {
				size = uvarintSize(uint64(l))
			}
			for i := range l {
				size += uvarintSize(uint64(rv.Index(i).Len())) + rv.Index(i).Len()
			}
			out.Grow(size)
			buffer := out.AvailableBuffer()
			if !c.array {
				buffer = binary.AppendUvarint(buffer, uint64(l))
			}
			for i := range l {
				value := rv.Index(i).String()
				buffer = binary.AppendUvarint(buffer, uint64(len(value)))
				buffer = append(buffer, value...)
			}
			e.Write(buffer)
			return
		}
	}
	if !c.array {
		e.WriteUvarint(uint64(l))
	}
	for i := range l {
		e.WriteString(rv.Index(i).String())
	}
	return
}

func uvarintSize(x uint64) int {
	size := 1
	for x >= 0x80 {
		x >>= 7
		size++
	}
	return size
}

// Decode decodes into a reflect value from the decoder.
func (c *stringSliceCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if c.array {
		l = uint64(rv.Len())
	} else {
		if l, err = d.ReadUvarint(); err != nil {
			return
		}
		resizeSlice(rv, int(l))
	}
	for i := 0; i < int(l); i++ {
		var value string
		if value, err = d.ReadString(); err != nil {
			return
		}
		rv.Index(i).SetString(value)
	}
	return
}

// ------------------------------------------------------------------------------

type boolSliceCodec struct{}

// Encode encodes a value into the encoder.
func (c *boolSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	if l > 0 {
		v := rv.Interface().([]bool)
		e.Write(boolsToBinary(&v))
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *boolSliceCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil {
		resizeSlice(rv, int(l))
		if l > 0 {
			v := rv.Interface().([]bool)
			_, err = d.Read(boolsToBinary(&v))
		}
	}
	return
}

// ------------------------------------------------------------------------------

type floatSliceCodec struct {
	elemSize uintptr
	array    bool
}

// Encode encodes a value into the encoder.
func (c *floatSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	if l >= 8 {
		if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil && (!c.array || rv.CanAddr()) {
			size := l * int(c.elemSize)
			if !c.array {
				size += uvarintSize(uint64(l))
			}
			out.Grow(size)
			buffer := out.AvailableBuffer()
			if !c.array {
				buffer = binary.AppendUvarint(buffer, uint64(l))
			}
			var base unsafe.Pointer
			if c.array {
				base = unsafe.Pointer(rv.UnsafeAddr())
			} else {
				base = rv.UnsafePointer()
			}
			switch c.elemSize {
			case 4:
				for _, value := range unsafe.Slice((*float32)(base), l) {
					buffer = binary.LittleEndian.AppendUint32(buffer, math.Float32bits(value))
				}
			case 8:
				for _, value := range unsafe.Slice((*float64)(base), l) {
					buffer = binary.LittleEndian.AppendUint64(buffer, math.Float64bits(value))
				}
			}
			e.Write(buffer)
			return
		}
	}
	if !c.array {
		e.WriteUvarint(uint64(l))
	}
	for i := range l {
		if c.elemSize == 4 {
			e.WriteFloat32(float32(rv.Index(i).Float()))
		} else {
			e.WriteFloat64(rv.Index(i).Float())
		}
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *floatSliceCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	var n int
	if c.array {
		n = rv.Len()
	} else {
		if l, err = d.ReadUvarint(); err != nil {
			return
		}
		maxInt := int(^uint(0) >> 1)
		if l > uint64(maxInt) || int(l) > maxInt/int(c.elemSize) {
			return io.ErrUnexpectedEOF
		}
		n = int(l)
		resizeSlice(rv, n)
	}
	if n == 0 {
		return nil
	}
	data, err := d.Slice(n * int(c.elemSize))
	if err != nil {
		return err
	}
	var base unsafe.Pointer
	if c.array {
		base = unsafe.Pointer(rv.UnsafeAddr())
	} else {
		base = rv.UnsafePointer()
	}
	switch c.elemSize {
	case 4:
		values := unsafe.Slice((*float32)(base), n)
		for i := range values {
			values[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[i*4:]))
		}
	case 8:
		values := unsafe.Slice((*float64)(base), n)
		for i := range values {
			values[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[i*8:]))
		}
	}
	return nil
}

// ------------------------------------------------------------------------------

type complexSliceCodec struct {
	elemSize uintptr
	array    bool
}

// Encode encodes a value into the encoder.
func (c *complexSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	if l >= 8 {
		if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil && (!c.array || rv.CanAddr()) {
			size := l * int(c.elemSize)
			if !c.array {
				size += uvarintSize(uint64(l))
			}
			out.Grow(size)
			buffer := out.AvailableBuffer()
			if !c.array {
				buffer = binary.AppendUvarint(buffer, uint64(l))
			}
			var base unsafe.Pointer
			if c.array {
				base = unsafe.Pointer(rv.UnsafeAddr())
			} else {
				base = rv.UnsafePointer()
			}
			switch c.elemSize {
			case 8:
				for _, value := range unsafe.Slice((*complex64)(base), l) {
					buffer = binary.LittleEndian.AppendUint32(buffer, math.Float32bits(real(value)))
					buffer = binary.LittleEndian.AppendUint32(buffer, math.Float32bits(imag(value)))
				}
			case 16:
				for _, value := range unsafe.Slice((*complex128)(base), l) {
					buffer = binary.LittleEndian.AppendUint64(buffer, math.Float64bits(real(value)))
					buffer = binary.LittleEndian.AppendUint64(buffer, math.Float64bits(imag(value)))
				}
			}
			e.Write(buffer)
			return
		}
	}
	if !c.array {
		e.WriteUvarint(uint64(l))
	}
	for i := range l {
		value := rv.Index(i).Complex()
		if c.elemSize == 8 {
			e.writeComplex64(complex64(value))
		} else {
			e.writeComplex128(value)
		}
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *complexSliceCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	var n int
	if c.array {
		n = rv.Len()
	} else {
		if l, err = d.ReadUvarint(); err != nil {
			return
		}
		maxInt := int(^uint(0) >> 1)
		if l > uint64(maxInt) || int(l) > maxInt/int(c.elemSize) {
			return io.ErrUnexpectedEOF
		}
		n = int(l)
		resizeSlice(rv, n)
	}
	if n == 0 {
		return nil
	}
	data, err := d.Slice(n * int(c.elemSize))
	if err != nil {
		return err
	}
	var base unsafe.Pointer
	if c.array {
		base = unsafe.Pointer(rv.UnsafeAddr())
	} else {
		base = rv.UnsafePointer()
	}
	switch c.elemSize {
	case 8:
		values := unsafe.Slice((*complex64)(base), n)
		for i := range values {
			offset := i * 8
			values[i] = complex(
				math.Float32frombits(binary.LittleEndian.Uint32(data[offset:])),
				math.Float32frombits(binary.LittleEndian.Uint32(data[offset+4:])),
			)
		}
	case 16:
		values := unsafe.Slice((*complex128)(base), n)
		for i := range values {
			offset := i * 16
			values[i] = complex(
				math.Float64frombits(binary.LittleEndian.Uint64(data[offset:])),
				math.Float64frombits(binary.LittleEndian.Uint64(data[offset+8:])),
			)
		}
	}
	return nil
}

// ------------------------------------------------------------------------------

type varintSliceCodec struct {
	elemSize uintptr
}

// Encode encodes a value into the encoder.
func (c *varintSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil {
		out.Grow(2 * l)
		buffer := out.AvailableBuffer()
		base := rv.UnsafePointer()
		switch c.elemSize {
		case 1:
			for _, v := range unsafe.Slice((*int8)(base), l) {
				buffer = binary.AppendVarint(buffer, int64(v))
			}
		case 2:
			for _, v := range unsafe.Slice((*int16)(base), l) {
				buffer = binary.AppendVarint(buffer, int64(v))
			}
		case 4:
			for _, v := range unsafe.Slice((*int32)(base), l) {
				buffer = binary.AppendVarint(buffer, int64(v))
			}
		case 8:
			for _, v := range unsafe.Slice((*int64)(base), l) {
				buffer = binary.AppendVarint(buffer, v)
			}
		}
		e.Write(buffer)
		return
	}
	base := rv.UnsafePointer()
	switch c.elemSize {
	case 1:
		for _, v := range unsafe.Slice((*int8)(base), l) {
			e.WriteVarint(int64(v))
		}
	case 2:
		for _, v := range unsafe.Slice((*int16)(base), l) {
			e.WriteVarint(int64(v))
		}
	case 4:
		for _, v := range unsafe.Slice((*int32)(base), l) {
			e.WriteVarint(int64(v))
		}
	case 8:
		for _, v := range unsafe.Slice((*int64)(base), l) {
			e.WriteVarint(v)
		}
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *varintSliceCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil {
		n := int(l)
		resizeSlice(rv, n)
		err = decodeVarints(d, rv.UnsafePointer(), n, c.elemSize)
	}
	return
}

// ------------------------------------------------------------------------------

func decodeVarints(d *Decoder, base unsafe.Pointer, n int, elemSize uintptr) (err error) {
	if d.slice != nil {
		switch elemSize {
		case 1:
			return readVarints(d.slice, unsafe.Slice((*int8)(base), n))
		case 2:
			return readVarints(d.slice, unsafe.Slice((*int16)(base), n))
		case 4:
			return readVarints(d.slice, unsafe.Slice((*int32)(base), n))
		case 8:
			return readVarints(d.slice, unsafe.Slice((*int64)(base), n))
		}
	}

	var v int64
	switch elemSize {
	case 1:
		values := unsafe.Slice((*int8)(base), n)
		for i := range values {
			if v, err = d.ReadVarint(); err != nil {
				return
			}
			values[i] = int8(v)
		}
	case 2:
		values := unsafe.Slice((*int16)(base), n)
		for i := range values {
			if v, err = d.ReadVarint(); err != nil {
				return
			}
			values[i] = int16(v)
		}
	case 4:
		values := unsafe.Slice((*int32)(base), n)
		for i := range values {
			if v, err = d.ReadVarint(); err != nil {
				return
			}
			values[i] = int32(v)
		}
	case 8:
		values := unsafe.Slice((*int64)(base), n)
		for i := range values {
			if values[i], err = d.ReadVarint(); err != nil {
				return
			}
		}
	}
	return
}

type varuintSliceCodec struct {
	elemSize uintptr
}

// Encode encodes a value into the encoder.
func (c *varuintSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	encodeVaruints(e, rv.UnsafePointer(), rv.Len(), c.elemSize)
	return nil
}

func encodeVaruints(e *Encoder, base unsafe.Pointer, l int, elemSize uintptr) {
	e.WriteUvarint(uint64(l))
	if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil {
		out.Grow(2 * l)
		buffer := out.AvailableBuffer()
		switch elemSize {
		case 2:
			for _, v := range unsafe.Slice((*uint16)(base), l) {
				buffer = binary.AppendUvarint(buffer, uint64(v))
			}
		case 4:
			for _, v := range unsafe.Slice((*uint32)(base), l) {
				buffer = binary.AppendUvarint(buffer, uint64(v))
			}
		case 8:
			for _, v := range unsafe.Slice((*uint64)(base), l) {
				buffer = binary.AppendUvarint(buffer, v)
			}
		}
		e.Write(buffer)
		return
	}
	switch elemSize {
	case 2:
		for _, v := range unsafe.Slice((*uint16)(base), l) {
			e.WriteUvarint(uint64(v))
		}
	case 4:
		for _, v := range unsafe.Slice((*uint32)(base), l) {
			e.WriteUvarint(uint64(v))
		}
	case 8:
		for _, v := range unsafe.Slice((*uint64)(base), l) {
			e.WriteUvarint(v)
		}
	}
}

// Decode decodes into a reflect value from the decoder.
func (c *varuintSliceCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil {
		n := int(l)
		resizeSlice(rv, n)
		err = decodeVaruints(d, rv.UnsafePointer(), n, c.elemSize)
	}
	return
}

func decodeVaruints(d *Decoder, base unsafe.Pointer, n int, elemSize uintptr) (err error) {
	if d.slice != nil {
		switch elemSize {
		case 2:
			return readUvarints(d.slice, unsafe.Slice((*uint16)(base), n))
		case 4:
			return readUvarints(d.slice, unsafe.Slice((*uint32)(base), n))
		case 8:
			return readUvarints(d.slice, unsafe.Slice((*uint64)(base), n))
		}
	}

	var v uint64
	switch elemSize {
	case 2:
		values := unsafe.Slice((*uint16)(base), n)
		for i := range values {
			if v, err = d.ReadUvarint(); err != nil {
				return
			}
			values[i] = uint16(v)
		}
	case 4:
		values := unsafe.Slice((*uint32)(base), n)
		for i := range values {
			if v, err = d.ReadUvarint(); err != nil {
				return
			}
			values[i] = uint32(v)
		}
	case 8:
		values := unsafe.Slice((*uint64)(base), n)
		for i := range values {
			if values[i], err = d.ReadUvarint(); err != nil {
				return
			}
		}
	}
	return
}

// ------------------------------------------------------------------------------

type reflectPointerCodec struct {
	elemCodec Codec
}

// Encode encodes a value into the encoder.
func (c *reflectPointerCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	if rv.IsNil() {
		e.writeBool(true)
		return
	}

	e.writeBool(false)
	err = c.elemCodec.EncodeTo(e, rv.Elem())
	if err != nil {
		return err
	}
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *reflectPointerCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	isNil, err := d.ReadBool()
	switch {
	case err != nil:
		return err
	case isNil:
		rv.SetZero()
		return
	}

	if rv.IsNil() {
		rv.Set(reflect.New(rv.Type().Elem()))
	}

	return c.elemCodec.DecodeTo(d, rv.Elem())
}

// ------------------------------------------------------------------------------

type reflectStructCodec []fieldCodec

const (
	fieldOffsetMask = uint64(1)<<56 - 1
	fieldKindShift  = 56
	fieldKindMask   = uint64(0x1f) << fieldKindShift
	fieldDirect     = uint64(1) << 61
	fieldIncluded   = uint64(1) << 62
	fieldWritable   = uint64(1) << 63
	fieldVaruint2   = reflect.Kind(27)
	fieldVaruint4   = reflect.Kind(28)
	fieldVaruint8   = reflect.Kind(29)
	fieldByteSlice  = reflect.Kind(30)
)

type fieldCodec struct {
	Field uint64
	Codec Codec
}

func (f *fieldCodec) offset() uintptr {
	return uintptr(f.Field & fieldOffsetMask)
}

func (f *fieldCodec) kind() reflect.Kind {
	return reflect.Kind((f.Field & fieldKindMask) >> fieldKindShift)
}

// Encode encodes a value into the encoder.
func (c reflectStructCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	if rv.CanAddr() && len(c) > 0 && c[0].Field&fieldDirect != 0 {
		base := unsafe.Pointer(rv.UnsafeAddr())
		for i := range c {
			field := &c[i]
			if field.Field&fieldIncluded == 0 {
				continue
			}
			pointer := unsafe.Add(base, field.offset())
			switch field.kind() {
			case reflect.String:
				e.WriteString(*(*string)(pointer))
			case reflect.Bool:
				e.writeBool(*(*bool)(pointer))
			case reflect.Int:
				e.WriteVarint(int64(*(*int)(pointer)))
			case reflect.Int8:
				e.WriteVarint(int64(*(*int8)(pointer)))
			case reflect.Int16:
				e.WriteVarint(int64(*(*int16)(pointer)))
			case reflect.Int32:
				e.WriteVarint(int64(*(*int32)(pointer)))
			case reflect.Int64:
				e.WriteVarint(*(*int64)(pointer))
			case reflect.Uint:
				e.WriteUvarint(uint64(*(*uint)(pointer)))
			case reflect.Uint8:
				e.WriteUvarint(uint64(*(*uint8)(pointer)))
			case reflect.Uint16:
				e.WriteUvarint(uint64(*(*uint16)(pointer)))
			case reflect.Uint32:
				e.WriteUvarint(uint64(*(*uint32)(pointer)))
			case reflect.Uint64:
				e.WriteUvarint(*(*uint64)(pointer))
			case reflect.Complex64:
				e.writeComplex64(*(*complex64)(pointer))
			case reflect.Complex128:
				e.writeComplex128(*(*complex128)(pointer))
			case reflect.Float32:
				e.WriteFloat32(*(*float32)(pointer))
			case reflect.Float64:
				e.WriteFloat64(*(*float64)(pointer))
			case fieldByteSlice:
				b := *(*[]byte)(pointer)
				e.WriteUvarint(uint64(len(b)))
				e.Write(b)
			case fieldVaruint2, fieldVaruint4, fieldVaruint8:
				size := uintptr(8)
				switch field.kind() {
				case fieldVaruint2:
					size = 2
				case fieldVaruint4:
					size = 4
				}
				encodeVaruints(e, sliceData(pointer), sliceLen(pointer), size)
			default:
				if err = field.Codec.EncodeTo(e, rv.Field(i)); err != nil {
					return
				}
			}
		}
		return
	}

	for i := range c {
		field := &c[i]
		if field.Field&fieldIncluded == 0 {
			continue
		}
		if err = field.Codec.EncodeTo(e, rv.Field(i)); err != nil {
			return
		}
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c reflectStructCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	if rv.CanAddr() && len(c) > 0 && c[0].Field&fieldDirect != 0 {
		base := unsafe.Pointer(rv.UnsafeAddr())
		for i := range c {
			field := &c[i]
			if field.Field&fieldWritable == 0 {
				continue
			}
			pointer := unsafe.Add(base, field.offset())
			switch field.kind() {
			case reflect.String:
				var value string
				value, err = d.ReadString()
				if err != nil {
					return
				}
				*(*string)(pointer) = value
			case reflect.Bool:
				var value bool
				value, err = d.ReadBool()
				if err != nil {
					return
				}
				*(*bool)(pointer) = value
			case reflect.Int:
				var value int64
				value, err = d.ReadVarint()
				if err != nil {
					return
				}
				*(*int)(pointer) = int(value)
			case reflect.Int8:
				var value int64
				value, err = d.ReadVarint()
				if err != nil {
					return
				}
				*(*int8)(pointer) = int8(value)
			case reflect.Int16:
				var value int64
				value, err = d.ReadVarint()
				if err != nil {
					return
				}
				*(*int16)(pointer) = int16(value)
			case reflect.Int32:
				var value int64
				value, err = d.ReadVarint()
				if err != nil {
					return
				}
				*(*int32)(pointer) = int32(value)
			case reflect.Int64:
				var value int64
				value, err = d.ReadVarint()
				if err != nil {
					return
				}
				*(*int64)(pointer) = value
			case reflect.Uint:
				var value uint64
				value, err = d.ReadUvarint()
				if err != nil {
					return
				}
				*(*uint)(pointer) = uint(value)
			case reflect.Uint8:
				var value uint64
				value, err = d.ReadUvarint()
				if err != nil {
					return
				}
				*(*uint8)(pointer) = uint8(value)
			case reflect.Uint16:
				var value uint64
				value, err = d.ReadUvarint()
				if err != nil {
					return
				}
				*(*uint16)(pointer) = uint16(value)
			case reflect.Uint32:
				var value uint64
				value, err = d.ReadUvarint()
				if err != nil {
					return
				}
				*(*uint32)(pointer) = uint32(value)
			case reflect.Uint64:
				var value uint64
				value, err = d.ReadUvarint()
				if err != nil {
					return
				}
				*(*uint64)(pointer) = value
			case reflect.Complex64:
				var value complex64
				value, err = d.readComplex64()
				if err != nil {
					return
				}
				*(*complex64)(pointer) = value
			case reflect.Complex128:
				var value complex128
				value, err = d.readComplex128()
				if err != nil {
					return
				}
				*(*complex128)(pointer) = value
			case reflect.Float32:
				var value float32
				value, err = d.ReadFloat32()
				if err != nil {
					return
				}
				*(*float32)(pointer) = value
			case reflect.Float64:
				var value float64
				value, err = d.ReadFloat64()
				if err != nil {
					return
				}
				*(*float64)(pointer) = value
			case fieldByteSlice:
				var length uint64
				if length, err = d.ReadUvarint(); err != nil {
					return
				}
				n := int(length)
				if sliceCap(pointer) >= n {
					*(*int)(unsafe.Add(pointer, unsafe.Sizeof(uintptr(0)))) = n
				} else {
					resizeSlice(rv.Field(i), n)
				}
				if n > 0 {
					_, err = d.Read(unsafe.Slice((*byte)(sliceData(pointer)), n))
					if err != nil {
						return
					}
				}
			case fieldVaruint2, fieldVaruint4, fieldVaruint8:
				var length uint64
				if length, err = d.ReadUvarint(); err != nil {
					return
				}
				n := int(length)
				if sliceCap(pointer) >= n {
					*(*int)(unsafe.Add(pointer, unsafe.Sizeof(uintptr(0)))) = n
				} else {
					resizeSlice(rv.Field(i), n)
				}
				size := uintptr(8)
				switch field.kind() {
				case fieldVaruint2:
					size = 2
				case fieldVaruint4:
					size = 4
				}
				if err = decodeVaruints(d, sliceData(pointer), n, size); err != nil {
					return
				}
			default:
				if err = field.Codec.DecodeTo(d, rv.Field(i)); err != nil {
					return
				}
			}
		}
		return
	}

	for i := range c {
		field := &c[i]
		if field.Field&fieldWritable != 0 {
			err = field.Codec.DecodeTo(d, rv.Field(i))
		}
		if err != nil {
			return
		}
	}
	return
}

// ------------------------------------------------------------------------------

// customCodec represents a custom binary marshaling.
type customCodec struct {
	marshaler      *reflect.Method
	unmarshaler    *reflect.Method
	ptrMarshaler   *reflect.Method
	ptrUnmarshaler *reflect.Method
}

// Encode encodes a value into the encoder.
func (c *customCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	m := c.GetMarshalBinary(rv)
	if m == nil {
		return errors.New("MarshalBinary not found on " + rv.Type().String())
	}

	// Special-case for pointers
	if rv.Kind() == reflect.Ptr {
		e.writeBool(rv.IsNil())
		if rv.IsNil() {
			return nil
		}
	}

	ret := m.Call([]reflect.Value{})
	if !ret[1].IsNil() {
		err = ret[1].Interface().(error)
		return
	}

	// Write the marshaled byte slice
	buffer := ret[0].Bytes()
	e.WriteUvarint(uint64(len(buffer)))
	e.Write(buffer)
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *customCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	m := c.GetUnmarshalBinary(rv)

	// Special case for pointers
	if rv.Kind() == reflect.Ptr {
		isNil, err := d.ReadBool()
		if err != nil {
			return err
		}
		if isNil {
			rv.SetZero()
			return nil
		}

		rv.Set(reflect.New(rv.Type().Elem()))
	}

	var l uint64
	if l, err = d.ReadUvarint(); err == nil {
		buffer := make([]byte, l)
		_, err = d.Read(buffer)
		ret := m.Call([]reflect.Value{reflect.ValueOf(buffer)})
		if !ret[0].IsNil() {
			err = ret[0].Interface().(error)
		}

	}
	return
}

func (c *customCodec) GetMarshalBinary(rv reflect.Value) *reflect.Value {
	if c.marshaler != nil {
		m := rv.Method(c.marshaler.Index)
		return &m
	}

	if c.ptrMarshaler != nil {
		m := rv.Addr().Method(c.ptrMarshaler.Index)
		return &m
	}

	return nil
}

func (c *customCodec) GetUnmarshalBinary(rv reflect.Value) *reflect.Value {
	if c.unmarshaler != nil {
		m := rv.Method(c.unmarshaler.Index)
		return &m
	}

	if c.ptrUnmarshaler != nil {
		m := rv.Addr().Method(c.ptrUnmarshaler.Index)
		return &m
	}

	return nil
}

// ------------------------------------------------------------------------------

type reflectMapCodec struct {
	key Codec // Codec for the key
	val Codec // Codec for the value
}

type stringBytesMapCodec struct{}

func (stringBytesMapCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	m := rv.Interface().(map[string][]byte)
	if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil {
		size := 10
		for key, value := range m {
			size += 2 + len(key) + 10 + len(value)
		}
		out.Grow(size)
		buffer := out.AvailableBuffer()
		buffer = binary.AppendUvarint(buffer, uint64(len(m)))
		for key, value := range m {
			buffer = append(buffer, byte(len(key)), byte(len(key)>>8))
			buffer = append(buffer, key...)
			buffer = binary.AppendUvarint(buffer, uint64(len(value)))
			buffer = append(buffer, value...)
		}
		e.Write(buffer)
		return
	}
	e.WriteUvarint(uint64(len(m)))
	for key, value := range m {
		e.WriteUint16(uint16(len(key)))
		e.Write(ToBytes(key))
		e.WriteUvarint(uint64(len(value)))
		e.Write(value)
	}
	return
}

func (stringBytesMapCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err != nil {
		return
	}

	m := rv.Interface().(map[string][]byte)
	if m == nil {
		m = make(map[string][]byte, int(l))
		rv.Set(reflect.ValueOf(m))
	} else {
		clear(m)
	}

	for i := 0; i < int(l); i++ {
		var size uint16
		if size, err = d.ReadUint16(); err != nil {
			return
		}
		var keyBytes []byte
		if keyBytes, err = d.Slice(int(size)); err != nil {
			return
		}
		key := string(keyBytes)

		var length uint64
		if length, err = d.ReadUvarint(); err != nil {
			return
		}
		var value []byte
		if length > 0 {
			if length > uint64(^uint(0)>>1) {
				return io.ErrUnexpectedEOF
			}
			value = make([]byte, int(length))
			if _, err = d.Read(value); err != nil {
				return
			}
		}
		m[key] = value
	}
	return
}

type stringStringMapCodec struct{}

func (stringStringMapCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	m := rv.Interface().(map[string]string)
	if len(m) >= 8 {
		if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil {
			size := 10
			for key, value := range m {
				size += 2 + len(key) + 10 + len(value)
			}
			out.Grow(size)
			buffer := out.AvailableBuffer()
			buffer = binary.AppendUvarint(buffer, uint64(len(m)))
			for key, value := range m {
				buffer = append(buffer, byte(len(key)), byte(len(key)>>8))
				buffer = append(buffer, key...)
				buffer = binary.AppendUvarint(buffer, uint64(len(value)))
				buffer = append(buffer, value...)
			}
			e.Write(buffer)
			return
		}
	}
	e.WriteUvarint(uint64(len(m)))
	for key, value := range m {
		e.WriteUint16(uint16(len(key)))
		e.Write(ToBytes(key))
		e.WriteString(value)
	}
	return
}

type uint64MapCodec struct{}

func (uint64MapCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	m := rv.Interface().(map[uint64]uint64)
	if len(m) >= 8 {
		if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil {
			size := uvarintSize(uint64(len(m)))
			for key, value := range m {
				size += 8 + uvarintSize(key) + uvarintSize(value)
			}
			out.Grow(size)
			buffer := out.AvailableBuffer()
			buffer = binary.AppendUvarint(buffer, uint64(len(m)))
			for key, value := range m {
				buffer = binary.LittleEndian.AppendUint64(buffer, key)
				buffer = binary.AppendUvarint(buffer, value)
			}
			e.Write(buffer)
			return
		}
	}
	e.WriteUvarint(uint64(len(m)))
	for key, value := range m {
		e.WriteUint64(key)
		e.WriteUvarint(value)
	}
	return
}

func (uint64MapCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err != nil {
		return
	}
	m := rv.Interface().(map[uint64]uint64)
	if m == nil {
		m = make(map[uint64]uint64, int(l))
		rv.Set(reflect.ValueOf(m))
	} else {
		clear(m)
	}
	for i := 0; i < int(l); i++ {
		var key, value uint64
		if key, err = d.ReadUint64(); err != nil {
			return
		}
		if value, err = d.ReadUvarint(); err != nil {
			return
		}
		m[key] = value
	}
	return
}

type stringUint64MapCodec struct{}

func (stringUint64MapCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	m := rv.Interface().(map[string]uint64)
	if len(m) >= 8 {
		if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil {
			size := uvarintSize(uint64(len(m)))
			for key, value := range m {
				size += 2 + len(key) + uvarintSize(value)
			}
			out.Grow(size)
			buffer := out.AvailableBuffer()
			buffer = binary.AppendUvarint(buffer, uint64(len(m)))
			for key, value := range m {
				buffer = append(buffer, byte(len(key)), byte(len(key)>>8))
				buffer = append(buffer, key...)
				buffer = binary.AppendUvarint(buffer, value)
			}
			e.Write(buffer)
			return
		}
	}
	e.WriteUvarint(uint64(len(m)))
	for key, value := range m {
		e.WriteUint16(uint16(len(key)))
		e.Write(ToBytes(key))
		e.WriteUvarint(value)
	}
	return
}

func (stringUint64MapCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err != nil {
		return
	}
	m := rv.Interface().(map[string]uint64)
	if m == nil {
		m = make(map[string]uint64, int(l))
		rv.Set(reflect.ValueOf(m))
	} else {
		clear(m)
	}
	for i := 0; i < int(l); i++ {
		var size uint16
		if size, err = d.ReadUint16(); err != nil {
			return
		}
		var keyBytes []byte
		if keyBytes, err = d.Slice(int(size)); err != nil {
			return
		}
		var value uint64
		if value, err = d.ReadUvarint(); err != nil {
			return
		}
		m[string(keyBytes)] = value
	}
	return
}

func (stringStringMapCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err != nil {
		return
	}

	m := rv.Interface().(map[string]string)
	if m == nil {
		m = make(map[string]string, int(l))
		rv.Set(reflect.ValueOf(m))
	} else {
		clear(m)
	}

	for i := 0; i < int(l); i++ {
		var size uint16
		if size, err = d.ReadUint16(); err != nil {
			return
		}
		var keyBytes []byte
		if keyBytes, err = d.Slice(int(size)); err != nil {
			return
		}

		var value string
		if value, err = d.ReadString(); err != nil {
			return
		}
		m[string(keyBytes)] = value
	}
	return
}

// Encode encodes a value into the encoder.
func (c *reflectMapCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	e.WriteUvarint(uint64(rv.Len()))
	iter := rv.MapRange()
	for iter.Next() {
		key, value := iter.Key(), iter.Value()
		if err = c.writeKey(e, key); err != nil {
			return err
		}

		if err = c.val.EncodeTo(e, value); err != nil {
			return err
		}
	}
	return
}

// Decode decodes into a reflect value from the decoder.
func (c *reflectMapCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil {
		t := rv.Type()
		vt := t.Elem()
		if rv.IsNil() {
			rv.Set(reflect.MakeMapWithSize(t, int(l)))
		} else {
			rv.Clear()
		}
		kv := reflect.New(t.Key()).Elem()
		vv := reflect.New(vt).Elem()
		for i := 0; i < int(l); i++ {
			kv.SetZero()
			if err = c.readKey(d, kv); err != nil {
				return
			}

			vv.SetZero()
			if err = c.val.DecodeTo(d, vv); err != nil {
				return
			}

			rv.SetMapIndex(kv, vv)
		}
	}
	return
}

// Write key writes a key to the encoder
func (c *reflectMapCodec) writeKey(e *Encoder, key reflect.Value) (err error) {
	switch key.Kind() {
	case reflect.Int16:
		e.WriteUint16(uint16(key.Int()))
	case reflect.Int32:
		e.WriteUint32(uint32(key.Int()))
	case reflect.Int64:
		e.WriteUint64(uint64(key.Int()))
	case reflect.Uint16:
		e.WriteUint16(uint16(key.Uint()))
	case reflect.Uint32:
		e.WriteUint32(uint32(key.Uint()))
	case reflect.Uint64:
		e.WriteUint64(key.Uint())
	case reflect.String:
		str := key.String()
		e.WriteUint16(uint16(len(str)))
		e.Write(ToBytes(str))
	default:
		err = c.key.EncodeTo(e, key)
	}
	return
}

// Read key reads a key from the decoder
func (c *reflectMapCodec) readKey(d *Decoder, key reflect.Value) (err error) {
	switch key.Kind() {

	case reflect.Int16:
		var v uint16
		if v, err = d.ReadUint16(); err == nil {
			key.SetInt(int64(int16(v)))
		}
	case reflect.Int32:
		var v uint32
		if v, err = d.ReadUint32(); err == nil {
			key.SetInt(int64(int32(v)))
		}
	case reflect.Int64:
		var v uint64
		if v, err = d.ReadUint64(); err == nil {
			key.SetInt(int64(v))
		}

	case reflect.Uint16:
		var v uint16
		if v, err = d.ReadUint16(); err == nil {
			key.SetUint(uint64(v))
		}
	case reflect.Uint32:
		var v uint32
		if v, err = d.ReadUint32(); err == nil {
			key.SetUint(uint64(v))
		}
	case reflect.Uint64:
		var v uint64
		if v, err = d.ReadUint64(); err == nil {
			key.SetUint(v)
		}

	// String keys must have max length of 65536
	case reflect.String:
		var l uint16
		var b []byte

		if l, err = d.ReadUint16(); err == nil {
			if b, err = d.Slice(int(l)); err == nil {
				key.SetString(string(b))
			}
		}

	// Default to a reflect-based approach
	default:
		err = c.key.DecodeTo(d, key)
	}
	return
}

// ------------------------------------------------------------------------------

type stringCodec struct{}

// Encode encodes a value into the encoder.
func (c *stringCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.WriteString(rv.String())
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *stringCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var s string
	if s, err = d.ReadString(); err == nil {
		rv.SetString(s)
	}
	return
}

// ------------------------------------------------------------------------------

type boolCodec struct{}

// Encode encodes a value into the encoder.
func (c *boolCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.writeBool(rv.Bool())
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *boolCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var out bool
	if out, err = d.ReadBool(); err == nil {
		rv.SetBool(out)
	}
	return
}

// ------------------------------------------------------------------------------

type varintCodec struct{}

// Encode encodes a value into the encoder.
func (c *varintCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.WriteVarint(rv.Int())
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *varintCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var v int64
	if v, err = d.ReadVarint(); err != nil {
		return
	}
	rv.SetInt(v)
	return
}

// ------------------------------------------------------------------------------

type varuintCodec struct{}

// Encode encodes a value into the encoder.
func (c *varuintCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.WriteUvarint(rv.Uint())
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *varuintCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var v uint64
	if v, err = d.ReadUvarint(); err != nil {
		return
	}
	rv.SetUint(v)
	return
}

// ------------------------------------------------------------------------------

type complex64Codec struct{}

// Encode encodes a value into the encoder.
func (c *complex64Codec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.writeComplex64(complex64(rv.Complex()))
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *complex64Codec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var out complex64
	out, err = d.readComplex64()
	rv.SetComplex(complex128(out))
	return
}

// ------------------------------------------------------------------------------

type complex128Codec struct{}

// Encode encodes a value into the encoder.
func (c *complex128Codec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.writeComplex128(rv.Complex())
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *complex128Codec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var out complex128
	out, err = d.readComplex128()
	rv.SetComplex(out)
	return
}

// ------------------------------------------------------------------------------

type float32Codec struct{}

// Encode encodes a value into the encoder.
func (c *float32Codec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.WriteFloat32(float32(rv.Float()))
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *float32Codec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var v float32
	if v, err = d.ReadFloat32(); err == nil {
		rv.SetFloat(float64(v))
	}
	return
}

// ------------------------------------------------------------------------------

type float64Codec struct{}

// Encode encodes a value into the encoder.
func (c *float64Codec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.WriteFloat64(rv.Float())
	return nil
}

// Decode decodes into a reflect value from the decoder.
func (c *float64Codec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var v float64
	if v, err = d.ReadFloat64(); err == nil {
		rv.SetFloat(v)
	}
	return
}
