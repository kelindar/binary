package main

import "testing"

func TestHelpers(t *testing.T) {
	t.Run("uint64s", func(t *testing.T) {
		got := makeUint64s(5)
		if len(got) != 5 || got[4] != 4 {
			t.Fatalf("makeUint64s(5) = %v", got)
		}
	})
	t.Run("bytes", func(t *testing.T) {
		got := makeBytes(5)
		if len(got) != 5 || got[4] != 4 {
			t.Fatalf("makeBytes(5) = %v", got)
		}
	})
}
