// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	"testing"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
)

func TestPayload(t *testing.T) {
	encoded := []byte{0x8, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2, 0x2}

	v := Int32s{1, 2, 3, 4, 5, 6, 7, 8}
	ev, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.Equal(t, encoded, ev)
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

func TestDecodeShort(t *testing.T) {
	tests := map[string]any{
		"timestamps":    new(Timestamps),
		"time series":   new(TimeSeries),
		"time counters": new(TimeCounters),
	}
	data := map[string][]byte{
		"missing count":   {},
		"missing size":    {1},
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
