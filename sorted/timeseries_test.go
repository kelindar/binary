// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	"testing"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
)

/*
BenchmarkTimeSeries/encode-10         	   10000	    103506 ns/op	  123136 B/op	       6 allocs/op
BenchmarkTimeSeries/decode-10         	   13610	     85692 ns/op	  661022 B/op	      22 allocs/op
*/
func BenchmarkTimeSeries(b *testing.B) {
	series := makeTimeCounters(20000)
	enc, _ := binary.Marshal(&series)

	b.Run("encode", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&series)
		}
	})

	b.Run("decode", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out Timestamps
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}

func TestTimeSeries(t *testing.T) {

	// Marshal
	ts := makeTimeSeries(100)
	b, err := binary.Marshal(ts)
	assert.NoError(t, err)
	assert.Equal(t, 341, len(b)) // Consider compressing using snappy after

	// Unmarshal
	var out TimeSeries
	assert.NoError(t, binary.Unmarshal(b, &out))
	assert.Equal(t, 100, len(out.Data))
	assert.Equal(t, *ts, out)
}

func makeTimeSeries(count int) *TimeSeries {
	var ts TimeSeries
	for i := count - 1; i >= 0; i-- {
		ts.Append(uint64(1500000000+i), float64(i))
	}
	return &ts
}
