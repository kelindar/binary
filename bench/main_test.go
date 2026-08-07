package main

import (
	"testing"

	"github.com/kelindar/binary"
)

func TestHelpers(t *testing.T) {
	t.Run("uint64s", func(t *testing.T) {
		got := makeUint64s(5)
		if len(got) != 5 || got[4] != 4 {
			t.Fatalf("makeUint64s(5) = %v", got)
		}
	})
	t.Run("bytes", func(t *testing.T) {
		got := makeBytes(5)
		if len(got) != 5 || got[4] != 4 {
			t.Fatalf("makeBytes(5) = %v", got)
		}
	})
}

func BenchmarkBinaryStructs(b *testing.B) {
	v := msg{
		Name:      "Roman",
		Timestamp: 1242345235,
		Payload:   []byte("hello"),
		Ssid:      []uint32{1, 2, 3},
	}
	array := [100]msg{}
	for i := range array {
		array[i] = v
	}
	arrayData, _ := binary.Marshal(&array)
	var arrayOut [100]msg

	b.Run("struct-enc", func(b *testing.B) {
		for range b.N {
			_, _ = binary.Marshal(&v)
		}
	})
	b.Run("struct-dec", func(b *testing.B) {
		data, _ := binary.Marshal(&v)
		var out msg
		b.ResetTimer()
		for range b.N {
			_ = binary.Unmarshal(data, &out)
		}
	})
	b.Run("array-enc", func(b *testing.B) {
		for range b.N {
			_, _ = binary.Marshal(&array)
		}
	})
	b.Run("array-dec", func(b *testing.B) {
		for range b.N {
			_ = binary.Unmarshal(arrayData, &arrayOut)
		}
	})
}
