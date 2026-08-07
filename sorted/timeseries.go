// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
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

	// Write the timestamps into the buffer
	buffer := appendDelta(
		make([]byte, 0, 4*len(data.Time)),
		data.Time,
	)

	// Write the values into the buffer
	prev := uint64(0)
	for _, v := range data.Data {
		curr := uint64(bits.Reverse32(math.Float32bits(float32(v))))
		diff := curr ^ prev
		prev = curr
		buffer = appendUvarint(buffer, diff)
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

	offset, err := readDelta(result.Time, buffer)
	if err != nil {
		return err
	}

	// Read encoded values
	prev := uint64(0)
	for i := 0; i < int(count); i++ {
		var diff uint64
		for shift := uint(0); shift < 64; shift += 7 {
			if offset >= len(buffer) {
				return errInvalidVarint
			}
			b := buffer[offset]
			offset++
			if shift == 63 && b > 1 {
				return errInvalidVarint
			}
			diff |= uint64(b&0x7f) << shift
			if b < 0x80 {
				goto value
			}
		}
	value:
		prev ^= diff
		result.Data[i] = float64(math.Float32frombits(bits.Reverse32(uint32(prev))))
	}

	rv.Set(reflect.ValueOf(result))
	return nil
}

// ------------------------------------------------------------------------------

// appendDelta appends a delta array into the buffer
func appendDelta(dst []byte, data []uint64) []byte {
	prev := uint64(0)
	for i := range data {
		diff := data[i] - prev
		prev = data[i]
		dst = appendUvarint(dst, diff)
	}

	return dst
}

func appendUvarint(dst []byte, value uint64) []byte {
	for value >= 0x80 {
		dst = append(dst, byte(value)|0x80)
		value >>= 7
	}
	return append(dst, byte(value))
}

// readDelta reads a delta array from the buffer
func readDelta(dst []uint64, src []byte) (read int, err error) {
	prev := uint64(0)
	for i := range dst {
		var diff uint64
		for shift := uint(0); shift < 64; shift += 7 {
			if read >= len(src) {
				return read, errInvalidVarint
			}
			b := src[read]
			read++
			if shift == 63 && b > 1 {
				return read, errInvalidVarint
			}
			diff |= uint64(b&0x7f) << shift
			if b < 0x80 {
				prev += diff
				dst[i] = prev
				goto next
			}
		}
	next:
	}

	return read, nil
}
