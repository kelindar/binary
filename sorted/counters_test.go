package sorted

import (
	stdbinary "encoding/binary"
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

func TestCountersDecode(t *testing.T) {
	input := TimeCounters{Time: []uint64{1, 3}, Data: []uint64{10, 20}}
	encoded, err := binary.Marshal(input)
	assert.NoError(t, err)
	got := TimeCounters{Time: make([]uint64, 0, 4), Data: make([]uint64, 0, 4)}
	assert.NoError(t, binary.Unmarshal(encoded, &got))
	assert.Equal(t, input, got)

	for _, data := range [][]byte{
		stdbinary.AppendUvarint(stdbinary.AppendUvarint(nil, 0), 1),
		append(stdbinary.AppendUvarint(stdbinary.AppendUvarint(nil, 1), 2), 0x80, 0x80),
	} {
		var out TimeCounters
		assert.Error(t, binary.Unmarshal(data, &out))
	}
}
