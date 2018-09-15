// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package unsafe

import (
	"testing"

	"github.com/kelindar/binary"
)

var arr = []uint64{4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6}

func BenchmarkUint64s_Safe(b *testing.B) {
	v := arr
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
	v := Uint64s(arr)
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
