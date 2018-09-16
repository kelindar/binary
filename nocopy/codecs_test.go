// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package nocopy

import (
	"testing"

	"github.com/kelindar/binary"
)

const testString = "Donec egestas enim vitae turpis imperdiet ultricies. Vivamus sollicitudin in felis quis euismod. Nunc at tellus lectus."

func makeUint64s(n int) (arr []uint64) {
	for i := 0; i < n; i++ {
		arr = append(arr, uint64(i))
	}
	return
}

func makeBytes(n int) (arr []byte) {
	for i := 0; i < n; i++ {
		arr = append(arr, byte(i%255))
	}
	return
}

func BenchmarkString_Safe(b *testing.B) {
	v := testString
	enc, _ := binary.Marshal(&v)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out []byte
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}

func BenchmarkString_Unsafe(b *testing.B) {
	v := String(testString)
	enc, _ := binary.Marshal(&v)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out String
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}

func asBytes(v []uint64) (o []byte) {
	for i := range v {
		o = append(o, byte(v[i]))
	}
	return
}

func BenchmarkBytes_Safe(b *testing.B) {
	v := makeBytes(500)
	enc, _ := binary.Marshal(&v)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out []byte
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}

func BenchmarkBytes_Unsafe(b *testing.B) {
	v := Bytes(makeBytes(500))
	enc, _ := binary.Marshal(&v)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out Bytes
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}

func BenchmarkUint64s_Safe(b *testing.B) {
	v := makeUint64s(500)
	enc, _ := binary.Marshal(&v)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out []uint64
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}

func BenchmarkUint64s_Unsafe(b *testing.B) {
	v := Uint64s(makeUint64s(500))
	enc, _ := binary.Marshal(&v)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out Uint64s
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}

func BenchmarkColumnar_Unsafe(b *testing.B) {
	v := composite{}
	v["a"] = column{
		Varchar: columnVarchar{
			Nulls: Bools{false, false, false, true, false},
			Sizes: Uint32s{2, 2, 2, 0, 2},
			Bytes: Bytes{10, 10, 10, 10, 10, 10, 10, 10},
		},
	}
	v["b"] = column{
		Float64: columnFloat64{
			Nulls:  Bools{false, false, false, true, false},
			Floats: Float64s{1.1, 2.2, 3.3, 0, 4.4},
		},
	}

	enc, _ := binary.Marshal(&v)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out composite
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}
