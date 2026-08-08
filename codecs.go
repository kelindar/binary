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

var (
	LittleEndian = binary.LittleEndian
	BigEndian    = binary.BigEndian
)

type Codec interface {
	EncodeTo(*Encoder, reflect.Value) error
	DecodeTo(*Decoder, reflect.Value) error
}

type reflectCollectionCodec struct {
	elemCodec Codec
	array     bool
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

func (c *reflectCollectionCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	if !c.array {
		e.WriteUvarint(uint64(l))
	}
	if codec, ok := c.elemCodec.(*reflectStructCodec); ok {
		for i := range l {
			if err = codec.EncodeTo(e, rv.Index(i)); err != nil {
				return
			}
		}
		return
	}
	for i := range l {
		v := rv.Index(i)
		if err = c.elemCodec.EncodeTo(e, v); err != nil {
			return
		}
	}
	return
}

func (c *reflectCollectionCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	n := rv.Len()
	if !c.array {
		var l uint64
		if l, err = d.ReadUvarint(); err != nil {
			return
		}
		n = int(l)
		resizeSlice(rv, n)
	}
	if codec, ok := c.elemCodec.(*reflectStructCodec); ok {
		for i := range n {
			if err = codec.DecodeTo(d, rv.Index(i)); err != nil {
				return
			}
		}
		return
	}
	for i := range n {
		if err = c.elemCodec.DecodeTo(d, rv.Index(i)); err != nil {
			return
		}
	}
	return
}

type reflectSliceOfPtrCodec struct {
	elemCodec Codec        // The codec of the slice's elements
	elemType  reflect.Type // The type of the element
}

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

type byteSliceCodec struct{}

func (c *byteSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	b := rv.Bytes()
	e.WriteUvarint(uint64(len(b)))
	e.Write(b)
	return
}
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

type stringSliceCodec struct {
	array bool
}

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
		if value, err = d.readString(rv.Index(i).String()); err != nil {
			return
		}
		rv.Index(i).SetString(value)
	}
	return
}

type boolSliceCodec struct{}

func (c *boolSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.WriteUvarint(uint64(l))
	if l > 0 {
		v := rv.Interface().([]bool)
		e.Write(boolsToBinary(&v))
	}
	return
}
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

type fixedSliceCodec struct {
	elemSize uintptr
	array    bool
	complex  bool
}

func (c *fixedSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
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
			if c.complex {
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
			} else {
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
			}
			e.Write(buffer)
			return
		}
	}
	if !c.array {
		e.WriteUvarint(uint64(l))
	}
	for i := range l {
		if c.complex {
			value := rv.Index(i).Complex()
			if c.elemSize == 8 {
				e.writeComplex64(complex64(value))
			} else {
				e.writeComplex128(value)
			}
		} else if c.elemSize == 4 {
			e.WriteFloat32(float32(rv.Index(i).Float()))
		} else {
			e.WriteFloat64(rv.Index(i).Float())
		}
	}
	return
}
func (c *fixedSliceCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
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
	if c.complex {
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
	} else {
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
	}
	return nil
}

type varSliceCodec struct {
	elemSize uintptr
	signed   bool
}

