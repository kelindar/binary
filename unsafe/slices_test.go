// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package unsafe

import (
	"testing"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
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
	b.ReportAllocs()

	b.Run("marshal", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
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
	b.ReportAllocs()

	b.Run("marshal", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ResetTimer()
		var out Uint64s
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}

func Test_Bools(t *testing.T) {
	v := Bools{true, false, true, true, false, false}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Bools
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Uint16(t *testing.T) {
	v := Uint16s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Uint16s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Int16(t *testing.T) {
	v := Int16s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Int16s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Uint32(t *testing.T) {
	v := Uint32s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Uint32s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Int32(t *testing.T) {
	v := Int32s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Int32s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Uint64(t *testing.T) {
	v := Uint64s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Uint64s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Int64(t *testing.T) {
	v := Int64s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Int64s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Float32(t *testing.T) {
	v := Float32s{4.5, 5.01, 6.61, 1.12, 2.1, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Float32s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Float64(t *testing.T) {
	v := Float64s{4.5, 5.01, 6.61, 1.12, 2.1, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Float64s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}
