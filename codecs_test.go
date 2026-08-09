// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"bytes"
	stdbinary "encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"
	"unsafe"

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

type arenaMapKey string
type arenaMapBytes []byte
type emptyMapValue struct{}

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

type customPointer struct {
	value string
}

type malformedBinary struct{}

func (malformedBinary) MarshalBinary(int) ([]byte, error) { return nil, nil }
func (*malformedBinary) UnmarshalBinary([]byte) error     { return nil }

type malformedCodec struct{}

func (*malformedCodec) GetBinaryCodec(int) Codec { return nil }

type nilCodecType struct{}

func (*nilCodecType) GetBinaryCodec() Codec { return nil }

func (v customPointer) MarshalBinary() ([]byte, error) {
	return []byte(v.value), nil
}

func (v *customPointer) UnmarshalBinary(data []byte) error {
	v.value = string(data)
	return nil
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

func TestStringMapEncode(t *testing.T) {
	for _, count := range []int{7, 8, 100} {
		value := make(map[string]string, count)
		for i := range count {
			value[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
		}

		encoded, err := Marshal(value)
		assert.NoError(t, err)
		var got map[string]string
		assert.NoError(t, Unmarshal(encoded, &got))
		assert.Equal(t, value, got)
	}
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

func TestStreamSlices(t *testing.T) {
	tests := []struct {
		name  string
		value any
		out   any
	}{
		{"bytes", []byte{1, 2, 3}, new([]byte)},
		{"empty bytes", []byte{}, new([]byte)},
		{"bools", []bool{true, false, true}, new([]bool)},
		{"strings", []string{"one", "two"}, new([]string)},
		{"structs", []simpleStruct{{Name: "one"}, {Name: "two"}}, new([]simpleStruct)},
		{"pointers", []*simpleStruct{{Name: "one"}, nil}, new([]*simpleStruct)},
		{"int8", []int8{-2, 0, 2}, new([]int8)},
		{"int16", []int16{-2, 0, 2}, new([]int16)},
		{"int32", []int32{-2, 0, 2}, new([]int32)},
		{"int64", []int64{-2, 0, 2}, new([]int64)},
		{"uint16", []uint16{0, 127, 128}, new([]uint16)},
		{"uint32", []uint32{0, 127, 128}, new([]uint32)},
		{"uint64", []uint64{0, 127, 128}, new([]uint64)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := Marshal(tc.value)
			assert.NoError(t, err)
			assert.NoError(t, NewDecoder(bytes.NewReader(encoded)).Decode(tc.out))
			if reflect.ValueOf(tc.out).Elem().Len() == 0 {
				assert.Empty(t, reflect.ValueOf(tc.out).Elem().Interface())
			} else {
				assert.Equal(t, tc.value, reflect.ValueOf(tc.out).Elem().Interface())
			}
		})
	}
}

func TestStreamErrors(t *testing.T) {
	tests := []struct {
		name string
		out  any
	}{
		{"bytes", new([]byte)},
		{"bools", new([]bool)},
		{"strings", new([]string)},
		{"pointers", new([]*int)},
		{"signed", new([]int16)},
		{"unsigned", new([]uint16)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, NewDecoder(bytes.NewReader([]byte{1})).Decode(tc.out))
		})
	}
}

func TestSliceResizeErrors(t *testing.T) {
	var value []int
	assert.Equal(t, io.ErrUnexpectedEOF, resizeSliceChecked(reflect.ValueOf(&value).Elem(), -1))
	assert.Equal(t, io.ErrUnexpectedEOF, makeSliceChecked(reflect.ValueOf(&value).Elem(), -1))
	assert.Equal(t, io.ErrUnexpectedEOF, makeSliceChecked(reflect.ValueOf(value), 1))

	d := NewDecoder(bytes.NewReader([]byte{4, 't', 'e', 's', 't'}))
	got, err := d.readString("test")
	assert.NoError(t, err)
	assert.Equal(t, "test", got)
	assert.Equal(t, io.ErrUnexpectedEOF, validateSliceLength(reflect.TypeFor[[]byte](), -1))
	assert.Equal(t, io.ErrUnexpectedEOF, validateSliceLength(reflect.TypeFor[[]byte](), int(^uint(0)>>1)))
	assert.Equal(t, io.ErrUnexpectedEOF, d.ensureAvailable(-1))
	assert.Equal(t, io.ErrUnexpectedEOF, d.ensureElements(-1, 1))
	sliceDecoder := NewDecoder(bytes.NewBuffer(nil))
	assert.Equal(t, io.EOF, sliceDecoder.ensureAvailable(5))
	assert.Equal(t, io.EOF, sliceDecoder.ensureElements(5, 1))
}