func (c *varSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	if c.signed {
		encodeVarints(e, rv.UnsafePointer(), rv.Len(), c.elemSize)
	} else {
		encodeVaruints(e, rv.UnsafePointer(), rv.Len(), c.elemSize)
	}
	return nil
}
func encodeVarints(e *Encoder, base unsafe.Pointer, l int, elemSize uintptr) {
	e.WriteUvarint(uint64(l))
	if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil {
		out.Grow(2 * l)
		buffer := out.AvailableBuffer()
		switch elemSize {
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
	switch elemSize {
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
func (c *varSliceCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil {
		n := int(l)
		resizeSlice(rv, n)
		if c.signed {
			err = decodeVarints(d, rv.UnsafePointer(), n, c.elemSize)
		} else {
			err = decodeVaruints(d, rv.UnsafePointer(), n, c.elemSize)
		}
	}
	return
}
func decodeVarints(d *Decoder, base unsafe.Pointer, n int, elemSize uintptr) (err error) {
	switch elemSize {
	case 1:
		values := unsafe.Slice((*int8)(base), n)
		if d.slice != nil {
			return readVarints(d.slice, values, true)
		}
		return readSigned(d, values)
	case 2:
		values := unsafe.Slice((*int16)(base), n)
		if d.slice != nil {
			return readVarints(d.slice, values, true)
		}
		return readSigned(d, values)
	case 4:
		values := unsafe.Slice((*int32)(base), n)
		if d.slice != nil {
			return readVarints(d.slice, values, true)
		}
		return readSigned(d, values)
	case 8:
		values := unsafe.Slice((*int64)(base), n)
		if d.slice != nil {
			return readVarints(d.slice, values, true)
		}
		return readSigned(d, values)
	}
	return
}
func readSigned[T integer](d *Decoder, values []T) error {
	for i := range values {
		value, err := d.ReadVarint()
		if err != nil {
			return err
		}
		values[i] = T(value)
	}
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
func decodeVaruints(d *Decoder, base unsafe.Pointer, n int, elemSize uintptr) (err error) {
	switch elemSize {
	case 2:
		values := unsafe.Slice((*uint16)(base), n)
		if d.slice != nil {
			return readVarints(d.slice, values, false)
		}
		return readUnsigned(d, values)
	case 4:
		values := unsafe.Slice((*uint32)(base), n)
		if d.slice != nil {
			return readVarints(d.slice, values, false)
		}
		return readUnsigned(d, values)
	case 8:
		values := unsafe.Slice((*uint64)(base), n)
		if d.slice != nil {
			return readVarints(d.slice, values, false)
		}
		return readUnsigned(d, values)
	}
	return
}
func readUnsigned[T integer](d *Decoder, values []T) error {
	for i := range values {
		value, err := d.ReadUvarint()
		if err != nil {
			return err
		}
		values[i] = T(value)
	}
	return nil
}

type reflectPointerCodec struct {
	elemCodec Codec
}

func (c *reflectPointerCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	if rv.IsNil() {
		e.writeBool(true)
		return nil
	}
	e.writeBool(false)
	return c.elemCodec.EncodeTo(e, rv.Elem())
}
func (c *reflectPointerCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	isNil, err := d.ReadBool()
	switch {
	case err != nil:
		return err
	case isNil:
		rv.SetZero()
		return nil
	}
	if rv.IsNil() {
		rv.Set(reflect.New(rv.Type().Elem()))
	}
	return c.elemCodec.DecodeTo(d, rv.Elem())
}

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
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				e.WriteVarint(rv.Field(i).Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				e.WriteUvarint(rv.Field(i).Uint())
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
				value, err = d.readString(*(*string)(pointer))
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
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				var value int64
				value, err = d.ReadVarint()
				if err != nil {
					return
				}
				switch field.kind() {
				case reflect.Int:
					*(*int)(pointer) = int(value)
				case reflect.Int8:
					*(*int8)(pointer) = int8(value)
				case reflect.Int16:
					*(*int16)(pointer) = int16(value)
				case reflect.Int32:
					*(*int32)(pointer) = int32(value)
				case reflect.Int64:
					*(*int64)(pointer) = value
				}
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				var value uint64
				value, err = d.ReadUvarint()
				if err != nil {
					return
				}
				switch field.kind() {
				case reflect.Uint:
					*(*uint)(pointer) = uint(value)
				case reflect.Uint8:
					*(*uint8)(pointer) = uint8(value)
				case reflect.Uint16:
					*(*uint16)(pointer) = uint16(value)
				case reflect.Uint32:
					*(*uint32)(pointer) = uint32(value)
				case reflect.Uint64:
					*(*uint64)(pointer) = value
				}
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

type customCodec struct {
	marshaler      *reflect.Method
	unmarshaler    *reflect.Method
	ptrMarshaler   *reflect.Method
	ptrUnmarshaler *reflect.Method
}

func (c *customCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	m := c.GetMarshalBinary(rv)
	if m == nil {
		return errors.New("MarshalBinary not found on " + rv.Type().String())
	}
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
	buffer := ret[0].Bytes()
	e.WriteUvarint(uint64(len(buffer)))
	e.Write(buffer)
	return
}

func (c *customCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	m := c.GetUnmarshalBinary(rv)
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
	return customMethod(rv, c.marshaler, c.ptrMarshaler)
}

func (c *customCodec) GetUnmarshalBinary(rv reflect.Value) *reflect.Value {
	return customMethod(rv, c.unmarshaler, c.ptrUnmarshaler)
}

func customMethod(rv reflect.Value, method, pointer *reflect.Method) *reflect.Value {
	if method != nil {
		m := rv.Method(method.Index)
		return &m
	}
	if pointer != nil {
		m := rv.Addr().Method(pointer.Index)
		return &m
	}
	return nil
}

type reflectMapCodec struct {
	key Codec // Codec for the key
	val Codec // Codec for the value
}

func mapArena(d *Decoder) *[]byte {
	if d.arena != nil {
		return &d.arena
	}
	if d.slice == nil || d.slice.Len() == 0 {
		return nil
	}
	d.arena = make([]byte, d.slice.Len())
	return &d.arena
}

func arenaBytes(arena *[]byte, src []byte) []byte {
	dst := (*arena)[:len(src)]
	copy(dst, src)
	*arena = (*arena)[len(src):]
	return dst
}

func arenaString(arena *[]byte, src []byte) string {
	dst := arenaBytes(arena, src)
	return ToString(&dst)
}

type stringMapValue interface{ string | []byte | uint64 }

type stringMapCodec[V stringMapValue] struct{}

func stringMapValueSize[V stringMapValue](value V) int {
	switch value := any(value).(type) {
	case string:
		return uvarintSize(uint64(len(value))) + len(value)
	case []byte:
		return uvarintSize(uint64(len(value))) + len(value)
	case uint64:
		return uvarintSize(value)
	}
	return 0
}

func appendStringMapValue[V stringMapValue](buffer []byte, value V) []byte {
	switch value := any(value).(type) {
	case string:
		buffer = binary.AppendUvarint(buffer, uint64(len(value)))
		return append(buffer, value...)
	case []byte:
		buffer = binary.AppendUvarint(buffer, uint64(len(value)))
		return append(buffer, value...)
	case uint64:
		return binary.AppendUvarint(buffer, value)
	}
	return buffer
}

func readStringMapValue[V stringMapValue](d *Decoder, arena *[]byte) (V, error) {
	var zero V
	switch any(zero).(type) {
	case string:
		b, err := d.ReadSlice()
		if err != nil {
			return zero, err
		}
		if arena != nil {
			return any(arenaString(arena, b)).(V), nil
		}
		return any(string(b)).(V), nil
	case []byte:
		b, err := d.ReadSlice()
		if err != nil {
			return zero, err
		}
		if len(b) == 0 {
			return zero, nil
		}
		if arena != nil {
			b = arenaBytes(arena, b)
		}
		return any(b).(V), nil
	case uint64:
		value, err := d.ReadUvarint()
		return any(value).(V), err
	}
	return zero, nil
}

func (stringMapCodec[V]) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	m := rv.Interface().(map[string]V)
	if len(m) >= 8 {
		if out, ok := e.out.(*bytes.Buffer); ok && e.err == nil {
			size := uvarintSize(uint64(len(m)))
			for key, value := range m {
				size += 2 + len(key) + stringMapValueSize(value)
			}
			out.Grow(size)
			buffer := out.AvailableBuffer()
			buffer = binary.AppendUvarint(buffer, uint64(len(m)))
			for key, value := range m {
				buffer = append(buffer, byte(len(key)), byte(len(key)>>8))
				buffer = append(buffer, key...)
				buffer = appendStringMapValue(buffer, value)
			}
			e.Write(buffer)
			return
		}
	}
	e.WriteUvarint(uint64(len(m)))
	for key, value := range m {
		e.WriteUint16(uint16(len(key)))
		e.Write(ToBytes(key))
		switch value := any(value).(type) {
		case string:
			e.WriteString(value)
		case []byte:
			e.WriteUvarint(uint64(len(value)))
			e.Write(value)
		case uint64:
			e.WriteUvarint(value)
		}
	}
	return
}

func (stringMapCodec[V]) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err != nil {
		return
	}
	m := rv.Interface().(map[string]V)
	if m == nil {
		m = make(map[string]V, int(l))
		rv.Set(reflect.ValueOf(m))
	} else {
		clear(m)
	}
	arena := mapArena(d)
	for i := 0; i < int(l); i++ {
		var size uint16
		if size, err = d.ReadUint16(); err != nil {
			return
		}
		keyBytes, readErr := d.Slice(int(size))
		if err = readErr; err != nil {
			return
		}
		var key string
		if arena != nil {
			key = arenaString(arena, keyBytes)
		} else {
			key = string(keyBytes)
		}
		value, readErr := readStringMapValue[V](d, arena)
		if err = readErr; err != nil {
			return
		}
		m[key] = value
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

func (c *reflectMapCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	var l uint64
	if l, err = d.ReadUvarint(); err == nil {
		t := rv.Type()
		vt := t.Elem()
		_, plainString := c.val.(*primitiveCodec)
		plainString = plainString && vt.Kind() == reflect.String
		_, plainBytes := c.val.(*byteSliceCodec)
		var arena *[]byte
		if t.Key().Kind() == reflect.String || plainString || plainBytes {
			arena = mapArena(d)
		}
		if rv.IsNil() {
			rv.Set(reflect.MakeMapWithSize(t, int(l)))
		} else {
			rv.Clear()
		}
		kv := reflect.New(t.Key()).Elem()
		vv := reflect.New(vt).Elem()
		for i := 0; i < int(l); i++ {
			kv.SetZero()
			if err = c.readKey(d, kv, arena); err != nil {
				return
			}
			vv.SetZero()
			if arena != nil && plainString {
				var b []byte
				if b, err = d.ReadSlice(); err == nil {
					vv.SetString(arenaString(arena, b))
				}
			} else if arena != nil && plainBytes {
				var b []byte
				if b, err = d.ReadSlice(); err == nil {
					if len(b) > 0 {
						vv.SetBytes(arenaBytes(arena, b))
					}
				}
			} else {
				err = c.val.DecodeTo(d, vv)
			}
			if err != nil {
				return
			}
			rv.SetMapIndex(kv, vv)
		}
	}
	return
}

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

func (c *reflectMapCodec) readKey(d *Decoder, key reflect.Value, arena *[]byte) (err error) {
	kind := key.Kind()
	switch kind {
	case reflect.Int16, reflect.Uint16:
		v, err := d.ReadUint16()
		if err != nil {
			return err
		}
		if kind == reflect.Int16 {
			key.SetInt(int64(int16(v)))
		} else {
			key.SetUint(uint64(v))
		}
	case reflect.Int32, reflect.Uint32:
		v, err := d.ReadUint32()
		if err != nil {
			return err
		}
		if kind == reflect.Int32 {
			key.SetInt(int64(int32(v)))
		} else {
			key.SetUint(uint64(v))
		}
	case reflect.Int64, reflect.Uint64:
		v, err := d.ReadUint64()
		if err != nil {
			return err
		}
		if kind == reflect.Int64 {
			key.SetInt(int64(v))
		} else {
			key.SetUint(v)
		}
	case reflect.String:
		l, err := d.ReadUint16()
		if err != nil {
			return err
		}
		b, err := d.Slice(int(l))
		if err != nil {
			return err
		}
		if arena != nil {
			key.SetString(arenaString(arena, b))
		} else {
			key.SetString(string(b))
		}
	default:
		return c.key.DecodeTo(d, key)
	}
	return nil
}

type primitiveCodec struct{}

func (*primitiveCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.String:
		e.WriteString(rv.String())
	case reflect.Bool:
		e.writeBool(rv.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.WriteVarint(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		e.WriteUvarint(rv.Uint())
	case reflect.Complex64:
		e.writeComplex64(complex64(rv.Complex()))
	case reflect.Complex128:
		e.writeComplex128(rv.Complex())
	case reflect.Float32:
		e.WriteFloat32(float32(rv.Float()))
	case reflect.Float64:
		e.WriteFloat64(rv.Float())
	}
	return nil
}

func (*primitiveCodec) DecodeTo(d *Decoder, rv reflect.Value) (err error) {
	switch rv.Kind() {
	case reflect.String:
		var value string
		if value, err = d.readString(rv.String()); err == nil {
			rv.SetString(value)
		}
	case reflect.Bool:
		var value bool
		if value, err = d.ReadBool(); err == nil {
			rv.SetBool(value)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var value int64
		if value, err = d.ReadVarint(); err == nil {
			rv.SetInt(value)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var value uint64
		if value, err = d.ReadUvarint(); err == nil {
			rv.SetUint(value)
		}
	case reflect.Complex64:
		var value complex64
		value, err = d.readComplex64()
		rv.SetComplex(complex128(value))
	case reflect.Complex128:
		var value complex128
		value, err = d.readComplex128()
		rv.SetComplex(value)
	case reflect.Float32:
		var value float32
		if value, err = d.ReadFloat32(); err == nil {
			rv.SetFloat(float64(value))
		}
	case reflect.Float64:
		var value float64
		if value, err = d.ReadFloat64(); err == nil {
			rv.SetFloat(value)
		}
	}
	return
}

type stringCodec = primitiveCodec
