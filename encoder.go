package binary

import (
	"encoding/binary"
	"errors"
	"io"
	"reflect"
	"sync"
)

// Encoder represents a binary encoder.
type Encoder struct {
	Order binary.ByteOrder
	out   io.Writer
	buf   []byte
	n     int
	Error error
}

// NewEncoder creates a new encoder.
func NewEncoder(out io.Writer) *Encoder {
	return &Encoder{
		Order: DefaultEndian,
		out:   out,
		buf:   make([]byte, 1024),
		n:     0,
		Error: nil,
	}
}

// GetEncoder borrows a pooled encoder.
func GetEncoder(out io.Writer) *Encoder {
	s := encoders.Get().(*Encoder)
	s.Reset(out)
	return s
}

// Encode encodes the value to the binary format.
func (e *Encoder) Encode(v interface{}) (err error) {
	if err = e.encodeValue(reflect.Indirect(reflect.ValueOf(v))); err == nil {
		return e.Flush()
	}
	return
}

func (e *Encoder) encodeValue(rv reflect.Value) (err error) {
	switch rv.Kind() {
	case reflect.Array:
		l := rv.Type().Len()
		for i := 0; i < l; i++ {
			v := reflect.Indirect(rv.Index(i).Addr())
			if err = e.encodeValue(v); err != nil {
				return
			}
		}

	case reflect.Slice:
		l := rv.Len()
		if err = e.writeUint64(uint64(l)); err != nil {
			return
		}

		// Fast-path for []byte
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			_, err = e.Write(rv.Bytes())
			return
		}

		for i := 0; i < l; i++ {
			v := reflect.Indirect(rv.Index(i).Addr())
			if err = e.encodeValue(v); err != nil {
				return
			}
		}

	case reflect.Ptr:
		hasValue := !rv.IsNil()
		if err = e.writeBool(hasValue); err != nil {
			return
		}

		if hasValue {
			v := reflect.Indirect(rv)
			if err = e.encodeValue(v); err != nil {
				return
			}
		}

	case reflect.Struct:
		meta := getMetadata(rv)

		// Call the marshaler
		if m := meta.GetMarshalBinary(rv); m != nil {
			ret := m.Call([]reflect.Value{})
			if !ret[1].IsNil() {
				err = ret[1].Interface().(error)
				return
			}

			// Write the marshaled byte slice
			buffer := ret[0].Bytes()
			if err = e.writeUint64(uint64(len(buffer))); err == nil {
				_, err = e.Write(buffer)
			}
			return
		}

		for _, i := range meta.fields {
			if err = e.encodeValue(rv.Field(i)); err != nil {
				return
			}
		}

	case reflect.Map:
		l := rv.Len()
		if err = e.writeUint64(uint64(l)); err != nil {
			return
		}
		for _, key := range rv.MapKeys() {
			value := rv.MapIndex(key)
			if err = e.encodeValue(key); err != nil {
				return err
			}
			if err = e.encodeValue(value); err != nil {
				return err
			}
		}

	case reflect.String:
		e.writeString(rv.String())

	case reflect.Bool:
		e.writeBool(rv.Bool())

	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int:
		fallthrough
	case reflect.Int64:
		err = e.writeInt64(rv.Int())

	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint:
		fallthrough
	case reflect.Uint64:
		err = e.writeUint64(rv.Uint())

	case reflect.Complex64:
		err = binary.Write(e, e.Order, complex64(rv.Complex()))

	case reflect.Complex128:
		err = binary.Write(e, e.Order, rv.Complex())

	case reflect.Float32:
		err = binary.Write(e, e.Order, float32(rv.Float()))

	case reflect.Float64:
		err = binary.Write(e, e.Order, rv.Float())

	default:
		return errors.New("binary: unsupported type " + rv.Type().String())
	}

	return
}

//-----------------------------------------------------------------------

// Reusable long-lived stream pool.
var encoders = &sync.Pool{New: func() interface{} {
	return &Encoder{
		Order: DefaultEndian,
		buf:   make([]byte, 1024),
		n:     0,
		Error: nil,
	}
}}

// Release releases the stream to the pool
func (e *Encoder) Release() {
	encoders.Put(e)
}

// Reset reuse this stream instance by assign a new writer
func (e *Encoder) Reset(out io.Writer) {
	e.out = out
	e.n = 0
	e.Error = nil
}

