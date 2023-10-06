package sorted

import (
	"testing"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
)

/*
BenchmarkTimeCounters/encode-10         	   10000	    105358 ns/op	  123137 B/op	       6 allocs/op
BenchmarkTimeCounters/decode-10         	   14527	     83441 ns/op	  661026 B/op	      22 allocs/op
*/
func BenchmarkTimeCounters(b *testing.B) {
	series := makeTimeCounters(20000)
	enc, _ := binary.Marshal(&series)

	b.Run("encode", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			binary.Marshal(&series)
		}
	})

	b.Run("decode", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		var out Timestamps
		for n := 0; n < b.N; n++ {
			binary.Unmarshal(enc, &out)
		}
	})
}

func TestTimeCounters(t *testing.T) {

	// Marshal
	ts := makeTimeCounters(100)
	b, err := binary.Marshal(ts)
	assert.NoError(t, err)
	assert.Equal(t, 207, len(b)) // Consider compressing using snappy after

	// Unmarshal
	var out TimeCounters
	assert.NoError(t, binary.Unmarshal(b, &out))
	assert.Equal(t, 100, len(out.Data))
	assert.Equal(t, *ts, out)
}

func makeTimeCounters(count int) *TimeCounters {
	var ts TimeCounters
	for i := count - 1; i >= 0; i-- {
		ts.Append(uint64(1500000000+i), uint64(i))
	}
	return &ts
}
