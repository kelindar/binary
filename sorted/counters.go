package sorted

import (
	"reflect"
	"sort"

	"github.com/kelindar/binary"
)

type tczCodec struct{}

func (tczCodec) EncodeTo(e *binary.Encoder, rv reflect.Value) (err error) {
	data := rv.Interface().(TimeCounters)
	if len(data.Time) != len(data.Data) {
		return errMismatchedSeries
	}
	if !isSorted(data.Time) {
		sort.Sort(&data)
	}
	buffer := make([]byte, 0, 4*len(data.Time))
	buffer = appendDelta(buffer, data.Time)
	buffer = appendDelta(buffer, data.Data)
	e.WriteUvarint(uint64(len(data.Time)))
	e.WriteUvarint(uint64(len(buffer)))
	e.Write(buffer)
	return
}
func (tczCodec) DecodeTo(d *binary.Decoder, rv reflect.Value) error {
	count, err := d.ReadUvarint()
	if err != nil {
		return err
	}
	n, err := decodeLength(count)
	if err != nil {
		return err
	}
	size, err := d.ReadUvarint()
	if err != nil {
		return err
	}
	bufferSize, err := decodeLength(size)
	if err != nil {
		return err
	}
	if n > bufferSize/2 {
		return errInvalidVarint
	}
	buffer, err := d.Slice(bufferSize)
	if err != nil {
		return err
	}
	result := rv.Interface().(TimeCounters)
	if result.Time == nil || cap(result.Time) < n {
		result.Time = make([]uint64, n)
	} else {
		result.Time = result.Time[:n]
	}
	if result.Data == nil || cap(result.Data) < n {
		result.Data = make([]uint64, n)
	} else {
		result.Data = result.Data[:n]
	}
	offset, err := readDelta(result.Time, buffer)
	if err != nil {
		return err
	}
	if _, err = readDelta(result.Data, buffer[offset:]); err != nil {
		return err
	}
	rv.Set(reflect.ValueOf(result))
	return nil
}
