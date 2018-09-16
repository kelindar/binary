// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	"testing"

	"github.com/kelindar/binary"
)

// 1000000	      1133 ns/op	     336 B/op	       9 allocs/op
// 1000000	      1110 ns/op	     312 B/op	       7 allocs/op
// 1000000	      1000 ns/op	     176 B/op	       3 allocs/op

func BenchmarkSortedSlice(b *testing.B) {
	v := Int64s{4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6}
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
		var out Int64s
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}
