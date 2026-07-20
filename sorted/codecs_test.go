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
