package binary

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScanner(t *testing.T) {
	rt := reflect.Indirect(reflect.ValueOf(s0v)).Type()
	codec, err := scan(rt)
	assert.NoError(t, err)
	assert.NotNil(t, codec)

	var b bytes.Buffer
	e := NewEncoder(&b)
	err = codec.EncodeTo(e, reflect.Indirect(reflect.ValueOf(s0v)))
	assert.NoError(t, err)

	//e.Flush()
	assert.Equal(t, s0b, b.Bytes())
}
