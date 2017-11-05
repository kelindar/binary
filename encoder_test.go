package binary

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkEncodeOne(b *testing.B) {
	codec := NewEncoder(new(bytes.Buffer))
	v := &msg{
		Name:      "Roman",
		Timestamp: 1242345235,
		Payload:   []byte("hi"),
		Ssid:      []uint32{1, 2, 3},
	}

	codec.Encode(v)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		codec.Encode(v)
	}
}

func TestBinaryEncodeStruct(t *testing.T) {
	b, err := Marshal(s0v)
	assert.NoError(t, err)
	assert.Equal(t, s0b, b)
}
