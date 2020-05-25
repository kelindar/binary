// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package nocopy

import (
	"testing"

	"github.com/kelindar/binary"
)

const testString = "Donec egestas enim vitae turpis imperdiet ultricies. Vivamus sollicitudin in felis quis euismod. Nunc at tellus lectus."
const defaultSize = 10000

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

// BenchmarkString_Safe/marshal-8         	 5000000	       368 ns/op	     368 B/op	       3 allocs/op
// BenchmarkString_Safe/unmarshal-8       	 5000000	       227 ns/op	     160 B/op	       2 allocs/op
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

// BenchmarkDictionary_Unsafe/marshal-8         	 4166210	       275 ns/op	     112 B/op	       2 allocs/op
// BenchmarkDictionary_Unsafe/unmarshal-8       	20000932	        59.4 ns/op	       0 B/op	       0 allocs/op
func BenchmarkDictionary_Unsafe(b *testing.B) {
	v := Dictionary{
		"name":   "Roman",
		"race":   "human",
		"status": "happy",
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
		var out String
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}

// BenchmarkByteMap_Unsafe/marshal-8         	 2240605	       539 ns/op	    1008 B/op	       4 allocs/op
// BenchmarkByteMap_Unsafe/unmarshal-8       	19098699	        62.1 ns/op	       0 B/op	       0 allocs/op
func BenchmarkByteMap_Unsafe(b *testing.B) {
	v := ByteMap{
		"name":   []byte(testString),
		"race":   []byte(testString),
		"status": []byte(testString),
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
		var out String
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}

// BenchmarkString_Unsafe/marshal-8         	 5000000	       348 ns/op	     368 B/op	       3 allocs/op
// BenchmarkString_Unsafe/unmarshal-8       	20000000	       108 ns/op	       0 B/op	       0 allocs/op
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

// BenchmarkBytes_Safe/marshal-8         	 1000000	      1616 ns/op	   10354 B/op	       3 allocs/op
// BenchmarkBytes_Safe/unmarshal-8       	 1000000	      1547 ns/op	   10274 B/op	       2 allocs/op
func BenchmarkBytes_Safe(b *testing.B) {
	v := makeBytes(defaultSize)
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

// BenchmarkBytes_Unsafe/marshal-8         	 1000000	      1676 ns/op	   10354 B/op	       3 allocs/op
// BenchmarkBytes_Unsafe/unmarshal-8       	20000000	       116 ns/op	       0 B/op	       0 allocs/op
func BenchmarkBytes_Unsafe(b *testing.B) {
	v := Bytes(makeBytes(defaultSize))
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

// BenchmarkUint64s_Safe/marshal-8         	   10000	    167612 ns/op	   78325 B/op	      11 allocs/op
// BenchmarkUint64s_Safe/unmarshal-8       	    5000	    286769 ns/op	   81975 B/op	       2 allocs/op
func BenchmarkUint64s_Safe(b *testing.B) {
	v := makeUint64s(defaultSize)
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

// BenchmarkUint64s_Unsafe/marshal-8         	  100000	     10711 ns/op	   82055 B/op	       3 allocs/op
// BenchmarkUint64s_Unsafe/unmarshal-8       	20000000	       109 ns/op	       0 B/op	       0 allocs/op
func BenchmarkUint64s_Unsafe(b *testing.B) {
	v := Uint64s(makeUint64s(defaultSize))
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

// BenchmarkColumnar_Unsafe/marshal-8         	 1000000	      1712 ns/op	    1450 B/op	      10 allocs/op
// BenchmarkColumnar_Unsafe/unmarshal-8       	 1000000	      1519 ns/op	    1056 B/op	      10 allocs/op
func BenchmarkColumnar_Unsafe(b *testing.B) {
	v := composite{}
	v["a"] = column{
		Varchar: columnVarchar{
			Nulls: Bools{false, false, false, true, false, false, false, false, true, false, false, false, false, true, false},
			Sizes: Uint32s{2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2},
			Bytes: Bytes{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
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

type message struct {
	A Bytes
	B Bytes
	C Bytes
	D Bytes
}

// BenchmarkStruct_Unsafe/marshal-8         	 3000000	       459 ns/op	     272 B/op	       3 allocs/op
// BenchmarkStruct_Unsafe/unmarshal-8       	10000000	       221 ns/op	       0 B/op	       0 allocs/op
func BenchmarkStruct_Unsafe(b *testing.B) {
	v := message{
		A: Bytes{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
		B: Bytes{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
		C: Bytes{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
		D: Bytes{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
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
		var out message
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}
