// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"bytes"
	"errors"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Message represents a message to be flushed
type msg struct {
	Name      string
	Timestamp int64
	Payload   []byte
	Ssid      []uint32
}

type s0 struct {
	A string
	B string
	C int16
}

var (
	s0v = &s0{"A", "B", 1}
	s0b = []byte{0x1, 0x41, 0x1, 0x42, 0x2}
)

func TestBinaryTime(t *testing.T) {
	input := []time.Time{
		time.Date(2013, 1, 2, 3, 4, 5, 6, time.UTC),
	}

	output := []byte{0x1, 0xf, 0x1, 0x0, 0x0, 0x0, 0xe, 0xc8, 0x75, 0x9a, 0xa5, 0x0, 0x0, 0x0, 0x6, 0xff, 0xff}

	b, err := Marshal(&input)
	assert.NoError(t, err)
	assert.Equal(t, output, b)

	var v []time.Time
	err = Unmarshal(b, &v)

	assert.NoError(t, err)
	assert.Equal(t, input, v)
	assert.Equal(t, 1, len(v))
}

// Message represents a message to be flushed
type simpleStruct struct {
	Name      string
	Timestamp time.Time
	Payload   []byte
	Ssid      []uint32
}

type sliceStruct struct {
	Payload []byte
}

func TestBinaryEncode_EOF(t *testing.T) {
	v := &sliceStruct{
		Payload: nil,
	}
	output := []byte{0x0}

	b, err := Marshal(v)
	assert.NoError(t, err)
	assert.Equal(t, output, b)

	s := &sliceStruct{}
	err = Unmarshal(b, s)
	assert.NoError(t, err)
	assert.Equal(t, v, s)
}

func TestBinaryEncodeSimpleStruct(t *testing.T) {
	v := &simpleStruct{
		Name:      "Roman",
		Timestamp: time.Date(2013, 1, 2, 3, 4, 5, 6, time.UTC),
		Payload:   []byte("hi"),
		Ssid:      []uint32{1, 2, 3},
	}
	output := []byte{0x5, 0x52, 0x6f, 0x6d, 0x61, 0x6e, 0xf, 0x1, 0x0, 0x0, 0x0, 0xe, 0xc8, 0x75, 0x9a, 0xa5, 0x0, 0x0, 0x0, 0x6, 0xff, 0xff, 0x2, 0x68, 0x69, 0x3, 0x1, 0x2, 0x3}

	b, err := Marshal(v)
	assert.NoError(t, err)
	assert.Equal(t, output, b)

	s := &simpleStruct{}
	err = Unmarshal(b, s)
	assert.NoError(t, err)
	assert.Equal(t, v, s)
}

func TestBinarySimpleStructSlice(t *testing.T) {
	input := []simpleStruct{{
		Name:      "Roman",
		Timestamp: time.Date(2013, 1, 2, 3, 4, 5, 6, time.UTC),
		Payload:   []byte("hi"),
		Ssid:      []uint32{1, 2, 3},
	}, {
		Name:      "Roman",
		Timestamp: time.Date(2013, 1, 2, 3, 4, 5, 6, time.UTC),
		Payload:   []byte("hi"),
		Ssid:      []uint32{1, 2, 3},
	}}

	b, err := Marshal(&input)

	var v []simpleStruct
	err = Unmarshal(b, &v)

	assert.NoError(t, err)
	assert.Equal(t, input, v)
	assert.Equal(t, 2, len(v))
}

type s1 struct {
	Name     string
	BirthDay time.Time
	Phone    string
	Siblings int
	Spouse   bool
	Money    float64
	Tags     map[string]string
	Aliases  []string
}

var (
	s1v = &s1{
		Name:     "Bob Smith",
		BirthDay: time.Date(2013, 1, 2, 3, 4, 5, 6, time.UTC),
		Phone:    "5551234567",
		Siblings: 2,
		Spouse:   false,
		Money:    100.0,
		Tags:     map[string]string{"key": "value"},
		Aliases:  []string{"Bobby", "Robert"},
	}

	svb = []byte{0x9, 0x42, 0x6f, 0x62, 0x20, 0x53, 0x6d, 0x69, 0x74, 0x68, 0xf, 0x1, 0x0, 0x0, 0x0, 0xe, 0xc8, 0x75, 0x9a, 0xa5, 0x0, 0x0, 0x0,
		0x6, 0xff, 0xff, 0xa, 0x35, 0x35, 0x35, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x59, 0x40, 0x1,
		0x3, 0x0, 0x6b, 0x65, 0x79, 0x5, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x2, 0x5, 0x42, 0x6f, 0x62, 0x62, 0x79, 0x6, 0x52, 0x6f, 0x62, 0x65, 0x72, 0x74}
)

