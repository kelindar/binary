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

type tszCodec struct{}

func (tszCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	data := rv.Interface().(TimeSeries)
	if !isSorted(data.Time) {
		sort.Sort(&data)
	}
	buffer := appendDelta(
		make([]byte, 0, 4*len(data.Time)),
		data.Time,
	)
	prev := uint64(0)
	for _, v := range data.Data {
		curr := uint64(bits.Reverse32(math.Float32bits(float32(v))))
		diff := curr ^ prev
		prev = curr
		buffer = bin.AppendUvarint(buffer, diff)
	}
	e.WriteUvarint(uint64(len(data.Time)))
	e.WriteUvarint(uint64(len(buffer)))
	e.Write(buffer)
	return
}
func (tszCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) error {
	count, err := d.ReadUvarint()
	if err != nil {
		return err
	}
	size, err := d.ReadUvarint()
	if err != nil {
		return err
	}
	buffer, err := d.Slice(int(size))
	if err != nil {
		return err
	}
	n := int(count)
	result := rv.Interface().(TimeSeries)
	if result.Time == nil || cap(result.Time) < n {
		result.Time = make([]uint64, n)
	} else {
		result.Time = result.Time[:n]
	}
	if result.Data == nil || cap(result.Data) < n {
		result.Data = make([]float64, n)
	} else {
		result.Data = result.Data[:n]
	}
	offset, err := readDelta(result.Time, buffer)
	if err != nil {
		return err
	}
	prev := uint64(0)
	for i := 0; i < n; i++ {
		diff, n := bin.Uvarint(buffer[offset:])
		if n <= 0 {
			return errInvalidVarint
		}
		offset += n
		prev ^= diff
		result.Data[i] = float64(math.Float32frombits(bits.Reverse32(uint32(prev))))
	}
	rv.Set(reflect.ValueOf(result))
	return nil
}

func appendDelta(dst []byte, data []uint64) []byte {
	prev := uint64(0)
	for i := range data {
		diff := data[i] - prev
		prev = data[i]
		dst = bin.AppendUvarint(dst, diff)
	}
	return dst
}

func readDelta(dst []uint64, src []byte) (read int, err error) {
	prev := uint64(0)
	for i := range dst {
		diff, n := bin.Uvarint(src[read:])
		if n <= 0 {
			return read, errInvalidVarint
		}
		read += n
		prev += diff
		dst[i] = prev
	}
	return read, nil
}
