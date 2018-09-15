// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReader_Slice(t *testing.T) {
	r := newReader([]byte("0123456789"))

	out, err := r.Slice(3)
	assert.NoError(t, err)
	assert.Len(t, out, 3)
	assert.Equal(t, "012", string(out))
}
