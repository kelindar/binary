package binary

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testMsg = &msg{
	Name:      "Roman",
	Timestamp: 1242345235,
	Payload:   []byte("hi"),
	Ssid:      []uint32{1, 2, 3},
}

func BenchmarkEncodeBinary(b *testing.B) {
	codec := NewEncoder(new(bytes.Buffer))

	codec.Encode(testMsg)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		codec.Encode(testMsg)
	}
}

func BenchmarkEncodeGob(b *testing.B) {
	codec := gob.NewEncoder(new(bytes.Buffer))

	codec.Encode(testMsg)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		codec.Encode(testMsg)
	}
}

func BenchmarkEncodeJson(b *testing.B) {
	codec := json.NewEncoder(new(bytes.Buffer))

	codec.Encode(testMsg)
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		codec.Encode(testMsg)
	}
}

func TestBinaryEncodeStruct(t *testing.T) {
	b, err := Marshal(s0v)
	assert.NoError(t, err)
	assert.Equal(t, s0b, b)
}
