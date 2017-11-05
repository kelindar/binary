package binary

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkDecodeOne(b *testing.B) {
	rdr := bytes.NewReader([]byte{0x5, 0x52, 0x6f, 0x6d, 0x61, 0x6e, 0xa6, 0xbc, 0xe5, 0xa0, 0x9, 0x2, 0x68, 0x69, 0x3, 0x1, 0x2, 0x3})
	codec := NewDecoder(rdr)

	o := &msg{}

	codec.Decode(o)
	rdr.Seek(0, 0)

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		codec.Decode(o)
		rdr.Seek(0, 0)
	}
}

func TestBinaryDecodeStruct(t *testing.T) {
	s := &s0{}
	err := Unmarshal(s0b, s)
	assert.NoError(t, err)
	assert.Equal(t, s0v, s)
}

func TestBinaryDecodeToValueErrors(t *testing.T) {
	b := []byte{1, 0, 0, 0}
	var v uint32
	err := Unmarshal(b, v)
	assert.Error(t, err)
	err = Unmarshal(b, &v)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), v)
}
