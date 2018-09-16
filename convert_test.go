// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvert_String(t *testing.T) {
	v := "hi there"

	b := stringToBinary(v)
	assert.NotEmpty(t, b)
	assert.Equal(t, v, string(b))

	o := binaryToString(&b)
	assert.NotEmpty(t, b)
	assert.Equal(t, v, o)
}

func TestConvert_Bools(t *testing.T) {
	v := []bool{true, false, true, true, false, false}

	b := boolsToBinary(&v)
	assert.NotEmpty(t, b)
	assert.Equal(t, []byte{0x1, 0x0, 0x1, 0x1, 0x0, 0x0}, b)

	o := binaryToBools(&b)
	assert.NotEmpty(t, b)
	assert.Equal(t, v, o)
}
