// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	bin "encoding/binary"
	"math"
	"math/bits"
	"reflect"
	"sort"

	"github.com/kelindar/binary"
)

// ------------------------------------------------------------------------------

type tszCodec struct{}

// EncodeTo encodes a value into the encoder.
func (tszCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	data := rv.Interface().(TimeSeries)
	if !sort.IsSorted(&data) {
		sort.Sort(&data)
	}

	buffer := make([]byte, 0, 4*len(data.Time))

	// Write the timestamps into the buffer
	prev := uint64(0)
	for _, curr := range data.Time {
		diff := curr - prev
		prev = curr
		buffer = appendDelta(buffer, diff)
	}

	// Write the values into the buffer
	prev = uint64(0)
	for _, v := range data.Data {
		curr := uint64(bits.Reverse32(math.Float32bits(float32(v))))
		diff := curr ^ prev
		prev = curr
		buffer = appendDelta(buffer, diff)
	}

	// Writhe the size and the buffer
	e.WriteUvarint(uint64(len(data.Time)))
	e.WriteUvarint(uint64(len(buffer)))
	e.Write(buffer)
	return
}

// DecodeTo decodes into a reflect value from the decoder.
func (tszCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) error {

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
	result := TimeSeries{
		Time: make([]uint64, count),
		Data: make([]float64, count),
	}

	// Current offset
	offset := 0

	// Read encoded timestamps
	prev := uint64(0)
	for i := 0; i < int(count); i++ {
		diff, n := bin.Uvarint(buffer[offset:])
		prev = prev + diff
		result.Time[i] = prev
		offset += n
	}

	d.ReadUvarint()

	// Read encoded values
	prev = uint64(0)
	for i := 0; i < int(count); i++ {
		diff, n := bin.Uvarint(buffer[offset:])
		prev = prev ^ diff
		result.Data[i] = float64(math.Float32frombits(bits.Reverse32(uint32(prev))))
		offset += n
	}

	rv.Set(reflect.ValueOf(result))
	return nil
}

// appendDelta appends a delta into the buffer
func appendDelta(buffer []byte, delta uint64) []byte {
	for delta >= 0x80 {
		buffer = append(buffer, byte(delta)|0x80)
		delta >>= 7
	}

	return append(buffer, byte(delta))
}