// Available returns how many bytes are unused in the buffer.
func (e *Encoder) Available() int {
	return len(e.buf) - e.n
}

// Buffered returns the number of bytes that have been written into the current buffer.
func (e *Encoder) Buffered() int {
	return e.n
}

// Buffer if writer is nil, use this method to take the result
func (e *Encoder) Buffer() []byte {
	return e.buf[:e.n]
}

// Write writes the contents of p into the buffer.
// It returns the number of bytes written.
// If nn < len(p), it also returns an error explaining
// why the write is short.
func (e *Encoder) Write(p []byte) (nn int, err error) {
	for len(p) > e.Available() && e.Error == nil {
		if e.out == nil {
			e.growAtLeast(len(p))
		} else {
			var n int
			if e.Buffered() == 0 {
				// Large write, empty buffer.
				// Write directly from p to avoid copy.
				n, e.Error = e.out.Write(p)
			} else {
				n = copy(e.buf[e.n:], p)
				e.n += n
				e.Flush()
			}
			nn += n
			p = p[n:]
		}
	}
	if e.Error != nil {
		return nn, e.Error
	}
	n := copy(e.buf[e.n:], p)
	e.n += n
	nn += n
	return nn, nil
}

// WriteByte writes a single byte.
func (e *Encoder) writeByte(c byte) {
	if e.Error != nil {
		return
	}
	if e.Available() < 1 {
		e.growAtLeast(1)
	}
	e.buf[e.n] = c
	e.n++
}

func (e *Encoder) writeTwoBytes(c1 byte, c2 byte) {
	if e.Error != nil {
		return
	}
	if e.Available() < 2 {
		e.growAtLeast(2)
	}
	e.buf[e.n] = c1
	e.buf[e.n+1] = c2
	e.n += 2
}

func (e *Encoder) writeThreeBytes(c1 byte, c2 byte, c3 byte) {
	if e.Error != nil {
		return
	}
	if e.Available() < 3 {
		e.growAtLeast(3)
	}
	e.buf[e.n] = c1
	e.buf[e.n+1] = c2
	e.buf[e.n+2] = c3
	e.n += 3
}

func (e *Encoder) writeInt64(v int64) error {
	if e.Error != nil {
		return e.Error
	}
	if e.Available() < 10 {
		e.growAtLeast(10)
	}

	e.n += binary.PutVarint(e.buf[e.n:], v)
	return nil
}

func (e *Encoder) writeUint64(v uint64) error {
	if e.Error != nil {
		return e.Error
	}
	if e.Available() < 10 {
		e.growAtLeast(10)
	}

	e.n += binary.PutUvarint(e.buf[e.n:], v)
	return nil
}

func (e *Encoder) writeBool(v bool) error {
	if e.Error != nil {
		return e.Error
	}
	if e.Available() < 1 {
		e.growAtLeast(1)
	}

	var b byte
	if v {
		b = 1
	}
	e.buf[e.n] = b
	e.n++
	return nil
}

func (e *Encoder) writeString(v string) {
	e.writeUint64(uint64(len(v)))
	e.Write(stringToByteSlice(&v))
}

// Flush writes any buffered data to the underlying io.Writer.
func (e *Encoder) Flush() error {
	if e.out == nil {
		return nil
	}
	if e.Error != nil {
		return e.Error
	}
	if e.n == 0 {
		return nil
	}
	n, err := e.out.Write(e.buf[0:e.n])
	if n < e.n && err == nil {
		err = io.ErrShortWrite
	}
	if err != nil {
		if n > 0 && n < e.n {
			copy(e.buf[0:e.n-n], e.buf[n:e.n])
		}
		e.n -= n
		e.Error = err
		return err
	}
	e.n = 0
	return nil
}

func (e *Encoder) ensure(minimal int) {
	available := e.Available()
	if available < minimal {
		e.growAtLeast(minimal)
	}
}

func (e *Encoder) growAtLeast(minimal int) {
	if e.out != nil {
		e.Flush()
		if e.Available() >= minimal {
			return
		}
	}
	toGrow := len(e.buf)
	if toGrow < minimal {
		toGrow = minimal
	}
	newBuf := make([]byte, len(e.buf)+toGrow)
	copy(newBuf, e.Buffer())
	e.buf = newBuf
}