func TestBinaryEncodeComplex(t *testing.T) {
	b, err := Marshal(s1v)
	assert.NoError(t, err)
	assert.Equal(t, svb, b)

	s := &s1{}
	err = Unmarshal(b, s)
	assert.NoError(t, err)
	assert.Equal(t, s1v, s)
}

type s2 struct {
	b []byte
}

func (s *s2) UnmarshalBinary(data []byte) error {
	if len(data) != 1 {
		return errors.New("expected data to be length 1")
	}
	s.b = data
	return nil
}

func (s *s2) MarshalBinary() (data []byte, err error) {
	return s.b, nil
}

func TestBinaryMarshalUnMarshaler(t *testing.T) {
	s2v := &s2{[]byte{0x13}}
	b, err := Marshal(s2v)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x1, 0x13}, b)
}

func TestMarshalUnMarshalTypeAliases(t *testing.T) {
	type Foo int64
	f := Foo(32)
	b, err := Marshal(f)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x40}, b)
}

func TestStructWithStruct(t *testing.T) {
	type T1 struct {
		ID    uint64
		Name  string
		Slice []int
	}
	type T2 uint64
	type Struct struct {
		V1 T1
		V2 T2
		V3 T1
	}

	s := Struct{V1: T1{1, "1", []int{1}}, V2: 2, V3: T1{3, "3", []int{3}}}
	buf := new(bytes.Buffer)
	enc := NewEncoder(buf)
	err := enc.Encode(&s)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	v := Struct{}
	dec := NewDecoder(buf)
	err = dec.Decode(&v)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	if !reflect.DeepEqual(s, v) {
		t.Fatalf("got= %#v\nwant=%#v\n", v, s)
	}

}

func TestStructWithEmbeddedStruct(t *testing.T) {
	type T1 struct {
		ID    uint64
		Name  string
		Slice []int
	}
	type T2 uint64
	type Struct struct {
		T1
		V2 T2
		V3 T1
	}

	s := Struct{T1: T1{1, "1", []int{1}}, V2: 2, V3: T1{3, "3", []int{3}}}
	buf := new(bytes.Buffer)
	enc := NewEncoder(buf)
	err := enc.Encode(&s)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	v := Struct{}
	dec := NewDecoder(buf)
	err = dec.Decode(&v)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	if !reflect.DeepEqual(s, v) {
		t.Fatalf("got= %#v\nwant=%#v\n", v, s)
	}

}

func TestArrayOfStructWithStruct(t *testing.T) {
	type T1 struct {
		ID    uint64
		Name  string
		Slice []int
	}
	type T2 uint64
	type Struct struct {
		V1 T1
		V2 T2
		V3 T1
	}

	s := [1]Struct{
		{V1: T1{1, "1", []int{1}}, V2: 2, V3: T1{3, "3", []int{3}}},
	}
	buf := new(bytes.Buffer)
	enc := NewEncoder(buf)
	err := enc.Encode(&s)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	v := [1]Struct{}
	dec := NewDecoder(buf)
	err = dec.Decode(&v)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	if !reflect.DeepEqual(s, v) {
		t.Fatalf("got= %#v\nwant=%#v\n", v, s)
	}

}

func TestSliceOfStructWithStruct(t *testing.T) {
	type T1 struct {
		ID    uint64
		Name  string
		Slice []int
	}
	type T2 uint64
	type Struct struct {
		V1 T1
		V2 T2
		V3 T1
	}

	s := []Struct{
		{V1: T1{1, "1", []int{1}}, V2: 2, V3: T1{3, "3", []int{3}}},
	}
	buf := new(bytes.Buffer)
	enc := NewEncoder(buf)
	err := enc.Encode(&s)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	v := []Struct{}
	dec := NewDecoder(buf)
	err = dec.Decode(&v)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	if !reflect.DeepEqual(s, v) {
		t.Fatalf("got= %#v\nwant=%#v\n", v, s)
	}

}

