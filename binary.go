package binary

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"unsafe"
)

// Constants
var (
	LittleEndian  = binary.LittleEndian
	BigEndian     = binary.BigEndian
	DefaultEndian = LittleEndian
)

var types = new(sync.Map)

type typeMeta struct {
	fields         []int
	marshaler      *reflect.Method
	unmarshaler    *reflect.Method
	ptrMarshaler   *reflect.Method
	ptrUnmarshaler *reflect.Method
}

func (m *typeMeta) GetMarshalBinary(rv reflect.Value) *reflect.Value {
	if m.marshaler != nil {
		m := rv.Method(m.marshaler.Index)
		return &m
	}

	if m.ptrMarshaler != nil {
		m := rv.Addr().Method(m.ptrMarshaler.Index)
		return &m
	}

	return nil
}

func (m *typeMeta) GetUnmarshalBinary(rv reflect.Value) *reflect.Value {
	if m.unmarshaler != nil {
		m := rv.Method(m.unmarshaler.Index)
		return &m
	}

	if m.ptrUnmarshaler != nil {
		m := rv.Addr().Method(m.ptrUnmarshaler.Index)
		return &m
	}

	return nil
}

func getMetadata(t reflect.Type, rv reflect.Value) (meta *typeMeta) {
	if f, ok := types.Load(t); ok {
		meta = f.(*typeMeta)
		return
	}

	l := rv.NumField()
	meta = new(typeMeta)
	for i := 0; i < l; i++ {
		if t.Field(i).Name != "_" {
			meta.fields = append(meta.fields, i)
		}
	}

	if m, ok := t.MethodByName("MarshalBinary"); ok {
		meta.marshaler = &m
	} else if m, ok := reflect.PtrTo(t).MethodByName("MarshalBinary"); ok {
		meta.ptrMarshaler = &m
	}

	if m, ok := t.MethodByName("UnmarshalBinary"); ok {
		meta.unmarshaler = &m
	} else if m, ok := reflect.PtrTo(t).MethodByName("UnmarshalBinary"); ok {
		meta.ptrUnmarshaler = &m
	}

	// Load or store again
	if f, ok := types.LoadOrStore(t, meta); ok {
		meta = f.(*typeMeta)
		return
	}
	return
}

// Marshal encodes the payload into binary format.
func Marshal(v interface{}) ([]byte, error) {
	b := &bytes.Buffer{}
	if err := NewEncoder(b).Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Unmarshal decodes the payload from the binary format.
func Unmarshal(b []byte, v interface{}) error {
	return NewDecoder(bytes.NewReader(b)).Decode(v)
}

type Encoder struct {
	buf    []byte
	Order  binary.ByteOrder
	strict bool
	w      io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		Order: DefaultEndian,
		w:     w,
		buf:   make([]byte, 10),
	}
}

// NewStrictEncoder creates an encoder similar to NewEncoder, however
// if this encoder attempts to encode a struct and the struct has no encodable
// fields an error is returned whereas the encoder returned from NewEncoder
// will simply not write anything to `w`.
func NewStrictEncoder(w io.Writer) *Encoder {
	e := NewEncoder(w)
	e.strict = true
	return e
}

func (b *Encoder) writeFlag(v bool) error {
	if v {
		b.buf[0] = 1
	} else {
		b.buf[0] = 0
	}

	_, err := b.w.Write(b.buf[:1])
	return err
}

func (b *Encoder) writeInt64(v int64) error {
	l := binary.PutVarint(b.buf, v)
	_, err := b.w.Write(b.buf[:l])
	return err
}

func (b *Encoder) writeUint64(v uint64) error {
	l := binary.PutUvarint(b.buf, v)
	_, err := b.w.Write(b.buf[:l])
	return err
}

func (b *Encoder) Encode(v interface{}) error {
	return b.encodeValue(reflect.Indirect(reflect.ValueOf(v)))
}

