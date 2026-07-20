// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

type composite map[string]column

type column struct {
	Varchar columnVarchar
	Float64 columnFloat64
	Float32 columnFloat32
}

type columnVarchar struct {
	Nulls []bool
	Sizes []uint32
	Bytes []byte
}

type columnFloat64 struct {
	Nulls  []bool
	Floats []float64
}

type columnFloat32 struct {
	Nulls  []bool
	Floats []float32
}

func Test_Full(t *testing.T) {
	v := composite{}
	v["a"] = column{
		Varchar: columnVarchar{
			Nulls: []bool{false, false, false, true, false},
			Sizes: []uint32{2, 2, 2, 0, 2},
			Bytes: []byte{10, 10, 10, 10, 10, 10, 10, 10},
		},
	}
	v["b"] = column{
		Float64: columnFloat64{
			Nulls:  []bool{false, false, false, true, false},
			Floats: []float64{1.1, 2.2, 3.3, 0, 4.4},
		},
	}

	b, err := Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o composite
	err = Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func newComposite() composite {
	v := composite{}
	v["a"] = column{
		Varchar: columnVarchar{
			Nulls: []bool{false, false, false, true, false, false, false, false, true, false, false, false, false, true, false},
			Sizes: []uint32{2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2},
			Bytes: []byte{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
		},
	}
	v["b"] = column{
		Float64: columnFloat64{
			Nulls:  []bool{false, false, false, true, false},
			Floats: []float64{1.1, 2.2, 3.3, 0, 4.4},
		},
	}
	return v
}

func TestEncodeStruct(t *testing.T) {
	b, err := Marshal(s0v)
	assert.NoError(t, err)
	assert.Equal(t, s0b, b)
}

func TestEncoderSizeOf(t *testing.T) {
	var e Encoder
	assert.Equal(t, 56, int(unsafe.Sizeof(e)))
}

func TestCustomCodec(t *testing.T) {
	v := testCustom("custom codec")

	b, err := Marshal(v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var out testCustom
	err = Unmarshal(b, &out)
	assert.NoError(t, err)
	assert.Equal(t, v, out)
}