func TestBasicTypePointers(t *testing.T) {
	type BT struct {
		B    *bool
		S    *string
		I    *int
		I8   *int8
		I16  *int16
		I32  *int32
		I64  *int64
		Ui   *uint
		Ui8  *uint8
		Ui16 *uint16
		Ui32 *uint32
		Ui64 *uint64
		F32  *float32
		F64  *float64
		C64  *complex64
		C128 *complex128
	}
	toss := func(chance float32) bool {
		return rand.Float32() < chance
	}
	fuzz := func(bt *BT, nilChance float32) {
		if toss(nilChance) {
			k := rand.Intn(2) == 1
			bt.B = &k
		}
		if toss(nilChance) {
			b := make([]byte, rand.Intn(32))
			rand.Read(b)
			sb := string(b)
			bt.S = &sb
		}
		if toss(nilChance) {
			i := rand.Int()
			bt.I = &i
		}
		if toss(nilChance) {
			i8 := int8(rand.Int())
			bt.I8 = &i8
		}
		if toss(nilChance) {
			i16 := int16(rand.Int())
			bt.I16 = &i16
		}
		if toss(nilChance) {
			i32 := rand.Int31()
			bt.I32 = &i32
		}
		if toss(nilChance) {
			i64 := rand.Int63()
			bt.I64 = &i64
		}
		if toss(nilChance) {
			ui := uint(rand.Uint64())
			bt.Ui = &ui
		}
		if toss(nilChance) {
			ui8 := uint8(rand.Uint32())
			bt.Ui8 = &ui8
		}
		if toss(nilChance) {
			ui16 := uint16(rand.Uint32())
			bt.Ui16 = &ui16
		}
		if toss(nilChance) {
			ui32 := rand.Uint32()
			bt.Ui32 = &ui32
		}
		if toss(nilChance) {
			ui64 := rand.Uint64()
			bt.Ui64 = &ui64
		}
		if toss(nilChance) {
			f32 := rand.Float32()
			bt.F32 = &f32
		}
		if toss(nilChance) {
			f64 := rand.Float64()
			bt.F64 = &f64
		}
		if toss(nilChance) {
			c64 := complex(rand.Float32(), rand.Float32())
			bt.C64 = &c64
		}
		if toss(nilChance) {
			c128 := complex(rand.Float64(), rand.Float64())
			bt.C128 = &c128
		}
	}
	for _, nilChance := range []float32{.5, 0, 1} {
		for i := 0; i < 10; i += 1 {
			btOrig := &BT{}
			fuzz(btOrig, nilChance)
			payload, err := Marshal(btOrig)
			if err != nil {
				t.Errorf("marshalling failed basic type struct for: %+v, err=%+v", btOrig, err)
				continue
			}
			btDecoded := &BT{}
			err = Unmarshal(payload, btDecoded)
			if err != nil {
				t.Errorf("unmarshalling failed for: %+v, err=%+v", btOrig, err)
				continue
			}
		}
	}
}

func TestPointerOfPointer(t *testing.T) {
	type S struct {
		V **int
	}
	i := rand.Int()
	pi := &i
	ppi := &pi
	sOrig := &S{
		V: ppi,
	}
	payload, err := Marshal(sOrig)
	if err != nil {
		t.Errorf("marshalling failed pointer of pointer type for: %+v, err=%+v", sOrig, err)
		return
	}
	sDecoded := &S{}
	err = Unmarshal(payload, sDecoded)
	if err != nil {
		t.Errorf("unmarshalling failed pointer of pointer type for: %+v, err=%+v", sOrig, err)
		return
	}
	if sDecoded.V == nil {
		t.Errorf("unmarshalling failed for pointer of pointer: expected non-nil pointer of pointer value")
		return
	}

	if *sDecoded.V == nil {
		t.Errorf("unmarshalling failed for pointer of pointer: expected non-nil pointer value")
		return
	}
	if **sDecoded.V != i {
		t.Errorf("unmarshalling failed for pointer of pointer: expected: %d, actual: %d", i, **sDecoded.V)
		return
	}
}

func TestStructPointer(t *testing.T) {
	type T struct {
		V int
	}
	type S struct {
		T *T
	}
	sOrig := &S{
		T: &T{
			V: rand.Int(),
		},
	}
	payload, err := Marshal(sOrig)
	if err != nil {
		t.Errorf("marshalling failed for struct containing pointer of another struct: %+v, err=%+v", sOrig, err)
		return
	}
	sDecoded := &S{}
	err = Unmarshal(payload, sDecoded)
	if err != nil {
		t.Errorf("unmarshalling failed for struct containing pointer of another struct: %+v, err=%+v", sOrig, err)
		return
	}
	if sDecoded.T == nil {
		t.Errorf("unmarshalling failed for struct containing pointer of another struct: expecting non-nil pointer value")
		return
	}
	if sDecoded.T.V != sOrig.T.V {
		t.Errorf(
			"unmarshalling failed for struct containing pointer of another struct: expected: %d, actual: %d",
			sOrig.T.V, sDecoded.T.V,
		)
	}
}

func TestMarshalNonPointer(t *testing.T) {
	type S struct {
		A int
	}
	s := S{A: 1}
	data, err := Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var res S
	if err := Unmarshal(data, &res); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res, s) {
		t.Fatalf("expect %v got %v", s, res)
	}
}

func Test_Float32(t *testing.T) {
	v := float32(1.15)

	b, err := Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o float32
	err = Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Float64(t *testing.T) {
	v := float64(1.15)

	b, err := Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o float64
	err = Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}
