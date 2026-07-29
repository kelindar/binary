// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"bytes"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---- fixtures ----------------------------------------------------------------

type textPayload struct {
	Msg string
}

type imagePayload struct {
	Width  int
	Height int
}

type payload struct {
	Text  *textPayload  `binary:"1"`
	Image *imagePayload `binary:"2"`
}

type docV1 struct {
	Title string
}

type docV2 struct {
	Title string
	Body  string
}

type doc struct {
	V1 *docV1 `binary:"1"`
	V2 *docV2 `binary:"2"`
}

type envelope struct {
	ID   uint64
	Body payload
}

type payloadWithSkip struct {
	mu    sync.Mutex    `binary:"-"`
	Text  *textPayload  `binary:"1"`
	Image *imagePayload `binary:"2"`
}

// ---- TestUnion ---------------------------------------------------------------

func TestUnion(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		in := payload{Text: &textPayload{Msg: "hi"}}
		b, err := Marshal(in)
		assert.NoError(t, err)

		var out payload
		assert.NoError(t, Unmarshal(b, &out))
		assert.Equal(t, in.Text, out.Text)
		assert.Nil(t, out.Image)
	})

	t.Run("none", func(t *testing.T) {
		b, err := Marshal(payload{})
		assert.NoError(t, err)
		assert.Equal(t, []byte{0x0, 0x0}, b) // tag 0, len 0

		var out payload
		assert.NoError(t, Unmarshal(b, &out))
		assert.Nil(t, out.Text)
		assert.Nil(t, out.Image)
	})

	t.Run("multiple arms", func(t *testing.T) {
		_, err := Marshal(payload{
			Text:  &textPayload{Msg: "a"},
			Image: &imagePayload{Width: 1, Height: 2},
		})
		assert.True(t, errors.Is(err, ErrMultipleArms))
	})

	t.Run("unknown tag skipped", func(t *testing.T) {
		// Simulate a future arm 3 (string wire: uvarint len + byte).
		var buf bytes.Buffer
		e := NewEncoder(&buf)
		e.WriteTagged(3, []byte{0x1, 'x'})
		assert.NoError(t, e.err)

		var out payload
		assert.NoError(t, Unmarshal(buf.Bytes(), &out))
		assert.Nil(t, out.Text)
		assert.Nil(t, out.Image)
	})

	t.Run("clears previous arm", func(t *testing.T) {
		out := payload{Text: &textPayload{Msg: "old"}}
		b, err := Marshal(payload{Image: &imagePayload{Width: 3, Height: 4}})
		assert.NoError(t, err)
		assert.NoError(t, Unmarshal(b, &out))
		assert.Nil(t, out.Text)
		assert.Equal(t, &imagePayload{Width: 3, Height: 4}, out.Image)
	})

	t.Run("nested", func(t *testing.T) {
		in := envelope{ID: 42, Body: payload{Image: &imagePayload{Width: 10, Height: 20}}}
		b, err := Marshal(in)
		assert.NoError(t, err)

		var out envelope
		assert.NoError(t, Unmarshal(b, &out))
		assert.Equal(t, in, out)
	})

	t.Run("skip tag", func(t *testing.T) {
		in := payloadWithSkip{Text: &textPayload{Msg: "ok"}}
		b, err := Marshal(in)
		assert.NoError(t, err)

		var out payloadWithSkip
		assert.NoError(t, Unmarshal(b, &out))
		assert.Equal(t, in.Text, out.Text)
	})

	t.Run("versioning", func(t *testing.T) {
		v1, err := Marshal(doc{V1: &docV1{Title: "a"}})
		assert.NoError(t, err)
		v2, err := Marshal(doc{V2: &docV2{Title: "a", Body: "b"}})
		assert.NoError(t, err)

		var got doc
		assert.NoError(t, Unmarshal(v1, &got))
		assert.Equal(t, &docV1{Title: "a"}, got.V1)
		assert.Nil(t, got.V2)

		got = doc{}
		assert.NoError(t, Unmarshal(v2, &got))
		assert.Nil(t, got.V1)
		assert.Equal(t, &docV2{Title: "a", Body: "b"}, got.V2)
	})

	t.Run("cached codec", func(t *testing.T) {
		rt := reflect.TypeFor[payload]()
		c1, err := scan(rt)
		assert.NoError(t, err)
		c2, err := scan(rt)
		assert.NoError(t, err)
		assert.True(t, c1 == c2)
		_, ok := c1.(*reflectUnionCodec)
		assert.True(t, ok)
	})
}

// ---- TestUnionScan -----------------------------------------------------------

func TestUnionScan(t *testing.T) {
	tests := map[string]interface{}{
		"mixed fields": func() interface{} {
			type mixed struct {
				ID   uint64
				Text *textPayload `binary:"1"`
			}
			return mixed{}
		}(),
		"non-pointer arm": func() interface{} {
			type nonPtr struct {
				Text textPayload `binary:"1"`
			}
			return nonPtr{}
		}(),
		"duplicate tag": func() interface{} {
			type dup struct {
				A *textPayload  `binary:"1"`
				B *imagePayload `binary:"1"`
			}
			return dup{}
		}(),
		"bad tag string": func() interface{} {
			type badTag struct {
				A *textPayload `binary:"nope"`
			}
			return badTag{}
		}(),
		"tag zero": func() interface{} {
			type zeroTag struct {
				A *textPayload `binary:"0"`
			}
			return zeroTag{}
		}(),
		"tag out of range": func() interface{} {
			type tooBig struct {
				A *textPayload `binary:"256"`
			}
			return tooBig{}
		}(),
	}
	for name, v := range tests {
		t.Run(name, func(t *testing.T) {
			rt := reflect.TypeOf(v)
			_, err := scanType(rt)
			assert.Error(t, err)
		})
	}
}

// ---- TestTagged --------------------------------------------------------------

func TestTagged(t *testing.T) {
	var buf bytes.Buffer
	e := NewEncoder(&buf)
	e.WriteTagged(7, []byte("abc"))
	assert.NoError(t, e.err)

	d := NewDecoder(bytes.NewReader(buf.Bytes()))
	tag, body, err := d.ReadTagged()
	assert.NoError(t, err)
	assert.Equal(t, uint64(7), tag)
	assert.Equal(t, []byte("abc"), body)
}
