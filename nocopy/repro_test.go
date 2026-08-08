//go:build repro

package nocopy

import (
	"testing"

	"github.com/kelindar/binary"
)

func TestDecode(t *testing.T) {
	tests := map[string]func(*testing.T){
		"empty bytes":            testEmptyBytes,
		"empty byte map value":   testEmptyByteMapValue,
		"empty hash map value":   testEmptyHashMapValue,
		"truncated dictionary":   testDictionaryTruncation,
		"truncated byte map key": testByteMapKeyTruncation,
		"truncated hash map key": testHashMapKeyTruncation,
	}
	for name, test := range tests {
		t.Run(name, test)
	}
}

func testEmptyBytes(t *testing.T) {
	data, err := binary.Marshal(reproEmptyBytes)
	if err != nil {
		t.Fatal(err)
	}

	got := Bytes("stale")
	if err := binary.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("empty decode retained stale bytes: %q", got)
	}
}

func testEmptyByteMapValue(t *testing.T) {
	data, err := binary.Marshal(ByteMap{"empty": {}})
	if err != nil {
		t.Fatal(err)
	}

	var got ByteMap
	if err := binary.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["empty"]; !ok {
		t.Fatal("decoding an empty map value dropped its key")
	}
}

func testEmptyHashMapValue(t *testing.T) {
	data, err := binary.Marshal(HashMap{1: {}})
	if err != nil {
		t.Fatal(err)
	}

	var got HashMap
	if err := binary.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got[1]; !ok {
		t.Fatal("decoding an empty hash map value dropped its key")
	}
}

func testDictionaryTruncation(t *testing.T) {
	var got Dictionary
	if err := binary.Unmarshal([]byte{1, 0}, &got); err == nil {
		t.Fatal("truncated dictionary decoded without an error")
	}
}

func testByteMapKeyTruncation(t *testing.T) {
	var got ByteMap
	if err := binary.Unmarshal([]byte{1, 0, 2, 0}, &got); err == nil {
		t.Fatal("truncated byte map key decoded without an error")
	}
}

func testHashMapKeyTruncation(t *testing.T) {
	var got HashMap
	if err := binary.Unmarshal([]byte{1, 0, 0, 0, 0, 0, 0, 0}, &got); err == nil {
		t.Fatal("truncated hash map key decoded without an error")
	}
}
