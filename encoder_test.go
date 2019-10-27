// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

var testMsg = msg{
	Name:      "Roman",
	Timestamp: 1242345235,
	Payload:   []byte("hi"),
	Ssid:      []uint32{1, 2, 3},
}

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

//type benchStruct = composite

type benchStruct = msg

func newBenchStruct() benchStruct {
	//return newComposite()
	return testMsg
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

// Benchmark_Binary/marshal-8         	 5286771	       226 ns/op	     112 B/op	       2 allocs/op
// Benchmark_Binary/marshal-to-8      	 6467770	       167 ns/op	      33 B/op	       0 allocs/op
// Benchmark_Binary/unmarshal-8       	 3350119	       355 ns/op	      88 B/op	       5 allocs/op
func Benchmark_Binary(b *testing.B) {
	v := newBenchStruct()
	enc, _ := Marshal(&v)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			Marshal(&v)
		}
	})

	var buffer bytes.Buffer
	b.Run("marshal-to", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			MarshalTo(&v, &buffer)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out benchStruct
		for n := 0; n < b.N; n++ {
			Unmarshal(enc, &out)
		}
	})
}

func Benchmark_Gob(b *testing.B) {
	v := newBenchStruct()

	buffer := new(bytes.Buffer)
	codec := gob.NewEncoder(buffer)
	codec.Encode(&v)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			gob.NewEncoder(new(bytes.Buffer)).Encode(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out benchStruct
		for n := 0; n < b.N; n++ {
			gob.NewDecoder(buffer).Decode(&out)
		}
	})
}

func Benchmark_JSON(b *testing.B) {
	v := newBenchStruct()
	enc, _ := json.Marshal(&v)

	b.Run("marshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			json.Marshal(&v)
		}
	})

	b.Run("unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out benchStruct
		for n := 0; n < b.N; n++ {
			json.Unmarshal(enc, &out)
		}
	})
}

func TestBinaryEncodeStruct(t *testing.T) {
	b, err := Marshal(s0v)
	assert.NoError(t, err)
	assert.Equal(t, s0b, b)
}

func TestEncoderSizeOf(t *testing.T) {
	var e Encoder
	assert.Equal(t, 56, int(unsafe.Sizeof(e)))
}

func TestMarshalWithCustomCodec(t *testing.T) {
	v := testCustom("custom codec")

	b, err := Marshal(v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var out testCustom
	err = Unmarshal(b, &out)
	assert.NoError(t, err)
	assert.Equal(t, v, out)
}