func (b *Encoder) encodeValue(rv reflect.Value) (err error) {
	t := rv.Type()
	switch t.Kind() {
	case reflect.Array:
		l := t.Len()
		for i := 0; i < l; i++ {
			v := reflect.Indirect(rv.Index(i).Addr())
			if err = b.encodeValue(v); err != nil {
				return
			}
		}

	case reflect.Slice:
		l := rv.Len()
		if err = b.writeUint64(uint64(l)); err != nil {
			return
		}

		// Fast-path for []byte
		if t.Elem().Kind() == reflect.Uint8 {
			_, err = b.w.Write(rv.Bytes())
			return
		}

		for i := 0; i < l; i++ {
			v := reflect.Indirect(rv.Index(i).Addr())
			if err = b.encodeValue(v); err != nil {
				return
			}
		}

	case reflect.Ptr:
		hasValue := !rv.IsNil()
		if err = b.writeFlag(hasValue); err != nil {
			return
		}

		if hasValue {
			v := reflect.Indirect(rv)
			if err = b.encodeValue(v); err != nil {
				return
			}
		}

	case reflect.Struct:
		meta := getMetadata(t, rv)

		// Call the marshaler
		if m := meta.GetMarshalBinary(rv); m != nil {
			ret := m.Call([]reflect.Value{})
			if !ret[1].IsNil() {
				err = ret[1].Interface().(error)
				return
			}

			// Write the marshaled byte slice
			buffer := ret[0].Bytes()
			if err = b.writeUint64(uint64(len(buffer))); err == nil {
				_, err = b.w.Write(buffer)
			}
			return
		}

		for _, i := range meta.fields {
			if err = b.encodeValue(rv.Field(i)); err != nil {
				return
			}
		}

		if b.strict && len(meta.fields) == 0 {
			return fmt.Errorf("binary: struct had no encodable fields")
		}

	case reflect.Map:
		l := rv.Len()
		if err = b.writeUint64(uint64(l)); err != nil {
			return
		}
		for _, key := range rv.MapKeys() {
			value := rv.MapIndex(key)
			if err = b.encodeValue(key); err != nil {
				return err
			}
			if err = b.encodeValue(value); err != nil {
				return err
			}
		}

	case reflect.String:
		if err = b.writeUint64(uint64(rv.Len())); err == nil {
			str := rv.String()
			_, err = b.w.Write(stringToByteSlice(&str))
		}

	case reflect.Bool:
		var out byte
		if rv.Bool() {
			out = 1
		}
		err = binary.Write(b.w, b.Order, out)

	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int:
		fallthrough
	case reflect.Int64:
		err = b.writeInt64(rv.Int())

	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint:
		fallthrough
	case reflect.Uint64:
		err = b.writeUint64(rv.Uint())

	case reflect.Complex64:
		err = binary.Write(b.w, b.Order, complex64(rv.Complex()))

	case reflect.Complex128:
		err = binary.Write(b.w, b.Order, rv.Complex())

	case reflect.Float32:
		err = binary.Write(b.w, b.Order, float32(rv.Float()))

	case reflect.Float64:
		err = binary.Write(b.w, b.Order, rv.Float())

	default:
		return errors.New("binary: unsupported type " + t.String())
	}

	return
}

// Reader represents the interface a reader should implement
type Reader interface {
	io.Reader
	io.ByteReader
}

type Decoder struct {
	Order binary.ByteOrder
	r     Reader
}

func NewDecoder(r Reader) *Decoder {
	return &Decoder{
		Order: DefaultEndian,
		r:     r,
	}
}

func (d *Decoder) Decode(v interface{}) error {
	rv := reflect.Indirect(reflect.ValueOf(v))
	if !rv.CanAddr() {
		return errors.New("binary: can only Decode to pointer type")
	}

	return d.decodeValue(rv)
}

