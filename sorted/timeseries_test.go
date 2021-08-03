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
}

func makeTimeSeries(count int) *TimeSeries {
	var ts TimeSeries
	for i := count - 1; i >= 0; i-- {
		ts.Append(uint64(1500000000+i), float64(i))
	}
	return &ts
}
