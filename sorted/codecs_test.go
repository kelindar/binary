// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	"testing"
	"time"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
)

/*
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkSortedSlice/encode-int-12         	 1875320	       614.1 ns/op	     216 B/op	       4 allocs/op
BenchmarkSortedSlice/decode-int-12         	  544180	      2187 ns/op	    1080 B/op	      36 allocs/op
BenchmarkSortedSlice/encode-uint-12        	 2051314	       581.3 ns/op	     216 B/op	       4 allocs/op
BenchmarkSortedSlice/decode-uint-12        	  571098	      2170 ns/op	    1080 B/op	      36 allocs/op
*/
func BenchmarkSortedSlice(b *testing.B) {
	ints := Int32s{4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6}
	intsEnc, _ := binary.Marshal(&ints)

	b.Run("encode-int", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&ints)
		}
	})

	b.Run("decode-int", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out Int64s
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(intsEnc, &out)
		}
	})

	uints := Uint32s{4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6}
	uintsEnc, _ := binary.Marshal(&uints)

	b.Run("encode-uint", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&uints)
		}
	})

	b.Run("decode-uint", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out Uint64s
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(uintsEnc, &out)
		}
	})
}

/*
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkTimes/encode-12         	   17698	     76181 ns/op	   30893 B/op	       6 allocs/op
BenchmarkTimes/decode-12         	   22126	     54699 ns/op	   81977 B/op	       2 allocs/op
*/
func BenchmarkTimes(b *testing.B) {
	var times Timestamps
	for i := uint64(0); i < 10000; i++ {
		times = append(times, uint64(time.Now().Unix())+i)
	}
	enc, _ := binary.Marshal(&times)

	b.Run("encode", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&times)
		}
	})

	b.Run("decode", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out Timestamps
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})

}

func TestPayload(t *testing.T) {
	encoded := []byte{0x8, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2}

	v := Int32s{1, 2, 3, 4, 5, 6, 7, 8}
	ev, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.Equal(t, encoded, ev)
}
