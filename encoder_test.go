// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"bytes"
	"io"
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

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

func TestEncoder(t *testing.T) {
	tests := map[string]func(*testing.T){
		"composite":         testEncoderComposite,
		"struct bytes":      testEncoderStructBytes,
		"sizeof":            testEncoderSizeOf,
		"alternating types": testEncoderAlternatingTypes,
		"custom codec":      testEncoderCustomCodec,
	}
	for name, fn := range tests {
		t.Run(name, fn)
	}
}

func testEncoderComposite(t *testing.T) {
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

func testEncoderStructBytes(t *testing.T) {
	b, err := Marshal(s0v)
	assert.NoError(t, err)
	assert.Equal(t, s0b, b)
}

func testEncoderSizeOf(t *testing.T) {
	var e Encoder
	assert.Equal(t, 80, int(unsafe.Sizeof(e)))
	assert.Equal(t, 24, int(unsafe.Sizeof(fieldCodec{})))
}

func testEncoderAlternatingTypes(t *testing.T) {
	type first struct{ Value uint64 }
	type second struct{ Value string }

	values := []any{
		&first{Value: 1},
		&second{Value: "two"},
		&first{Value: 3},
	}

	var buffer bytes.Buffer
	encoder := NewEncoder(&buffer)
	for _, value := range values {
		assert.NoError(t, encoder.Encode(value))
	}

	decoder := NewDecoder(&buffer)
	for _, want := range values {
		got := reflect.New(reflect.TypeOf(want).Elem()).Interface()
		assert.NoError(t, decoder.Decode(got))
		assert.Equal(t, want, got)
	}
}

func testEncoderCustomCodec(t *testing.T) {
	v := testCustom("custom codec")

	b, err := Marshal(v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var out testCustom
	err = Unmarshal(b, &out)
	assert.NoError(t, err)
	assert.Equal(t, v, out)
}

func TestEncoderErrorPaths(t *testing.T) {
	e := NewEncoder(nil)
	assert.Error(t, e.err)
	var nilBuffer *bytes.Buffer
	e.Reset(nilBuffer)
	assert.Error(t, e.err)
	var nilWriter *errorWriter
	e.Reset(nilWriter)
	assert.Error(t, e.err)

	e.Reset(io.Discard)
	e.err = io.ErrClosedPipe
	e.Write([]byte("ignored"))
	assert.Equal(t, io.ErrClosedPipe, e.err)

	assert.Equal(t, "binary: cannot encode nil value", NewEncoder(io.Discard).Encode(nil).Error())
	assert.Error(t, MarshalTo([]complex64{1 + 2i}, errorWriter{}))
	assert.Error(t, MarshalTo([]complex128{1 + 2i}, errorWriter{}))

	large := reflect.New(reflect.ArrayOf(1<<20+1, reflect.TypeFor[byte]())).Elem()
	assert.Equal(t, 64, marshalCapacity(large))
}