func TestCodecMalformedLengths(t *testing.T) {
	huge := stdbinary.AppendUvarint(nil, ^uint64(0))
	for _, out := range []any{
		new([]byte),
		new([]bool),
		new([]string),
		new([]int),
		new([]*int),
		new([]simpleStruct),
	} {
		assert.Error(t, Unmarshal(huge, out))
	}

	var structs []simpleStruct
	assert.Error(t, Unmarshal([]byte{2}, &structs))
	var strings [2]string
	assert.Error(t, Unmarshal(nil, &strings))

	var zeroWire []struct{ _ int }
	encoded, err := Marshal([]struct{ _ int }{{}, {}})
	assert.NoError(t, err)
	assert.NoError(t, Unmarshal(encoded, &zeroWire))
	assert.Len(t, zeroWire, 2)
}

func TestVarintDecoderHelpers(t *testing.T) {
	signed := []struct {
		name string
		size uintptr
		out  any
	}{
		{"int8", 1, []int8{0}},
		{"int16", 2, []int16{0}},
		{"int32", 4, []int32{0}},
		{"int64", 8, []int64{0}},
	}
	for _, tc := range signed {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDecoder(bytes.NewReader([]byte{1}))
			value := reflect.ValueOf(tc.out)
			assert.NoError(t, decodeVarints(d, unsafe.Pointer(value.Pointer()), 1, tc.size))
		})
	}

	unsigned := []struct {
		name string
		size uintptr
		out  any
	}{
		{"uint16", 2, []uint16{0}},
		{"uint32", 4, []uint32{0}},
		{"uint64", 8, []uint64{0}},
	}
	for _, tc := range unsigned {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDecoder(bytes.NewReader([]byte{1}))
			value := reflect.ValueOf(tc.out)
			assert.NoError(t, decodeVaruints(d, unsafe.Pointer(value.Pointer()), 1, tc.size))
		})
	}

	signedValue := []int16{0}
	unsignedValue := []uint16{0}
	assert.Error(t, decodeVarints(NewDecoder(bytes.NewReader(nil)), unsafe.Pointer(unsafe.SliceData(signedValue)), 1, 2))
	assert.Error(t, decodeVaruints(NewDecoder(bytes.NewReader(nil)), unsafe.Pointer(unsafe.SliceData(unsignedValue)), 1, 2))
}

func TestGenericMapCodecs(t *testing.T) {
	input := map[arenaMapKey]arenaMapBytes{
		"a": []byte("x"),
		"b": []byte("y"),
	}
	encoded, err := Marshal(input)
	assert.NoError(t, err)
	var got map[arenaMapKey]arenaMapBytes
	assert.NoError(t, Unmarshal(encoded, &got))
	assert.Equal(t, input, got)
	sibling := got["b"]
	_ = append(got["a"], 'z', 'q')
	assert.Equal(t, arenaMapBytes("y"), sibling)

	var streamed map[arenaMapKey]arenaMapBytes
	assert.NoError(t, NewDecoder(bytes.NewReader(encoded)).Decode(&streamed))
	assert.Equal(t, input, streamed)

	zero := map[emptyMapValue]emptyMapValue{{}: {}}
	zeroData, err := Marshal(zero)
	assert.NoError(t, err)
	var zeroGot map[emptyMapValue]emptyMapValue
	assert.NoError(t, Unmarshal(zeroData, &zeroGot))
	assert.Equal(t, zero, zeroGot)

	withStructKey := map[emptyMapValue]string{{}: "value"}
	withStructKeyData, err := Marshal(withStructKey)
	assert.NoError(t, err)
	var withStructKeyGot map[emptyMapValue]string
	assert.NoError(t, Unmarshal(withStructKeyData, &withStructKeyGot))
	assert.Equal(t, withStructKey, withStructKeyGot)

	long := string(bytes.Repeat([]byte{'k'}, maxMapKeyLength+1))
	_, err = Marshal(map[arenaMapKey]arenaMapBytes{arenaMapKey(long): {'x'}})
	assert.True(t, errors.Is(err, errMapKeyTooLong))

	for _, value := range []any{
		map[string]string{"a": "b"},
		map[string][]byte{"a": []byte("b")},
		map[string]uint64{"a": 1},
	} {
		data, err := Marshal(value)
		assert.NoError(t, err)
		out := reflect.New(reflect.TypeOf(value))
		assert.NoError(t, NewDecoder(bytes.NewReader(data)).Decode(out.Interface()))
		assert.Equal(t, value, out.Elem().Interface())
	}
}

