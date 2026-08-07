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

type s0 struct {
	A string
	B string
	C int16
}

var (
	s0v = &s0{"A", "B", 1}
	s0b = []byte{0x1, 0x41, 0x1, 0x42, 0x2}
)

type simpleStruct struct {
	Name      string
	Timestamp time.Time
	Payload   []byte
	Ssid      []uint32
}

type sliceStruct struct {
	Payload []byte
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

type failingBinary struct{}

func (failingBinary) MarshalBinary() ([]byte, error) {
	return nil, errors.New("encode failed")
}

func (*failingBinary) UnmarshalBinary([]byte) error {
	return nil
}

func TestMarshal(t *testing.T) {
	tests := map[string]func(*testing.T){
		"time slice":          testMarshalTimeSlice,
		"nil slice EOF":       testMarshalNilSliceEOF,
		"simple struct":       testMarshalSimpleStruct,
		"simple struct slice": testMarshalSimpleStructSlice,
		"complex struct":      testMarshalComplexStruct,
		"binary marshaler":    testMarshalBinaryMarshaler,
		"type alias":          testMarshalTypeAlias,
		"non-pointer value":   testMarshalNonPointer,
		"big struct":          testMarshalBigStruct,
	}
	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			fn(t)
		})
	}
}

func testMarshalTimeSlice(t *testing.T) {
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

func testMarshalNilSliceEOF(t *testing.T) {
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

func testMarshalSimpleStruct(t *testing.T) {
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

func testMarshalSimpleStructSlice(t *testing.T) {
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

func TestSliceWireFormat(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  []byte
	}{
		{"ints", []int64{-2, -1, 0, 1, 2}, []byte{5, 3, 1, 0, 2, 4}},
		{"uints", []uint64{0, 1, 127, 128}, []byte{4, 0, 1, 127, 128, 1}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Marshal(tc.value)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDecodePopulatedSlice(t *testing.T) {
	type item struct {
		Bytes []byte
		Uints []uint64
	}

	want := []item{
		{Bytes: []byte("first"), Uints: []uint64{1, 2, 3}},
		{Bytes: []byte("second"), Uints: []uint64{4, 5, 6}},
	}
	encoded, err := Marshal(want)
	assert.NoError(t, err)

	got := []item{
		{Bytes: make([]byte, 1, 32), Uints: make([]uint64, 1, 32)},
		{Bytes: make([]byte, 1, 32), Uints: make([]uint64, 1, 32)},
	}
	for range 2 {
		assert.NoError(t, Unmarshal(encoded, &got))
		assert.Equal(t, want, got)
	}
}

func TestDecodeEmptySlices(t *testing.T) {
	type item struct {
		Bytes []byte
		Uints []uint64
	}

	got := []item{{Bytes: []byte{1}, Uints: []uint64{1}}}
	encoded, err := Marshal([]item{{}})
	assert.NoError(t, err)
	assert.NoError(t, Unmarshal(encoded, &got))
	if assert.Len(t, got, 1) {
		assert.Empty(t, got[0].Bytes)
		assert.Empty(t, got[0].Uints)
	}

	encoded, err = Marshal([]item{})
	assert.NoError(t, err)
	assert.NoError(t, Unmarshal(encoded, &got))
	assert.Empty(t, got)
}

func TestPointerSliceReuse(t *testing.T) {
	type item struct {
		Value int64
	}

	first, err := Marshal([]*item{{Value: 1}, {Value: 2}})
	assert.NoError(t, err)
	second, err := Marshal([]*item{nil, {Value: 3}})
	assert.NoError(t, err)

	got := []*item{{Value: 9}, {Value: 8}}
	assert.NoError(t, Unmarshal(first, &got))
	reused := got[1]
	assert.NoError(t, Unmarshal(second, &got))
	assert.Nil(t, got[0])
	if got[1] != reused {
		t.Fatalf("pointer was not reused: got %p want %p", got[1], reused)
	}
	assert.Equal(t, int64(3), got[1].Value)

	empty, err := Marshal([]*item{})
	assert.NoError(t, err)
	assert.NoError(t, Unmarshal(empty, &got))
	assert.Empty(t, got)

	for _, data := range [][]byte{{}, {1}, {1, 0}} {
		assert.Error(t, Unmarshal(data, &got))
	}
}

func TestPointerClear(t *testing.T) {
	type item struct {
		Value *int64
	}

	value := int64(1)
	encoded, err := Marshal(item{})
	assert.NoError(t, err)

	got := item{Value: &value}
	assert.NoError(t, Unmarshal(encoded, &got))
	assert.Nil(t, got.Value)
}

func TestDecodePopulatedMap(t *testing.T) {
	want := map[string][]byte{
		"first":  []byte("one"),
		"second": []byte("two"),
	}
	encoded, err := Marshal(want)
	assert.NoError(t, err)

	got := map[string][]byte{"stale": []byte("value")}
	for range 2 {
		assert.NoError(t, Unmarshal(encoded, &got))
		assert.Equal(t, want, got)
	}

	encoded, err = Marshal(map[string][]byte{})
	assert.NoError(t, err)
	assert.NoError(t, Unmarshal(encoded, &got))
	assert.Empty(t, got)
}

func TestNumericSlices(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"int8", []int8{-2, 0, 2}},
		{"int16", []int16{-2, 0, 2}},
		{"int32", []int32{-2, 0, 2}},
		{"int64", []int64{-2, 0, 2}},
		{"uint16", []uint16{0, 127, 128}},
		{"uint32", []uint32{0, 127, 128}},
		{"uint64", []uint64{0, 127, 128}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want, err := Marshal(tc.value)
			assert.NoError(t, err)

			var writer struct{ bytes.Buffer }
			assert.NoError(t, MarshalTo(tc.value, &writer))
			assert.Equal(t, want, writer.Bytes())

			got := reflect.New(reflect.TypeOf(tc.value))
			assert.NoError(t, Unmarshal(want, got.Interface()))
			assert.Equal(t, tc.value, got.Elem().Interface())
		})
	}
}

func TestMapDecode(t *testing.T) {
	data := append([]byte{1, 0, 0}, bytes.Repeat([]byte{0xff}, 9)...)
	data = append(data, 1)

	var got map[string][]byte
	for _, data := range [][]byte{{}, {1}, {1, 1, 0}, {1, 0, 0}} {
		assert.Error(t, Unmarshal(data, &got))
	}
	assert.Error(t, Unmarshal(data, &got))
	assert.Error(t, Unmarshal([]byte{1, 0, 0, 1}, &got))

	var strings map[string]string
	for _, data := range [][]byte{{}, {1}, {1, 1, 0}, {1, 0, 0}} {
		assert.Error(t, Unmarshal(data, &strings))
	}

	encoded, err := Marshal(map[string]string{"fresh": "value"})
	assert.NoError(t, err)
	strings = map[string]string{"stale": "value"}
	assert.NoError(t, Unmarshal(encoded, &strings))
	assert.Equal(t, map[string]string{"fresh": "value"}, strings)
}

func TestCustomDecode(t *testing.T) {
	type payload struct {
		Value *s2
	}

	data, err := Marshal(payload{Value: &s2{b: []byte{0x13}}})
	assert.NoError(t, err)

	got := payload{Value: &s2{b: []byte{0x99}}}
	assert.NoError(t, Unmarshal(data, &got))
	assert.Equal(t, []byte{0x13}, got.Value.b)

	data, err = Marshal(payload{})
	assert.NoError(t, err)
	assert.NoError(t, Unmarshal(data, &got))
	assert.Nil(t, got.Value)
}

func TestPointerEncodeError(t *testing.T) {
	type inner struct {
		Value failingBinary
	}
	type outer struct {
		Value *inner
	}

	_, err := Marshal(outer{Value: &inner{}})
	assert.Error(t, err)
}

func testMarshalComplexStruct(t *testing.T) {
	b, err := Marshal(s1v)
	assert.NoError(t, err)
	assert.Equal(t, svb, b)

	s := &s1{}
	err = Unmarshal(b, s)
	assert.NoError(t, err)
	assert.Equal(t, s1v, s)
}

func testMarshalBinaryMarshaler(t *testing.T) {
	s2v := &s2{[]byte{0x13}}
	b, err := Marshal(s2v)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x1, 0x13}, b)
}

func testMarshalTypeAlias(t *testing.T) {
	type Foo int64
	f := Foo(32)
	b, err := Marshal(f)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x40}, b)
}

