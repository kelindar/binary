package binary

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// Reader represents the interface a reader should implement
type Reader interface {
	io.Reader
	io.ByteReader
}

// Decoder represents a binary decoder.
type Decoder struct {
	Order binary.ByteOrder
	r     Reader
}

// NewDecoder creates a binary decoder.
func NewDecoder(r Reader) *Decoder {
	return &Decoder{
		Order: DefaultEndian,
		r:     r,
	}
}

// Decode decodes a value by reading from the underlying io.Reader.
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
