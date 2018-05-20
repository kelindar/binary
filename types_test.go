// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 1000000	      1133 ns/op	     336 B/op	       9 allocs/op
// 1000000	      1110 ns/op	     312 B/op	       7 allocs/op
func BenchmarkSortedSlice_Marshal(b *testing.B) {
	v := SortedInt64s{4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6}

	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		Marshal(&v)
	}
}

func Test_SortedSliceInt64(t *testing.T) {
	v := SortedInt64s{4, 5, 6, 1, 2, 3}

	b, err := Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o SortedInt64s
	err = Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_SortedSliceUint64(t *testing.T) {
	v := SortedUint64s{4, 5, 6, 1, 2, 3}

	b, err := Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o SortedUint64s
	err = Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_SortedSliceInt32(t *testing.T) {
	v := SortedInt32s{4, 5, 6, 1, 2, 3}

	b, err := Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o SortedInt32s
	err = Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_SortedSliceUint32(t *testing.T) {
	v := SortedUint32s{4, 5, 6, 1, 2, 3}

	b, err := Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o SortedUint32s
	err = Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}