func TestStringMapHelpers(t *testing.T) {
	assert.Equal(t, 2, stringMapValueSize([]byte("x")))
	assert.Equal(t, []byte{1, 'x'}, appendStringMapValue[[]byte](nil, []byte("x")))

	value, err := readStringMapValue[string](NewDecoder(bytes.NewBuffer([]byte{1, 'x'})), nil)
	assert.NoError(t, err)
	assert.Equal(t, "x", value)
	empty, err := readStringMapValue[[]byte](NewDecoder(bytes.NewBuffer([]byte{0})), nil)
	assert.NoError(t, err)
	assert.Nil(t, empty)
	_, err = readStringMapValue[string](NewDecoder(bytes.NewBuffer(nil)), nil)
	assert.Error(t, err)

	assert.Error(t, Unmarshal(stdbinary.AppendUvarint(nil, ^uint64(0)), new(map[string]string)))
	assert.Error(t, Unmarshal([]byte{1, 2, 0, 'k'}, new(map[string]string)))

	long := string(bytes.Repeat([]byte{'k'}, maxMapKeyLength+1))
	large := make(map[string]string, 8)
	large[long] = "value"
	for i := 0; i < 7; i++ {
		large[fmt.Sprintf("key-%d", i)] = "value"
	}
	_, err = Marshal(large)
	assert.True(t, errors.Is(err, errMapKeyTooLong))
}

func TestScannerErrorPaths(t *testing.T) {
	_, err := Marshal(make(chan int))
	assert.Error(t, err)
	_, err = scanType(reflect.TypeFor[nilCodecType]())
	assert.Error(t, err)
	assert.True(t, isNilInterface(nil))
	var nilSlice []byte
	assert.True(t, isNilInterface(nilSlice))
}

func TestCodecMetadata(t *testing.T) {
	emptyStruct := &reflectStructCodec{}
	assert.True(t, isZeroWireCodec(emptyStruct))
	assert.Equal(t, 0, wireMinBytes(emptyStruct))

	zeroCollection := &reflectCollectionCodec{elemCodec: emptyStruct, array: true}
	assert.True(t, isZeroWireCodec(zeroCollection))
	assert.Equal(t, 0, wireMinBytes(zeroCollection))
	collection := &reflectCollectionCodec{elemCodec: &primitiveCodec{}, array: true, length: 2}
	assert.False(t, isZeroWireCodec(collection))
	assert.Equal(t, 2, wireMinBytes(collection))
	assert.Equal(t, 1, wireMinBytes(&reflectCollectionCodec{elemCodec: &primitiveCodec{}}))

	assert.True(t, isZeroWireCodec(&stringSliceCodec{array: true}))
	assert.False(t, isZeroWireCodec(&stringSliceCodec{array: true, length: 1}))
	assert.True(t, isZeroWireCodec(&fixedSliceCodec{array: true}))
	assert.False(t, isZeroWireCodec(&fixedSliceCodec{array: true, length: 1}))
	assert.Equal(t, 2, wireMinBytes(&stringSliceCodec{array: true, length: 2}))
	assert.Equal(t, 1, wireMinBytes(&stringSliceCodec{}))
	assert.Equal(t, 8, wireMinBytes(&fixedSliceCodec{array: true, length: 2, elemSize: 4}))
	assert.Equal(t, 1, wireMinBytes(&fixedSliceCodec{elemSize: 4}))

	assert.Equal(t, 2, wireMinBytes(&reflectUnionCodec{}))
	assert.Equal(t, 1, wireMinBytes(stringMapCodec[string]{}))
	assert.Equal(t, 1, wireMinBytes(&reflectSliceOfPtrCodec{}))
	assert.Equal(t, 1, wireMinBytes(&primitiveCodec{}))
	assert.Equal(t, 0, wireMinBytes(nil))
	assert.Equal(t, 1, wireMinBytes(&reflectStructCodec{{Field: fieldIncluded, Codec: &primitiveCodec{}}}))
}

