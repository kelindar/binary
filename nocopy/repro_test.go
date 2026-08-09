//go:build repro

package nocopy

import (
	"bytes"
	stdbinary "encoding/binary"
	"testing"

	"github.com/kelindar/binary"
)

func TestDecode(t *testing.T) {
	tests := map[string]func(*testing.T){
		"empty bytes":               testEmptyBytes,
		"misaligned integer length": testMisalignedIntegerLength,
		"empty byte map value":      testEmptyByteMapValue,
		"empty hash map value":      testEmptyHashMapValue,
		"byte map value capacity":   testByteMapValueCapacity,
		"truncated dictionary":      testDictionaryTruncation,
		"truncated byte map key":    testByteMapKeyTruncation,
		"truncated hash map key":    testHashMapKeyTruncation,
	}
	for name, test := range tests {
		t.Run(name, test)
	}
}

func testByteMapValueCapacity(t *testing.T) {
	data := []byte{2, 0, 1, 'a', 1, 'x', 1, 'b', 1, 'y'}
	var got ByteMap
	if err := binary.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	sibling := got["b"]
	_ = append(got["a"], 'z', 'q', 'r', 's')
	if !bytes.Equal(sibling, []byte("y")) {
		t.Fatalf("appending to one no-copy value changed its sibling: got=%q", sibling)
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

func testMisalignedIntegerLength(t *testing.T) {
	data := []byte{1, 0, 0, 0, 0, 0, 0, 0, 1}
	var got Uint16s
	if err := binary.Unmarshal(data, &got); err == nil {
		t.Fatal("non-element-aligned length decoded without an error")
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

func TestLargeMap(t *testing.T) {
	data := make([]byte, 4)
	stdbinary.LittleEndian.PutUint32(data, ^uint32(0))
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("large map count panicked: %v", recovered)
		}
	}()

	var got HashMap
	if err := binary.Unmarshal(data, &got); err == nil {
		t.Fatal("expected large truncated map count to return an error")
	}
}

func TestMapBounds(t *testing.T) {
	bytesMap := make(ByteMap, 1<<16)
	dictionary := make(Dictionary, 1<<16)
	for i := 0; i < 1<<16; i++ {
		key := string([]byte{byte(i), byte(i >> 8)})
		bytesMap[key] = nil
		dictionary[key] = ""
	}
	for name, input := range map[string]any{
		"byte map":   bytesMap,
		"dictionary": dictionary,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := binary.Marshal(input); err == nil {
				t.Fatal("expected oversized map to be rejected")
			}
		})
	}
}

func TestMapWriterNil(t *testing.T) {
	var writer *bytes.Buffer
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("typed-nil writer panicked in map fast path: %v", recovered)
		}
	}()
	if err := binary.MarshalTo(ByteMap{"key": []byte("value")}, writer); err == nil {
		t.Fatal("expected typed-nil writer to return an error")
	}
}
