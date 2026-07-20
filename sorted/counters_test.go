package sorted

import (
	"testing"

	"github.com/kelindar/binary"
	"github.com/stretchr/testify/assert"
)

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