func (d *Decoder) decodeValue(rv reflect.Value) (err error) {
	t := rv.Type()

	switch t.Kind() {
	case reflect.Array:
		len := t.Len()
		for i := 0; i < int(len); i++ {
			if err = d.decodeValue(reflect.Indirect(rv.Index(i).Addr())); err != nil {
				return err
			}
		}

	case reflect.Slice:
		var l uint64
		if l, err = binary.ReadUvarint(d.r); err != nil {
			return
		}

		// Fast-path for []byte
		if t.Elem().Kind() == reflect.Uint8 {
			buffer := make([]byte, int(l))
			if _, err = d.r.Read(buffer); err == nil {
				rv.Set(reflect.ValueOf(buffer))
			}
			return
		}

		if t.Kind() == reflect.Slice {
			rv.Set(reflect.MakeSlice(t, int(l), int(l)))
		} else if int(l) != t.Len() {
			return fmt.Errorf("binary: encoded size %d != real size %d", l, t.Len())
		}

		for i := 0; i < int(l); i++ {
			if err = d.decodeValue(reflect.Indirect(rv.Index(i).Addr())); err != nil {
				return err
			}
		}

	case reflect.Ptr:
		var hasValue byte
		if hasValue, err = d.r.ReadByte(); err != nil {
			return
		}

		if hasValue == 1 {
			if err = d.decodeValue(rv); err != nil {
				return err
			}
		}

	case reflect.Struct:
		meta := getMetadata(t, rv)

		// Call the unmarshaler
		if m := meta.GetUnmarshalBinary(rv); m != nil {
			var l uint64
			if l, err = binary.ReadUvarint(d.r); err != nil {
				return
			}

			buffer := make([]byte, l)
			_, err = d.r.Read(buffer)

			ret := m.Call([]reflect.Value{reflect.ValueOf(buffer)})
			if !ret[0].IsNil() {
				err = ret[0].Interface().(error)
			}

			return
		}

		for _, i := range meta.fields {
			if v := rv.Field(i); v.CanSet() {
				if err = d.decodeValue(reflect.Indirect(v.Addr())); err != nil {
					return
				}
			}
		}

	case reflect.Map:
		var l uint64
		if l, err = binary.ReadUvarint(d.r); err != nil {
			return
		}
		kt := t.Key()
		vt := t.Elem()
		rv.Set(reflect.MakeMap(t))
		for i := 0; i < int(l); i++ {
			kv := reflect.Indirect(reflect.New(kt))
			if err = d.decodeValue(kv); err != nil {
				return
			}

			vv := reflect.Indirect(reflect.New(vt))
			if err = d.decodeValue(vv); err != nil {
				return
			}

			rv.SetMapIndex(kv, vv)
		}

	case reflect.String:
		var l uint64
		if l, err = binary.ReadUvarint(d.r); err != nil {
			return
		}

		buf := make([]byte, l)
		_, err = d.r.Read(buf)
		rv.SetString(byteSliceToString(buf))

	case reflect.Bool:
		var out byte
		err = binary.Read(d.r, d.Order, &out)
		rv.SetBool(out != 0)

	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int:
		fallthrough
	case reflect.Int64:
		var v int64
		if v, err = binary.ReadVarint(d.r); err != nil {
			return
		}
		rv.SetInt(v)

	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint:
		fallthrough
	case reflect.Uint64:
		var v uint64
		if v, err = binary.ReadUvarint(d.r); err != nil {
			return
		}
		rv.SetUint(v)

	case reflect.Complex64:
		var out complex64
		err = binary.Read(d.r, d.Order, &out)
		rv.SetComplex(complex128(out))

	case reflect.Complex128:
		var out complex128
		err = binary.Read(d.r, d.Order, &out)
		rv.SetComplex(out)

	case reflect.Float32:
		var out float32
		err = binary.Read(d.r, d.Order, &out)
		rv.SetFloat(float64(out))

	case reflect.Float64:
		var out float64
		err = binary.Read(d.r, d.Order, &out)
		rv.SetFloat(out)

	default:
		return errors.New("binary: unsupported type " + t.String())
	}
	return
}

// ByteSliceToString is used when you really want to convert a slice
// of bytes to a string without incurring overhead. It is only safe
// to use if you really know the byte slice is not going to change
// in the lifetime of the string
func byteSliceToString(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

// StringToByteSlice converts a string pointer to a byte slice.
func stringToByteSlice(str *string) []byte {
	strHeader := (*reflect.StringHeader)(unsafe.Pointer(str))

	var b []byte
	byteHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	byteHeader.Data = strHeader.Data

	// need to take the length of s here to ensure s is live until after we update b's Data
	// field since the garbage collector can collect a variable once it is no longer used
	// not when it goes out of scope, for more details see https://github.com/golang/go/issues/9046
	l := len(*str)
	byteHeader.Len = l
	byteHeader.Cap = l
	return b
}