func TestCollectionErrors(t *testing.T) {
	var values []s2
	assert.Error(t, Unmarshal(nil, &values))
	encoded, err := Marshal([]s2{{b: []byte{1}}})
	assert.NoError(t, err)
	assert.NoError(t, NewDecoder(bytes.NewReader(encoded)).Decode(&values))
	assert.Error(t, NewDecoder(bytes.NewReader([]byte{1, 0})).Decode(&values))

	var pointers []*int
	assert.Error(t, Unmarshal([]byte{1}, &pointers))
	var bytesValue []byte
	assert.Error(t, Unmarshal([]byte{1}, &bytesValue))

	array := [2]struct{}{}
	zeroArray := &reflectCollectionCodec{elemCodec: &reflectStructCodec{}, array: true, length: 2}
	assert.NoError(t, zeroArray.DecodeTo(NewDecoder(bytes.NewBuffer(nil)), reflect.ValueOf(&array).Elem()))

	huge := stdbinary.AppendUvarint(nil, ^uint64(0))
	var custom s2
	assert.Error(t, Unmarshal(huge, &custom))
	assert.Error(t, Unmarshal(stdbinary.AppendUvarint(nil, uint64(^uint(0)>>1)), &custom))
	assert.Error(t, NewDecoder(bytes.NewReader([]byte{1})).Decode(&custom))

	var numeric map[uint64]uint64
	assert.Error(t, Unmarshal(huge, &numeric))
	var generic map[int]string
	assert.Error(t, Unmarshal(huge, &generic))
}

func TestFixedWidthSlices(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"float32", []float32{0, 1.25, -2.5, 3, 4, 5, 6, 7, 8}},
		{"float64", []float64{0, 1.25, -2.5, 3, 4, 5, 6, 7, 8}},
		{"complex64", []complex64{0, 1.25 - 2.5i, 3, 4, 5, 6, 7, 8}},
		{"complex128", []complex128{0, 1.25 - 2.5i, 3, 4, 5, 6, 7, 8}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want, err := Marshal(tc.value)
			assert.NoError(t, err)

			var writer bytes.Buffer
			assert.NoError(t, MarshalTo(tc.value, &writer))
			assert.Equal(t, want, writer.Bytes())

			got := reflect.New(reflect.TypeOf(tc.value))
			assert.NoError(t, Unmarshal(want, got.Interface()))
			assert.Equal(t, tc.value, got.Elem().Interface())

			streamed := reflect.New(reflect.TypeOf(tc.value))
			assert.NoError(t, NewDecoder(bytes.NewReader(want)).Decode(streamed.Interface()))
			assert.Equal(t, tc.value, streamed.Elem().Interface())
		})
	}
}

func TestStringSlice(t *testing.T) {
	value := make([]string, 8)
	value[0] = string(bytes.Repeat([]byte{'x'}, 128))
	for i := 1; i < len(value); i++ {
		value[i] = fmt.Sprintf("value-%d", i)
	}

	encoded, err := Marshal(value)
	assert.NoError(t, err)

	var got []string
	assert.NoError(t, Unmarshal(encoded, &got))
	assert.Equal(t, value, got)
}

func TestFixedWidthErrors(t *testing.T) {
	length := append(bytes.Repeat([]byte{0x80}, 9), 1)

	var floats []float32
	assert.Equal(t, io.ErrUnexpectedEOF, Unmarshal(length, &floats))

	var complexValues []complex128
	assert.Equal(t, io.ErrUnexpectedEOF, Unmarshal(length, &complexValues))
	assert.Equal(t, io.EOF, Unmarshal([]byte{0x80}, &complexValues))

	var empty []complex64
	assert.NoError(t, Unmarshal([]byte{0}, &empty))
	assert.Equal(t, io.EOF, Unmarshal([]byte{1}, &complexValues))
}

func TestComplexFallback(t *testing.T) {
	assert.NoError(t, MarshalTo([]complex64{1 + 2i}, io.Discard))
	assert.NoError(t, MarshalTo([]complex128{1 + 2i}, io.Discard))
}

