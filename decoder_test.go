// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBinaryDecodeStruct(t *testing.T) {
	s := &s0{}
	err := Unmarshal(s0b, s)
	assert.NoError(t, err)
	assert.Equal(t, s0v, s)
}

func TestBinaryDecodeToValueErrors(t *testing.T) {
	b := []byte{1, 0, 0, 0}
	var v uint32
	err := Unmarshal(b, v)
	assert.Error(t, err)
	err = Unmarshal(b, &v)
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), v)
}

type oneByteReader struct {
	content []byte
}

// Read method of io.Reader reads *up to* len(buf) bytes.
// It is possible to read LESS, and it can happen when reading a file.
func (r *oneByteReader) Read(buf []byte) (n int, err error) {
	if len(r.content) == 0 {
		err = io.EOF
		return
	}

	if len(buf) == 0 {
		return
	}
	n = 1
	buf[0] = r.content[0]
	r.content = r.content[1:]
	return
}

func TestDecodeFromReader(t *testing.T) {
	data := "data string"
	encoded, err := Marshal(data)
	assert.NoError(t, err)
	decoder := NewDecoder(&oneByteReader{content: encoded})
	str, err := decoder.ReadString()
	assert.NoError(t, err)
	assert.Equal(t, data, str)
}
