//go:build repro

package unsafe

import (
	stdbinary "encoding/binary"
	"testing"

	"github.com/kelindar/binary"
)

func TestEmptyDecode(t *testing.T) {
	data, err := binary.Marshal(reproEmptyIntegers)
	if err != nil {
		t.Fatal(err)
	}

	got := Uint16s{42}
	if err := binary.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("empty decode retained stale integers: %v", got)
	}
}

func TestHugeLength(t *testing.T) {
	data := make([]byte, 8)
	stdbinary.LittleEndian.PutUint64(data, ^uint64(0))
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("malformed length panicked: %v", recovered)
		}
	}()

	var got Uint16s
	if err := binary.Unmarshal(data, &got); err == nil {
		t.Fatal("expected malformed length to return an error")
	}
}
