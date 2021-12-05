// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"bytes"
	"io"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReader_Slice(t *testing.T) {
	r := newSliceReader([]byte("0123456789"))

	out, err := r.Slice(3)
	assert.NoError(t, err)
	assert.Len(t, out, 3)
	assert.Equal(t, "012", string(out))
	assert.Equal(t, 7, r.Len())
}

func TestReaderEOF(t *testing.T) {
	b, _ := Marshal(newBigStruct())

	for size := 0; size < len(b)-1; size++ {
		var output bigStruct
		assert.Error(t, Unmarshal(b[0:size], &output))
	}
}

func TestStreamReader(t *testing.T) {
	input := newBigStruct()
	b, _ := Marshal(input)

	dec := NewDecoder(newNetworkSource(b))
	out := new(bigStruct)
	assert.NoError(t, dec.Decode(out))
}

// --------------------------------------- Big Structure (Every Field Type) ---------------------------------------

// structure with every possible codec type
type bigStruct struct {
	String    string
	Uint8     uint8
	Uint16    uint16
	Uint32    uint32
	Uint64    uint64
	Int8      int8
	Int16     int16
	Int32     int32
	Int64     int64
	Float32   float32
	Float64   float64
	Strings   []string
	Bytes     []byte
	Bools     []bool
	Uint8s    []uint8
	Uint16s   []uint16
	Uint32s   []uint32
	Uint64s   []uint64
	Int8s     []int8
	Int16s    []int16
	Int32s    []int32
	Int64s    []int64
	Float32s  []float32
	Float64s  []float64
	MapStr    map[string]simpleStruct
	MapPtr    map[string]*simpleStruct
	MapUint8  map[uint8]uint8
	MapUint16 map[uint16]uint16
	MapUint32 map[uint32]uint32
	MapUint64 map[uint64]uint64
	MapInt8   map[int8]int8
	MapInt16  map[int16]int16
	MapInt32  map[int32]int32
	MapInt64  map[int64]int64
	Time      *time.Time
	Nil       *time.Time
	Pointer   *simpleStruct
	Value     simpleStruct
	Array     [6]byte
	Byte      byte
	Bool      bool
}

func newBigStruct() *bigStruct {
	timestamp := time.Date(2013, 1, 2, 3, 4, 5, 6, time.UTC)
	child := simpleStruct{
		Name:      "Roman",
		Timestamp: timestamp,
		Payload:   []byte("hi"),
		Ssid:      []uint32{1, 2, 3},
	}

	return &bigStruct{
		String:    "hello",
		Byte:      0x3a,
		Bool:      true,
		Uint8:     math.MaxUint8,
		Uint16:    math.MaxUint16,
		Uint32:    math.MaxUint32,
		Uint64:    math.MaxUint64,
		Int8:      math.MaxInt8,
		Int16:     math.MaxInt16,
		Int32:     math.MaxInt32,
		Int64:     math.MaxInt64,
		Float32:   math.MaxFloat32,
		Float64:   math.MaxFloat64,
		Strings:   []string{"a", "b", "c"},
		Bytes:     []byte("hello-bytes"),
		Bools:     []bool{true, false, true},
		Uint8s:    []uint8{0, math.MaxUint8},
		Uint16s:   []uint16{0, math.MaxUint16},
		Uint32s:   []uint32{0, math.MaxUint32},
		Uint64s:   []uint64{0, math.MaxUint64},
		Int8s:     []int8{math.MinInt8, math.MaxInt8},
		Int16s:    []int16{math.MinInt16, math.MaxInt16},
		Int32s:    []int32{math.MinInt32, math.MaxInt32},
		Int64s:    []int64{math.MinInt64, math.MaxInt64},
		Float32s:  []float32{0, math.MaxFloat32},
		Float64s:  []float64{0, math.MaxFloat64},
		MapStr:    map[string]simpleStruct{"a": child, "b": child},
		MapPtr:    map[string]*simpleStruct{"a": &child, "b": nil},
		MapUint8:  map[uint8]uint8{1: 1},
		MapUint16: map[uint16]uint16{1: 1},
		MapUint32: map[uint32]uint32{1: 1},
		MapUint64: map[uint64]uint64{1: 1},
		MapInt8:   map[int8]int8{1: 1},
		MapInt16:  map[int16]int16{1: 1},
		MapInt32:  map[int32]int32{1: 1},
		MapInt64:  map[int64]int64{1: 1},
		Array:     [6]byte{1, 2, 3, 4, 5, 6},
		Time:      &timestamp,
		Nil:       nil,
		Pointer:   &child,
		Value:     child,
	}
}

// --------------------------------------- Fake Network Reader ---------------------------------------

type networkSource struct {
	r io.Reader
}

func newNetworkSource(data []byte) io.Reader {
	return &networkSource{
		r: bytes.NewBuffer(data),
	}
}

func (s *networkSource) Read(p []byte) (n int, err error) {
	return s.r.Read(p)
}