func TestFixedWidthArrays(t *testing.T) {
	float32Values := [9]float32{0, 1.25, -2.5, 3, 4, 5, 6, 7, 8}
	floatValues := [9]float64{0, 1.25, -2.5, 3, 4, 5, 6, 7, 8}
	complex64Values := [9]complex64{0, 1.25 - 2.5i, 3, 4, 5, 6, 7, 8}
	complexValues := [9]complex128{0, 1.25 - 2.5i, 3, 4, 5, 6, 7, 8}
	stringValues := [9]string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
	tests := []struct {
		value   any
		pointer any
		out     any
	}{
		{float32Values, &float32Values, new([9]float32)},
		{floatValues, &floatValues, new([9]float64)},
		{complex64Values, &complex64Values, new([9]complex64)},
		{complexValues, &complexValues, new([9]complex128)},
		{stringValues, &stringValues, new([9]string)},
	}
	for _, tc := range tests {
		encoded, err := Marshal(tc.pointer)
		assert.NoError(t, err)
		assert.NoError(t, Unmarshal(encoded, tc.out))
		assert.Equal(t, tc.value, reflect.ValueOf(tc.out).Elem().Interface())

		valueEncoded, err := Marshal(tc.value)
		assert.NoError(t, err)
		assert.Equal(t, encoded, valueEncoded)
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

func TestNumericMapWire(t *testing.T) {
	u64s := make(map[uint64]uint64, 8)
	strings := make(map[string]uint64, 8)
	for i := range 8 {
		u64s[uint64(i)] = uint64(i + 1)
		strings[fmt.Sprintf("key-%d", i)] = uint64(i + 1)
	}
	smallStrings := map[string]uint64{"small": 1}
	tests := []any{u64s, strings, smallStrings}
	for _, value := range tests {
		encoded, err := Marshal(value)
		assert.NoError(t, err)

		got := reflect.New(reflect.TypeOf(value))
		assert.NoError(t, Unmarshal(encoded, got.Interface()))
		assert.Equal(t, value, got.Elem().Interface())
	}

	encoded, err := Marshal(u64s)
	assert.NoError(t, err)
	reusedU64s := map[uint64]uint64{99: 99}
	assert.NoError(t, Unmarshal(encoded, &reusedU64s))
	assert.Equal(t, u64s, reusedU64s)

	encoded, err = Marshal(smallStrings)
	assert.NoError(t, err)
	reusedStrings := map[string]uint64{"stale": 99}
	assert.NoError(t, Unmarshal(encoded, &reusedStrings))
	assert.Equal(t, smallStrings, reusedStrings)

	for _, data := range [][]byte{{}, {1}, {1, 1, 0}, {1, 0, 0}} {
		var got map[string]uint64
		assert.Error(t, Unmarshal(data, &got))
	}
}

func TestSliceReuse(t *testing.T) {
	type value struct {
		Name   string
		Bytes  []byte
		Values []uint32
	}

	want := value{Name: "value", Bytes: []byte{1, 2, 3}, Values: []uint32{4, 5, 6}}
	encoded, err := Marshal(want)
	assert.NoError(t, err)

	got := value{
		Bytes:  make([]byte, 0, len(want.Bytes)),
		Values: make([]uint32, 0, len(want.Values)),
	}
	assert.NoError(t, Unmarshal(encoded, &got))
	assert.Equal(t, want, got)
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

	t.Run("allocates pointer before unmarshaling", func(t *testing.T) {
		type payload struct {
			Value *customPointer
		}
		data, err := Marshal(payload{Value: &customPointer{value: "decoded"}})
		assert.NoError(t, err)

		var got payload
		assert.NoError(t, Unmarshal(data, &got))
		assert.Equal(t, "decoded", got.Value.value)
	})
}

func TestCustomScan(t *testing.T) {
	for name, value := range map[string]any{
		"binary": malformedBinary{},
		"codec":  malformedCodec{},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("invalid custom method panicked during scan: %v", recovered)
				}
			}()
			if _, err := Marshal(value); err != nil {
				t.Fatal(err)
			}
		})
	}
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

func TestArrayOfPointers(t *testing.T) {
	type item struct {
		Value uint64
	}
	want := [2]*item{{Value: 1}, nil}
	data, err := Marshal(&want)
	assert.NoError(t, err)

	var got [2]*item
	assert.NoError(t, Unmarshal(data, &got))
	assert.Equal(t, want, got)
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

func TestNamedPrimitives(t *testing.T) {
	type namedFloat32 float32
	type namedFloat64 float64
	type namedComplex64 complex64
	type namedComplex128 complex128

	tests := []struct {
		name  string
		value any
		out   any
	}{
		{"float32", namedFloat32(1.25), new(namedFloat32)},
		{"float64", namedFloat64(1.25), new(namedFloat64)},
		{"complex64", namedComplex64(1.25 - 2.5i), new(namedComplex64)},
		{"complex128", namedComplex128(1.25 - 2.5i), new(namedComplex128)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := Marshal(tc.value)
			assert.NoError(t, err)
			assert.NoError(t, Unmarshal(encoded, tc.out))
			assert.Equal(t, tc.value, reflect.ValueOf(tc.out).Elem().Interface())
		})
	}

	assert.Error(t, NewDecoder(bytes.NewReader(nil)).Decode(new(complex64)))
	assert.Error(t, NewDecoder(bytes.NewReader(nil)).Decode(new(complex128)))
}