func testMarshalNonPointer(t *testing.T) {
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

func testMarshalBigStruct(t *testing.T) {
	input := newBigStruct()
	b, err := Marshal(input)
	assert.NoError(t, err)

	var output bigStruct
	assert.NoError(t, Unmarshal(b, &output))
	assert.Equal(t, input, &output)
}

func TestStruct(t *testing.T) {
	tests := map[string]func(*testing.T){
		"nested fields":            testStructNestedFields,
		"embedded struct":          testStructEmbedded,
		"array of struct":          testStructArray,
		"slice of struct":          testStructSlice,
		"trace slice wire":         testStructTraceSliceWire,
		"decode error keeps field": testStructDecodeErrorKeepsField,
		"custom field codec":       testStructCustomFieldCodec,
	}
	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			fn(t)
		})
	}
}

func testStructNestedFields(t *testing.T) {
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

func testStructEmbedded(t *testing.T) {
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

func testStructArray(t *testing.T) {
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

func testStructSlice(t *testing.T) {
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

type tracePayload struct {
	Spans []traceSpan
}

type traceSpan struct {
	Ordinal    uint32
	At         int64
	Scope      string
	Node       string
	NodeType   string
	Phase      string
	Invocation uint32
	Target     string
}

type customFieldString string

func (customFieldString) MarshalBinary() ([]byte, error) {
	return []byte("custom"), nil
}

func (s *customFieldString) UnmarshalBinary([]byte) error {
	*s = "decoded"
	return nil
}

func testStructTraceSliceWire(t *testing.T) {
	want := tracePayload{Spans: []traceSpan{{
		Ordinal:    1,
		At:         2,
		Scope:      "s",
		Node:       "n",
		NodeType:   "t",
		Phase:      "p",
		Invocation: 3,
		Target:     "x",
	}}}
	wire := []byte{1, 1, 4, 1, 's', 1, 'n', 1, 't', 1, 'p', 3, 1, 'x'}

	encoded, err := Marshal(&want)
	assert.NoError(t, err)
	assert.Equal(t, wire, encoded)

	encodedValue, err := Marshal(want)
	assert.NoError(t, err)
	assert.Equal(t, wire, encodedValue)

	var got tracePayload
	assert.NoError(t, Unmarshal(wire, &got))
	assert.Equal(t, want, got)
}

func testStructDecodeErrorKeepsField(t *testing.T) {
	type payload struct {
		Prefix uint32
		Value  string
	}

	got := payload{Value: "keep"}
	err := Unmarshal([]byte{1, 2, 'x'}, &got)
	assert.Error(t, err)
	assert.Equal(t, payload{Prefix: 1, Value: "keep"}, got)
}

func testStructCustomFieldCodec(t *testing.T) {
	type payload struct {
		Value customFieldString
	}

	wire, err := Marshal(&payload{Value: "input"})
	assert.NoError(t, err)
	assert.Equal(t, []byte{6, 'c', 'u', 's', 't', 'o', 'm'}, wire)

	var got payload
	assert.NoError(t, Unmarshal(wire, &got))
	assert.Equal(t, payload{Value: "decoded"}, got)
}

func TestPointer(t *testing.T) {
	tests := map[string]func(*testing.T){
		"basic types":            testPointerBasicTypes,
		"pointer of pointer":     testPointerOfPointer,
		"struct pointer field":   testPointerStructField,
		"slice of pointers":      testPointerSlice,
		"slice of time pointers": testPointerTimeSlice,
	}
	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			fn(t)
		})
	}
}

func testPointerBasicTypes(t *testing.T) {
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
		for range 10 {
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

func testPointerOfPointer(t *testing.T) {
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

func testPointerStructField(t *testing.T) {
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

func testPointerSlice(t *testing.T) {
	type A struct {
		V int64
	}

	v := []*A{{1}, nil, {2}}
	b, err := Marshal(v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o []*A
	err = Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func testPointerTimeSlice(t *testing.T) {
	type A struct {
		T0 *time.Time
		T1 *time.Time
		T2 time.Time
	}

	x := time.Unix(1637686933, 0)
	v := []*A{{&x, nil, x}}
	b, err := Marshal(v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o []*A
	err = Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func TestStructUintComplex(t *testing.T) {
	type allTypes struct {
		U    uint
		C64  complex64
		C128 complex128
	}
	in := allTypes{U: 42, C64: 1 + 2i, C128: 3 + 4i}
	b, err := Marshal(&in)
	assert.NoError(t, err)

	var out allTypes
	assert.NoError(t, Unmarshal(b, &out))
	assert.Equal(t, in, out)

	// By-value (non-addressable) exercises the reflect fallback encode path
	b2, err := Marshal(in)
	assert.NoError(t, err)

	var out2 allTypes
	assert.NoError(t, Unmarshal(b2, &out2))
	assert.Equal(t, in, out2)
}

func TestStructReflectPath(t *testing.T) {
	// Struct with non-primitive fields forces the reflect fallback encode/decode path.
	type inner struct {
		Items []string
		Tags  map[string]int
	}
	type outer struct {
		Name  string
		Inner inner
	}

	// Encode by value (non-addressable) to exercise the reflect-path encode
	in := outer{Name: "hi", Inner: inner{
		Items: []string{"a", "b"},
		Tags:  map[string]int{"x": 1},
	}}
	b, err := Marshal(in)
	assert.NoError(t, err)

	var out outer
	assert.NoError(t, Unmarshal(b, &out))
	assert.Equal(t, in, out)
}

func TestStructSkippedField(t *testing.T) {
	type withSkip struct {
		_    int `binary:"-"`
		Name string
		Skip int `binary:"-"`
		Val  uint
	}
	in := withSkip{Name: "test", Val: 42}
	b, err := Marshal(&in)
	assert.NoError(t, err)

	var out withSkip
	assert.NoError(t, Unmarshal(b, &out))
	assert.Equal(t, in.Name, out.Name)
	assert.Equal(t, in.Val, out.Val)
}

func TestSliceEncodeError(t *testing.T) {
	// varint slice with items
	v := []int64{1, 2, 3, 4, 5}
	b, err := Marshal(&v)
	assert.NoError(t, err)

	var out []int64
	assert.NoError(t, Unmarshal(b, &out))
	assert.Equal(t, v, out)

	// varuint slice
	v2 := []uint64{10, 20, 30}
	b2, err := Marshal(&v2)
	assert.NoError(t, err)

	var out2 []uint64
	assert.NoError(t, Unmarshal(b2, &out2))
	assert.Equal(t, v2, out2)

	_, err = Marshal([1]failingBinary{{}})
	assert.Error(t, err)
	_, err = Marshal([]failingBinary{{}})
	assert.Error(t, err)
}

func TestMapEncodeError(t *testing.T) {
	v := map[int]string{1: "a", 2: "b"}
	b, err := Marshal(&v)
	assert.NoError(t, err)

	out := map[int]string{99: "stale"}
	assert.NoError(t, Unmarshal(b, &out))
	assert.Equal(t, v, out)

	_, err = Marshal(map[failingBinary]string{failingBinary{}: "value"})
	assert.Error(t, err)
	_, err = Marshal(map[string]failingBinary{"key": {}})
	assert.Error(t, err)
}

func TestEncoderBuffer(t *testing.T) {
	var buf bytes.Buffer
	e := NewEncoder(&buf)
	assert.Equal(t, &buf, e.Buffer())
}

func TestReaderHelpers(t *testing.T) {
	r := newSliceReader([]byte{1, 2, 3, 4, 5})
	assert.Equal(t, int64(5), r.Size())
	assert.Equal(t, 5, r.Len())

	// Read past end
	r.offset = 100
	assert.Equal(t, 0, r.Len())
}

func TestScanUnsupported(t *testing.T) {
	_, err := scanType(reflect.TypeFor[chan int]())
	assert.Error(t, err)
}

func TestScanErrors(t *testing.T) {
	type badSliceElem struct {
		V []chan int
	}
	_, err := scanType(reflect.TypeFor[badSliceElem]())
	assert.Error(t, err)

	type badPtr struct {
		V *chan int
	}
	_, err = scanType(reflect.TypeFor[badPtr]())
	assert.Error(t, err)

	type badArray struct {
		V [2]chan int
	}
	_, err = scanType(reflect.TypeFor[badArray]())
	assert.Error(t, err)

	type badMapKey struct {
		V map[chan int]string
	}
	_, err = scanType(reflect.TypeFor[badMapKey]())
	assert.Error(t, err)

	type badMapVal struct {
		V map[string]chan int
	}
	_, err = scanType(reflect.TypeFor[badMapVal]())
	assert.Error(t, err)

	type badStructField struct {
		V chan int
	}
	_, err = scanType(reflect.TypeFor[badStructField]())
	assert.Error(t, err)

	type badSliceOfPtr struct {
		V []*chan int
	}
	_, err = scanType(reflect.TypeFor[badSliceOfPtr]())
	assert.Error(t, err)
}

func TestFloat(t *testing.T) {
	tests := map[string]struct {
		marshal   func() (any, []byte, error)
		unmarshal func([]byte) (any, error)
		want      any
	}{
		"float32": {
			marshal: func() (any, []byte, error) {
				v := float32(1.15)
				b, err := Marshal(&v)
				return v, b, err
			},
			unmarshal: func(b []byte) (any, error) {
				var o float32
				err := Unmarshal(b, &o)
				return o, err
			},
		},
		"float64": {
			marshal: func() (any, []byte, error) {
				v := float64(1.15)
				b, err := Marshal(&v)
				return v, b, err
			},
			unmarshal: func(b []byte) (any, error) {
				var o float64
				err := Unmarshal(b, &o)
				return o, err
			},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			want, b, err := tc.marshal()
			assert.NoError(t, err)
			assert.NotNil(t, b)

			got, err := tc.unmarshal(b)
			assert.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
