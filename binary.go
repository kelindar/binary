package binary

import (
	"bytes"
	"encoding/binary"
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

// Marshal encodes the payload into binary format.
func Marshal(v interface{}) ([]byte, error) {
	b := &bytes.Buffer{}
	encoder := GetEncoder(b)
	defer encoder.Release()

	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Unmarshal decodes the payload from the binary format.
func Unmarshal(b []byte, v interface{}) error {
	return NewDecoder(bytes.NewReader(b)).Decode(v)
}

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

func getMetadata(rv reflect.Value) (meta *typeMeta) {
	t := rv.Type()
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
