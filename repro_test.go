//go:build repro

package binary

import (
	"bytes"
	stdbinary "encoding/binary"
	"testing"
)

func TestArenaMap(t *testing.T) {
	data := []byte{
		2,
		1, 0, 'a', 1, 'x',
		1, 0, 'b', 1, 'y',
	}
	var got map[string][]byte
	if err := Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	sibling := got["b"]
	first := append(got["a"], 'z', 'q')
	if !bytes.Equal(sibling, []byte("y")) {
		t.Fatalf("appending to one decoded value changed its sibling: first=%q sibling=%q", first, sibling)
	}
}

func TestUnionArena(t *testing.T) {
	in := reproUnionMapEnvelope{
		Body: reproUnionMapContainer{
			Arm: &reproUnionMapArm{Values: map[string][]byte{"a": {'x'}}},
		},
		Tail: map[string][]byte{"t": bytes.Repeat([]byte{'y'}, 16)},
	}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("decoding a union followed by a map panicked: %v", recovered)
		}
	}()
	var got reproUnionMapEnvelope
	if err := Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
}

func TestHugeLength(t *testing.T) {
	data := stdbinary.AppendUvarint(nil, ^uint64(0))
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("malformed length panicked: %v", recovered)
		}
	}()

	var got []byte
	if err := Unmarshal(data, &got); err == nil {
		t.Fatal("expected malformed length to return an error")
	}
}
