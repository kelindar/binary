//go:build repro

package unsafe

import (
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