type fuzzMessage struct {
	Bool       bool
	Int        int64
	Uint       uint64
	Float32    float32
	Float64    float64
	Complex64  complex64
	Complex128 complex128
	String     string
	Bytes      []byte
	Ints       []int64
	Uints      []uint64
	Strings    []string
	Floats     []float64
	Complexes  []complex128
	Child      *fuzzChild
	Children   []*fuzzChild
	Array      [2]fuzzChild
	Values     map[string][]byte
	Counts     map[uint64]uint64
	Generic    map[int32]string
	Variant    fuzzVariant
	Custom     fuzzCustom
}

type fuzzChild struct {
	ID   int32
	Name string
	Data []byte
}

type fuzzText struct {
	Msg string
}

type fuzzImage struct {
	Width  int32
	Height int32
}

type fuzzVariant struct {
	Text  *fuzzText  `binary:"1,union"`
	Image *fuzzImage `binary:"2,union"`
}

type fuzzCustom []byte

func (v fuzzCustom) MarshalBinary() ([]byte, error) {
	return append([]byte(nil), v...), nil
}

func (v *fuzzCustom) UnmarshalBinary(data []byte) error {
	*v = append((*v)[:0], data...)
	return nil
}

func FuzzDecodeWire(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte{0})
	f.Add([]byte{1})
	f.Add([]byte{0x80})
	f.Add(bytes.Repeat([]byte{0x80}, 10))
	f.Add(mustFuzzMarshal(fuzzMessage{}))
	f.Add(mustFuzzMarshal(fuzzBoundaryMessage()))

	image := fuzzBoundaryMessage()
	image.Variant = fuzzVariant{Image: &fuzzImage{Width: 7, Height: -9}}
	f.Add(mustFuzzMarshal(image))

	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 64<<10 {
			return
		}
		var got fuzzMessage
		if err := Unmarshal(wire, &got); err != nil {
			return
		}
		if _, err := Marshal(&got); err != nil {
			t.Fatal(err)
		}
		var streamed fuzzMessage
		if err := NewDecoder(&fuzzOneByteReader{data: wire}).Decode(&streamed); err == nil {
			if _, err := Marshal(&streamed); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func FuzzDecodeUnion(f *testing.F) {
	f.Add([]byte{0, 0})
	f.Add([]byte{3, 0})
	f.Add(mustFuzzMarshal(fuzzVariant{Text: &fuzzText{Msg: "text"}}))
	f.Add(mustFuzzMarshal(fuzzVariant{Image: &fuzzImage{Width: 3, Height: 4}}))

	f.Fuzz(func(t *testing.T, wire []byte) {
		if len(wire) > 64<<10 {
			return
		}
		var got fuzzVariant
		if err := Unmarshal(wire, &got); err != nil {
			return
		}
		if _, err := Marshal(&got); err != nil {
			t.Fatal(err)
		}
		var streamed fuzzVariant
		if err := NewDecoder(&fuzzOneByteReader{data: wire}).Decode(&streamed); err == nil {
			if _, err := Marshal(&streamed); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func FuzzRoundTrip(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{0},
		{1},
		{8},
		bytes.Repeat([]byte{8}, 128),
		bytes.Repeat([]byte{0xff}, 128),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		input := fuzzMessageFrom(data)
		for _, value := range []any{input, &input} {
			wire, err := Marshal(value)
			if err != nil {
				t.Fatal(err)
			}

			var got fuzzMessage
			if err := Unmarshal(wire, &got); err != nil {
				t.Fatal(err)
			}
			if !equalFuzzMessage(input, got) {
				t.Fatalf("slice round trip mismatch: %#v != %#v", input, got)
			}

			var streamed fuzzMessage
			reader := &fuzzOneByteReader{data: wire}
			if err := NewDecoder(reader).Decode(&streamed); err != nil {
				t.Fatal(err)
			}
			if !equalFuzzMessage(input, streamed) {
				t.Fatalf("stream round trip mismatch: %#v != %#v", input, streamed)
			}

			var buffer bytes.Buffer
			if err := MarshalTo(value, &buffer); err != nil {
				t.Fatal(err)
			}
			var fromWriter fuzzMessage
			if err := Unmarshal(buffer.Bytes(), &fromWriter); err != nil {
				t.Fatal(err)
			}
			if !equalFuzzMessage(input, fromWriter) {
				t.Fatalf("writer round trip mismatch: %#v != %#v", input, fromWriter)
			}
		}
	})
}

type fuzzCursor struct {
	data []byte
	off  int
}

func (c *fuzzCursor) next() byte {
	if len(c.data) == 0 {
		return 0
	}
	v := c.data[c.off%len(c.data)]
	c.off++
	return v
}

func (c *fuzzCursor) u64() uint64 {
	var b [8]byte
	for i := range b {
		b[i] = c.next()
	}
	return stdbinary.LittleEndian.Uint64(b[:])
}

func (c *fuzzCursor) count(max int) int {
	return int(c.next()) % (max + 1)
}

func (c *fuzzCursor) bytes(max int) []byte {
	data := make([]byte, c.count(max))
	for i := range data {
		data[i] = c.next()
	}
	return data
}

func (c *fuzzCursor) child() fuzzChild {
	return fuzzChild{ID: int32(c.u64()), Name: string(c.bytes(16)), Data: c.bytes(24)}
}

func fuzzMessageFrom(data []byte) fuzzMessage {
	c := &fuzzCursor{data: data}
	out := fuzzMessage{
		Bool:       c.next()&1 != 0,
		Int:        int64(c.u64()),
		Uint:       c.u64(),
		Float32:    math.Float32frombits(uint32(c.u64())),
		Float64:    math.Float64frombits(c.u64()),
		Complex64:  complex(math.Float32frombits(uint32(c.u64())), math.Float32frombits(uint32(c.u64()))),
		Complex128: complex(math.Float64frombits(c.u64()), math.Float64frombits(c.u64())),
		String:     string(c.bytes(32)),
		Bytes:      c.bytes(32),
	}

	out.Ints = make([]int64, c.count(8))
	for i := range out.Ints {
		out.Ints[i] = int64(c.u64())
	}
	out.Uints = make([]uint64, c.count(8))
	for i := range out.Uints {
		out.Uints[i] = c.u64()
	}
	out.Strings = make([]string, c.count(8))
	for i := range out.Strings {
		out.Strings[i] = string(c.bytes(16))
	}
	out.Floats = make([]float64, c.count(8))
	for i := range out.Floats {
		out.Floats[i] = math.Float64frombits(c.u64())
	}
	out.Complexes = make([]complex128, c.count(8))
	for i := range out.Complexes {
		out.Complexes[i] = complex(math.Float64frombits(c.u64()), math.Float64frombits(c.u64()))
	}

	if c.next()&1 != 0 {
		out.Child = new(fuzzChild)
		*out.Child = c.child()
	}
	out.Children = make([]*fuzzChild, c.count(8))
	for i := range out.Children {
		if c.next()&1 != 0 {
			out.Children[i] = new(fuzzChild)
			*out.Children[i] = c.child()
		}
	}
	for i := range out.Array {
		out.Array[i] = c.child()
	}

	valueCount := c.count(8)
	out.Values = make(map[string][]byte, valueCount)
	for i := 0; i < valueCount; i++ {
		out.Values[string(c.bytes(16))] = c.bytes(24)
	}
	countCount := c.count(8)
	out.Counts = make(map[uint64]uint64, countCount)
	for i := 0; i < countCount; i++ {
		out.Counts[c.u64()] = c.u64()
	}
	genericCount := c.count(8)
	out.Generic = make(map[int32]string, genericCount)
	for i := 0; i < genericCount; i++ {
		out.Generic[int32(c.u64())] = string(c.bytes(16))
	}

	switch c.next() % 3 {
	case 1:
		out.Variant.Text = &fuzzText{Msg: string(c.bytes(24))}
	case 2:
		out.Variant.Image = &fuzzImage{Width: int32(c.u64()), Height: int32(c.u64())}
	}
	out.Custom = fuzzCustom(c.bytes(32))
	return out
}

func fuzzBoundaryMessage() fuzzMessage {
	values := make(map[string][]byte, 8)
	counts := make(map[uint64]uint64, 8)
	generic := make(map[int32]string, 8)
	for i := 0; i < 8; i++ {
		key := string(rune('a' + i))
		values[key] = []byte{byte(i), 127, 128, 255}
		counts[uint64(i)] = ^uint64(i)
		generic[int32(i)] = key
	}
	return fuzzMessage{
		Int:        -1 << 63,
		Uint:       ^uint64(0),
		Float32:    math.Float32frombits(0x7fc00001),
		Float64:    math.Float64frombits(0x7ff8000000000001),
		Complex64:  complex(float32(math.Copysign(0, -1)), float32(math.Inf(1))),
		Complex128: complex(math.NaN(), math.Inf(-1)),
		String:     string([]byte{0, 127, 128, 255}),
		Bytes:      []byte{0, 127, 128, 255},
		Ints:       []int64{-1 << 63, -1, 0, 1, 1<<63 - 1},
		Uints:      []uint64{0, 1, 127, 128, ^uint64(0)},
		Strings:    []string{"", "a", string(bytes.Repeat([]byte{'x'}, 128)), "z", "y", "w", "v", "u"},
		Floats:     []float64{0, math.Copysign(0, -1), math.Inf(1), math.Inf(-1), math.NaN(), 1, -1, 2},
		Complexes:  []complex128{0, complex(math.NaN(), math.Inf(1))},
		Child:      &fuzzChild{ID: -1, Name: "child", Data: []byte{1, 2, 3}},
		Children:   []*fuzzChild{nil, {ID: 7, Name: "two"}},
		Array:      [2]fuzzChild{{ID: 1, Name: "one"}, {ID: 2, Name: "two"}},
		Values:     values,
		Counts:     counts,
		Generic:    generic,
		Variant:    fuzzVariant{Text: &fuzzText{Msg: "text"}},
		Custom:     fuzzCustom{0, 1, 127, 128, 255},
	}
}

type fuzzOneByteReader struct {
	data []byte
	off  int
}

func (r *fuzzOneByteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.off == len(r.data) {
		return 0, io.EOF
	}
	p[0] = r.data[r.off]
	r.off++
	return 1, nil
}

func equalFuzzMessage(a, b fuzzMessage) bool {
	if a.Bool != b.Bool || a.Int != b.Int || a.Uint != b.Uint ||
		math.Float32bits(a.Float32) != math.Float32bits(b.Float32) ||
		math.Float64bits(a.Float64) != math.Float64bits(b.Float64) ||
		!equalFuzzComplex64(a.Complex64, b.Complex64) ||
		!equalFuzzComplex128(a.Complex128, b.Complex128) ||
		a.String != b.String || !bytes.Equal(a.Bytes, b.Bytes) ||
		!equalFuzzInt64s(a.Ints, b.Ints) || !equalFuzzUint64s(a.Uints, b.Uints) ||
		!equalFuzzStrings(a.Strings, b.Strings) || !equalFuzzFloats(a.Floats, b.Floats) ||
		!equalFuzzComplexes(a.Complexes, b.Complexes) || !equalFuzzChildPtr(a.Child, b.Child) ||
		!equalFuzzChildren(a.Children, b.Children) || !equalFuzzMapBytes(a.Values, b.Values) ||
		!equalFuzzMapUint64(a.Counts, b.Counts) || !equalFuzzMapString(a.Generic, b.Generic) ||
		!equalFuzzVariant(a.Variant, b.Variant) || !bytes.Equal(a.Custom, b.Custom) {
		return false
	}
	for i := range a.Array {
		if !equalFuzzChild(a.Array[i], b.Array[i]) {
			return false
		}
	}
	return true
}

func equalFuzzChild(a, b fuzzChild) bool {
	return a.ID == b.ID && a.Name == b.Name && bytes.Equal(a.Data, b.Data)
}

func equalFuzzChildPtr(a, b *fuzzChild) bool {
	return (a == nil) == (b == nil) && (a == nil || equalFuzzChild(*a, *b))
}

func equalFuzzChildren(a, b []*fuzzChild) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalFuzzChildPtr(a[i], b[i]) {
			return false
		}
	}
	return true
}

func equalFuzzInt64s(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalFuzzUint64s(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalFuzzStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalFuzzFloats(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Float64bits(a[i]) != math.Float64bits(b[i]) {
			return false
		}
	}
	return true
}

func equalFuzzComplex64(a, b complex64) bool {
	return math.Float32bits(real(a)) == math.Float32bits(real(b)) &&
		math.Float32bits(imag(a)) == math.Float32bits(imag(b))
}

func equalFuzzComplex128(a, b complex128) bool {
	return math.Float64bits(real(a)) == math.Float64bits(real(b)) &&
		math.Float64bits(imag(a)) == math.Float64bits(imag(b))
}

func equalFuzzComplexes(a, b []complex128) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalFuzzComplex128(a[i], b[i]) {
			return false
		}
	}
	return true
}

func equalFuzzMapBytes(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		other, ok := b[key]
		if !ok || !bytes.Equal(value, other) {
			return false
		}
	}
	return true
}

func equalFuzzMapUint64(a, b map[uint64]uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func equalFuzzMapString(a, b map[int32]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func equalFuzzVariant(a, b fuzzVariant) bool {
	if (a.Text == nil) != (b.Text == nil) || (a.Image == nil) != (b.Image == nil) {
		return false
	}
	if a.Text != nil && a.Text.Msg != b.Text.Msg {
		return false
	}
	return a.Image == nil || (a.Image.Width == b.Image.Width && a.Image.Height == b.Image.Height)
}

func mustFuzzMarshal(value any) []byte {
	data, err := Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
