// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	"bytes"
	stdbinary "encoding/binary"
	"reflect"
	"testing"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
)

type testInt8s []int8

func (s testInt8s) Len() int           { return len(s) }
func (s testInt8s) Less(i, j int) bool { return s[i] < s[j] }
func (s testInt8s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type testUint8s []uint8

func (s testUint8s) Len() int           { return len(s) }
func (s testUint8s) Less(i, j int) bool { return s[i] < s[j] }
func (s testUint8s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func TestPayload(t *testing.T) {
	encoded := []byte{0x8, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2}

	v := Int32s{1, 2, 3, 4, 5, 6, 7, 8}
	ev, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.Equal(t, encoded, ev)
}

func TestFixedWidthCodecs(t *testing.T) {
	tests := []struct {
		name  string
		codec binary.Codec
		value any
	}{
		{"int8", IntsCodecAs(reflect.TypeFor[testInt8s](), 1), testInt8s{2, -1, 1}},
		{"uint8", UintsCodecAs(reflect.TypeFor[testUint8s](), 1), testUint8s{2, 0, 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value := reflect.ValueOf(tc.value)
			var first bytes.Buffer
			assert.NoError(t, tc.codec.EncodeTo(binary.NewEncoder(&first), value))

			var second bytes.Buffer
			assert.NoError(t, tc.codec.EncodeTo(binary.NewEncoder(&second), value))
			assert.Equal(t, first.Bytes(), second.Bytes())

			out := reflect.New(value.Type()).Elem()
			assert.NoError(t, tc.codec.DecodeTo(binary.NewDecoder(bytes.NewBuffer(first.Bytes())), out))
			assert.Equal(t, value.Interface(), out.Interface())
		})
	}
}

func TestInvalidVarint(t *testing.T) {
	for name, out := range map[string]any{
		"int":  new(Int32s),
		"uint": new(Uint32s),
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, errInvalidVarint, binary.Unmarshal([]byte{1, 0x80}, out))
		})
	}
}

func TestInvalidCompressedPayload(t *testing.T) {
	overflow := append([]byte{1, 10}, bytes.Repeat([]byte{0x80}, 10)...)
	valueOverflow := append([]byte{1, 11, 0}, bytes.Repeat([]byte{0x80}, 10)...)
	tests := []struct {
		name string
		out  any
		data []byte
	}{
		{"timestamp EOF", new(Timestamps), []byte{1, 0}},
		{"timestamp overflow", new(Timestamps), overflow},
		{"series timestamp EOF", new(TimeSeries), []byte{1, 0}},
		{"series value EOF", new(TimeSeries), []byte{1, 1, 0}},
		{"series value overflow", new(TimeSeries), valueOverflow},
		{"counters timestamp EOF", new(TimeCounters), []byte{1, 0}},
		{"counters value EOF", new(TimeCounters), []byte{1, 1, 0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, errInvalidVarint, binary.Unmarshal(tc.data, tc.out))
		})
	}
}

func TestDecodeShort(t *testing.T) {
	tests := map[string]any{
		"timestamps":    new(Timestamps),
		"time series":   new(TimeSeries),
		"time counters": new(TimeCounters),
	}
	data := map[string][]byte{
		"missing count":   {},
		"missing size":    {1},
		"empty payload":   {1, 0},
		"missing payload": {1, 1},
	}

	for name, out := range tests {
		t.Run(name, func(t *testing.T) {
			for name, input := range data {
				t.Run(name, func(t *testing.T) {
					assert.Error(t, binary.Unmarshal(input, out))
				})
			}
		})
	}
}

func TestLength(t *testing.T) {
	data := stdbinary.AppendUvarint(nil, ^uint64(0))
	tests := map[string]struct {
		data []byte
		out  any
	}{
		"delta slice":   {data: data, out: new(Int32s)},
		"timestamps":    {data: append(append([]byte{}, data...), 0), out: new(Timestamps)},
		"time series":   {data: append(append([]byte{}, data...), 0), out: new(TimeSeries)},
		"time counters": {data: append(append([]byte{}, data...), 0), out: new(TimeCounters)},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("malformed length panicked: %v", recovered)
				}
			}()
			if err := binary.Unmarshal(tc.data, tc.out); err == nil {
				t.Fatal("expected malformed length to return an error")
			}
		})
	}
}

func TestSeriesBounds(t *testing.T) {
	tests := map[string]any{
		"time series":   TimeSeries{Time: []uint64{2, 1}, Data: []float64{1}},
		"time counters": TimeCounters{Time: []uint64{2, 1}, Data: []uint64{1}},
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("mismatched series panicked: %v", recovered)
				}
			}()
			if _, err := binary.Marshal(input); err == nil {
				t.Fatal("expected mismatched series to return an error")
			}
		})
	}
}
