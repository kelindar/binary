//go:build repro

package nocopy

import (
	"testing"

	"github.com/kelindar/binary"
)

func TestEmptyDecode(t *testing.T) {
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
