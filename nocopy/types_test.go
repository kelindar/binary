// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package nocopy

import (
	"testing"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
)

func Test_String(t *testing.T) {
	v := String("ABCD")

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o String
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Bytes(t *testing.T) {
	v := Bytes([]byte("ABCD"))

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Bytes
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Bools(t *testing.T) {
	v := Bools{true, false, true, true, false, false}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Bools
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Uint16(t *testing.T) {
	v := Uint16s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Uint16s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Int16(t *testing.T) {
	v := Int16s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Int16s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Uint32(t *testing.T) {
	v := Uint32s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Uint32s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Int32(t *testing.T) {
	v := Int32s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Int32s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Uint64(t *testing.T) {
	v := Uint64s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Uint64s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Int64(t *testing.T) {
	v := Int64s{4, 5, 6, 1, 2, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Int64s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Float32(t *testing.T) {
	v := Float32s{4.5, 5.01, 6.61, 1.12, 2.1, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Float32s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}

func Test_Float64(t *testing.T) {
	v := Float64s{4.5, 5.01, 6.61, 1.12, 2.1, 3}

	b, err := binary.Marshal(&v)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	var o Float64s
	err = binary.Unmarshal(b, &o)
	assert.NoError(t, err)
	assert.Equal(t, v, o)
}
