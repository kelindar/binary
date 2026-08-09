// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package unsafe

import (
	"bytes"
	stdbinary "encoding/binary"
	"io"
	"sort"
	"testing"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
)

func TestTypes(t *testing.T) {
	tests := map[string]struct {
		value any
		out   any
	}{
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

func TestIntegerDecodeErrors(t *testing.T) {
	for _, out := range []any{
		new(Bools),
		new(Uint16s),
		new(Int16s),
		new(Uint32s),
		new(Int32s),
		new(Uint64s),
		new(Int64s),
	} {
		assert.Error(t, binary.Unmarshal(nil, out))
	}

	zero := make([]byte, 8)
	var empty Uint16s
	assert.NoError(t, binary.Unmarshal(zero, &empty))
	assert.Empty(t, empty)

	truncated := make([]byte, 8)
	stdbinary.LittleEndian.PutUint64(truncated, 1)
	for _, out := range []any{new(Bools), new(Uint16s), new(Int16s), new(Uint32s), new(Int32s), new(Uint64s), new(Int64s)} {
		assert.Error(t, binary.Unmarshal(truncated, out))
	}

	tooLarge := make([]byte, 8)
	stdbinary.LittleEndian.PutUint64(tooLarge, uint64(^uint(0)>>1))
	assert.Equal(t, io.ErrUnexpectedEOF, binary.Unmarshal(tooLarge, new(Uint16s)))

	data := append(truncated, bytes.Repeat([]byte{0}, 8)...)
	var values Uint64s
	assert.NoError(t, binary.Unmarshal(data, &values))
}

func TestSort(t *testing.T) {
	tests := map[string]struct {
		value sort.Interface
		want  sort.Interface
	}{
		"uint16":  {Uint16s{4, 1, 3, 2}, Uint16s{1, 2, 3, 4}},
		"int16":   {Int16s{4, 1, 3, 2}, Int16s{1, 2, 3, 4}},
		"uint32":  {Uint32s{4, 1, 3, 2}, Uint32s{1, 2, 3, 4}},
		"int32":   {Int32s{4, 1, 3, 2}, Int32s{1, 2, 3, 4}},
		"uint64":  {Uint64s{4, 1, 3, 2}, Uint64s{1, 2, 3, 4}},
		"int64":   {Int64s{4, 1, 3, 2}, Int64s{1, 2, 3, 4}},
		"float32": {Float32s{4, 1, 3, 2}, Float32s{1, 2, 3, 4}},
		"float64": {Float64s{4, 1, 3, 2}, Float64s{1, 2, 3, 4}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			sort.Sort(tc.value)
			assert.Equal(t, tc.want, tc.value)
		})
	}
}

func deref(v any) any {
	switch x := v.(type) {
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
	default:
		return v
	}
}
