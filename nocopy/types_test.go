// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package nocopy

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
)

func TestJSON(t *testing.T) {
	input := []byte(`{"answer":42}`)
	var value JSON
	assert.NoError(t, json.Unmarshal(input, &value))
	input[2] = 'x'
	assert.JSONEq(t, `{"answer":42}`, string(value))

	output, err := json.Marshal(struct {
		Value JSON `json:"value"`
	}{Value: value})
	assert.NoError(t, err)
	assert.JSONEq(t, `{"value":{"answer":42}}`, string(output))

	encoded, err := binary.Marshal(value)
	assert.NoError(t, err)
	var decoded JSON
	assert.NoError(t, binary.Unmarshal(encoded, &decoded))
	encoded[len(encoded)-1] = '!'
	assert.Equal(t, byte('!'), decoded[len(decoded)-1])
}

func TestStreamDecodeRetainsStrings(t *testing.T) {
	type pair struct {
		First  String
		Second String
	}

	want := pair{First: "first", Second: "second"}
	data, err := binary.Marshal(want)
	assert.NoError(t, err)

	var got pair
	assert.NoError(t, binary.NewDecoder(bytes.NewReader(data)).Decode(&got))
	assert.Equal(t, want, got)
}

type composite map[string]column

type column struct {
	Varchar columnVarchar
	Float64 columnFloat64
	Float32 columnFloat32
}

type columnVarchar struct {
	Nulls Bools
	Sizes Uint32s
	Bytes Bytes
}

type columnFloat64 struct {
	Nulls  Bools
	Floats Float64s
}

type columnFloat32 struct {
	Nulls  Bools
	Floats Float32s
}

type nested struct {
	Numbers Uint64s
}

func TestTypes(t *testing.T) {
	tests := map[string]struct {
		value any
		out   any
	}{
		"composite": {
			value: composite{
				"a": column{
					Varchar: columnVarchar{
						Nulls: Bools{false, false, false, true, false},
						Sizes: Uint32s{2, 2, 2, 0, 2},
						Bytes: Bytes{10, 10, 10, 10, 10, 10, 10, 10},
					},
				},
				"b": column{
					Float64: columnFloat64{
						Nulls:  Bools{false, false, false, true, false},
						Floats: Float64s{1.1, 2.2, 3.3, 0, 4.4},
					},
				},
			},
			out: &composite{},
		},
		"dictionary": {
			value: Dictionary{"name": "Roman", "race": "human", "status": "happy"},
			out:   &Dictionary{},
		},
		"bytemap": {
			value: ByteMap{"name": []byte("Roman"), "race": []byte("human"), "status": []byte("happy")},
			out:   &ByteMap{},
		},
		"hashmap": {
			value: HashMap{1: []byte("Roman"), 2: []byte("human"), 3: []byte("happy")},
			out:   &HashMap{},
		},
		"string": {
			value: String("ABCD"),
			out:   new(String),
		},
		"bytes": {
			value: Bytes([]byte("ABCD")),
			out:   new(Bytes),
		},
		"json": {
			value: JSON(`{"answer":42}`),
			out:   new(JSON),
		},
		"bools": {
			value: Bools{true, false, true, true, false, false},
			out:   new(Bools),
		},
		"uint16": {
			value: Uint16s{4, 5, 6, 1, 2, 3},
			out:   new(Uint16s),
		},
		"int16": {
			value: Int16s{4, 5, 6, 1, 2, 3},
			out:   new(Int16s),
		},
		"uint32": {
			value: Uint32s{4, 5, 6, 1, 2, 3},
			out:   new(Uint32s),
		},
		"int32": {
			value: Int32s{4, 5, 6, 1, 2, 3},
			out:   new(Int32s),
		},
		"uint64": {
			value: Uint64s{4, 5, 6, 1, 2, 3},
			out:   new(Uint64s),
		},
		"int64": {
			value: Int64s{4, 5, 6, 1, 2, 3},
			out:   new(Int64s),
		},
		"float32": {
			value: Float32s{4.5, 5.01, 6.61, 1.12, 2.1, 3},
			out:   new(Float32s),
		},
		"float64": {
			value: Float64s{4.5, 5.01, 6.61, 1.12, 2.1, 3},
			out:   new(Float64s),
		},
		"nested uint64": {
			value: nested{Numbers: Uint64s{4, 5, 6, 1, 2, 3}},
			out:   &nested{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			b, err := binary.Marshal(tc.value)
			assert.NoError(t, err)
			assert.NotNil(t, b)
			assert.NoError(t, binary.Unmarshal(b, tc.out))
			assert.Equal(t, tc.value, deref(tc.out))
		})
	}
}

func deref(v any) any {
	switch x := v.(type) {
	case *composite:
		return *x
	case *Dictionary:
		return *x
	case *ByteMap:
		return *x
	case *HashMap:
		return *x
	case *String:
		return *x
	case *Bytes:
		return *x
	case *JSON:
		return *x
	case *Bools:
		return *x
	case *Uint16s:
		return *x
	case *Int16s:
		return *x
	case *Uint32s:
		return *x
	case *Int32s:
		return *x
	case *Uint64s:
		return *x
	case *Int64s:
		return *x
	case *Float32s:
		return *x
	case *Float64s:
		return *x
	case *nested:
		return *x
	default:
		return v
	}
}
