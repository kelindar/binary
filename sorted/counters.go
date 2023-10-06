package sorted

import (
	"reflect"
	"sort"

	"github.com/kelindar/binary"
)

// ------------------------------------------------------------------------------

type tczCodec struct{}

// EncodeTo encodes a value into the encoder.
func (tczCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	data := rv.Interface().(TimeCounters)
	if !sort.IsSorted(&data) {
		sort.Sort(&data)
	}

	buffer := make([]byte, 0, 4*len(data.Time))
	buffer = appendDelta(buffer, data.Time)
	buffer = appendDelta(buffer, data.Data)

	// Writhe the size and the buffer
	e.WriteUvarint(uint64(len(data.Time)))
	e.WriteUvarint(uint64(len(buffer)))
	e.Write(buffer)
	return
}

// DecodeTo decodes into a reflect value from the decoder.
func (tczCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) error {

	// Read the number of timestamps
	count, err := d.ReadUvarint()
	if err != nil {
		return err
	}

	// Read the size in bytes
	size, err := d.ReadUvarint()
	if err != nil {
		return err
	}

	// Read the timestamp buffer
	buffer, err := d.Slice(int(size))
	if err != nil {
		return err
	}

	// Read the timestamps
	result := TimeCounters{
		Time: make([]uint64, count),
		Data: make([]uint64, count),
	}

	// Current offset
	offset := readDelta(result.Time, buffer[0:])
	offset = readDelta(result.Data, buffer[offset:])

	rv.Set(reflect.ValueOf(result))
	return nil
}
