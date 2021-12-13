// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"bytes"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCustom string

// GetBinaryCodec retrieves a custom binary codec.
func (s *testCustom) GetBinaryCodec() Codec {
	return new(stringCodec)
}

func TestScanner(t *testing.T) {
	rt := reflect.Indirect(reflect.ValueOf(s0v)).Type()
	codec, err := scan(rt)
	assert.NoError(t, err)
	assert.NotNil(t, codec)

	var b bytes.Buffer
	e := NewEncoder(&b)
	err = codec.EncodeTo(e, reflect.Indirect(reflect.ValueOf(s0v)))
	assert.NoError(t, err)
	assert.Equal(t, s0b, b.Bytes())
}

func TestScanner_Custom(t *testing.T) {
	v := testCustom("test")
	rt := reflect.Indirect(reflect.ValueOf(v)).Type()
	codec, err := scan(rt)
	assert.NoError(t, err)
	assert.NotNil(t, codec)
}

func TestScannerComposed(t *testing.T) {
	codec, err := scan(reflect.TypeOf(Partition{}))
	assert.NoError(t, err)
	assert.NotNil(t, codec)
}

type Partition struct {
	Strings
	Filters map[uint32][]uint64
}

type Strings struct {
	lock sync.Mutex `binary:"-"`
	Key  string
	Fill []uint64
	Hash []uint32
	Data map[uint64][]byte
}
