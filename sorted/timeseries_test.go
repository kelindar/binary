// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	"testing"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
)

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

func TestNestedTimeSeries(t *testing.T) {
	type envelope struct {
		Series TimeSeries
		Value  uint64
	}
	want := envelope{Series: *makeTimeSeries(2), Value: 7}
	b, err := binary.Marshal(want)
	assert.NoError(t, err)

	var got envelope
	assert.NoError(t, binary.Unmarshal(b, &got))
	assert.Equal(t, want, got)
}

func makeTimeSeries(count int) *TimeSeries {
	var ts TimeSeries
	for i := count - 1; i >= 0; i-- {
		ts.Append(uint64(1500000000+i), float64(i))
	}
	return &ts
}
