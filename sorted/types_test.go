// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	"bytes"
	stdbinary "encoding/binary"
	"math"
	"testing"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
)

func TestTypes(t *testing.T) {
	tests := map[string]struct {
		value any
		out   any
	}{
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
		"timestamps": {
			value: Timestamps{4, 5, 6, 1, 2, 3},
			out:   new(Timestamps),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			b, err := binary.Marshal(tc.value)
			assert.NoError(t, err)
			assert.NotNil(t, b)
			again, err := binary.Marshal(tc.value)
			assert.NoError(t, err)
			assert.Equal(t, b, again)
			for range 2 {
				assert.NoError(t, binary.Unmarshal(b, tc.out))
				assert.Equal(t, tc.value, deref(tc.out))
			}
		})
	}
}

func TestDecodeEmptySlices(t *testing.T) {
	ints := Int32s{1}
	assert.NoError(t, binary.Unmarshal([]byte{0}, &ints))
	assert.Empty(t, ints)

	uints := Uint32s{1}
	assert.NoError(t, binary.Unmarshal([]byte{0}, &uints))
	assert.Empty(t, uints)
}

func deref(v any) any {
	switch x := v.(type) {
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
	case *Timestamps:
		return *x
	default:
		return v
	}
}

func FuzzDecodeWire(f *testing.F) {
	seeds := []struct {
		kind  byte
		value any
	}{
		{0, Uint16s{4, 1, 3, 2}},
		{1, Int16s{4, -1, 3, 2}},
		{2, Uint32s{4, 1, 3, 2}},
		{3, Int32s{4, -1, 3, 2}},
		{4, Uint64s{4, 1, 3, 2}},
		{5, Int64s{4, -1, 3, 2}},
		{6, Timestamps{4, 1, 3, 2}},
		{7, TimeSeries{Time: []uint64{4, 1, 3, 2}, Data: []float64{4, 1, 3, 2}}},
		{8, TimeCounters{Time: []uint64{4, 1, 3, 2}, Data: []uint64{4, 1, 3, 2}}},
	}
	for _, seed := range seeds {
		f.Add(seed.kind, mustFuzzMarshal(seed.value))
	}
	f.Add(byte(0), []byte(nil))
	f.Add(byte(7), []byte{1, 1, 0})

	f.Fuzz(func(t *testing.T, kind byte, wire []byte) {
		if len(wire) > 64<<10 {
			return
		}
		out := fuzzOutput(kind)
		if err := binary.Unmarshal(wire, out); err == nil {
			if _, err := binary.Marshal(out); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func FuzzRoundTrip(f *testing.F) {
	for kind := byte(0); kind < 9; kind++ {
		f.Add(kind, []byte(nil))
		f.Add(kind, bytes.Repeat([]byte{0xff}, 128))
		f.Add(kind, bytes.Repeat([]byte{0x01}, 128))
	}

	f.Fuzz(func(t *testing.T, kind byte, data []byte) {
		switch kind % 9 {
		case 0:
			checkSortedRoundTrip(t, Uint16s(fuzzSortedValues[uint16](data)), new(Uint16s))
		case 1:
			checkSortedRoundTrip(t, Int16s(fuzzSortedValues[int16](data)), new(Int16s))
		case 2:
			checkSortedRoundTrip(t, Uint32s(fuzzSortedValues[uint32](data)), new(Uint32s))
		case 3:
			checkSortedRoundTrip(t, Int32s(fuzzSortedValues[int32](data)), new(Int32s))
		case 4:
			checkSortedRoundTrip(t, Uint64s(fuzzSortedValues[uint64](data)), new(Uint64s))
		case 5:
			checkSortedRoundTrip(t, Int64s(fuzzSortedValues[int64](data)), new(Int64s))
		case 6:
			checkSortedRoundTrip(t, Timestamps(fuzzSortedValues[uint64](data)), new(Timestamps))
		case 7:
			checkSortedRoundTrip(t, fuzzTimeSeries(data), new(TimeSeries))
		case 8:
			checkSortedRoundTrip(t, fuzzTimeCounters(data), new(TimeCounters))
		}
	})
}

func fuzzOutput(kind byte) any {
	switch kind % 9 {
	case 0:
		return new(Uint16s)
	case 1:
		return new(Int16s)
	case 2:
		return new(Uint32s)
	case 3:
		return new(Int32s)
	case 4:
		return new(Uint64s)
	case 5:
		return new(Int64s)
	case 6:
		return new(Timestamps)
	case 7:
		return new(TimeSeries)
	default:
		return new(TimeCounters)
	}
}

func checkSortedRoundTrip(t *testing.T, value, out any) {
	t.Helper()
	wire, err := binary.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := binary.Unmarshal(wire, out); err != nil {
		t.Fatal(err)
	}
	reencoded, err := binary.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wire, reencoded) {
		t.Fatalf("non-canonical round trip: %x != %x", wire, reencoded)
	}
}

type sortedCursor struct {
	data []byte
	off  int
}

func (c *sortedCursor) u64() uint64 {
	var b [8]byte
	for i := range b {
		if len(c.data) > 0 {
			b[i] = c.data[c.off%len(c.data)]
			c.off++
		}
	}
	return stdbinary.LittleEndian.Uint64(b[:])
}

func fuzzSortedValues[T interface {
	~int16 | ~int32 | ~int64 | ~uint16 | ~uint32 | ~uint64
}](data []byte) []T {
	n := len(data) / 8
	if n > 16 {
		n = 16
	}
	values := make([]T, n)
	c := &sortedCursor{data: data}
	for i := range values {
		values[i] = T(c.u64())
	}
	return values
}

func fuzzTimeSeries(data []byte) TimeSeries {
	n := len(data) / 16
	if n > 16 {
		n = 16
	}
	series := TimeSeries{Time: make([]uint64, n), Data: make([]float64, n)}
	c := &sortedCursor{data: data}
	for i := 0; i < n; i++ {
		series.Time[i] = c.u64()
		series.Data[i] = math.Float64frombits(c.u64())
	}
	return series
}

func fuzzTimeCounters(data []byte) TimeCounters {
	n := len(data) / 16
	if n > 16 {
		n = 16
	}
	counters := TimeCounters{Time: make([]uint64, n), Data: make([]uint64, n)}
	c := &sortedCursor{data: data}
	for i := 0; i < n; i++ {
		counters.Time[i] = c.u64()
		counters.Data[i] = c.u64()
	}
	return counters
}

func mustFuzzMarshal(value any) []byte {
	data, err := binary.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
