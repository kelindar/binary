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

/*
func TestPtrSlice(t *testing.T) {
	v1 := []int{1, 2, 3}
	v2 := []*int{&v1[0], &v1[1], &v1[2]}

	b, err := Marshal(v2)
	assert.NoError(t, err)
	//assert.Equal(t, s0b, b)

	var v3 []*int
	Unmarshal(b, &v3)

	assert.Equal(t, s0b, v3)
}
*/
