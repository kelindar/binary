package binary

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
)

// Encoder represents a binary encoder.
type Encoder struct {
	buf   []byte
	Order binary.ByteOrder
	w     io.Writer
}

// NewEncoder creates a new encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		Order: DefaultEndian,
		w:     w,
		buf:   make([]byte, 10),
	}
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
