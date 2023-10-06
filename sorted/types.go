// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package sorted

import (
	"reflect"

	"github.com/kelindar/binary"
)

// ------------------------------------------------------------------------------

// Uint16s represents a slice serialized in an unsafe, non portable manner.
type Uint16s []uint16

func (s Uint16s) Len() int           { return len(s) }
func (s Uint16s) Less(i, j int) bool { return s[i] < s[j] }
func (s Uint16s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint16s) GetBinaryCodec() binary.Codec {
	return UintsCodecAs(reflect.TypeOf(Uint16s{}), 2)
}

// ------------------------------------------------------------------------------

// Int16s represents a slice serialized in an unsafe, non portable manner.
type Int16s []int16

func (s Int16s) Len() int           { return len(s) }
func (s Int16s) Less(i, j int) bool { return s[i] < s[j] }
func (s Int16s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int16s) GetBinaryCodec() binary.Codec {
	return IntsCodecAs(reflect.TypeOf(Int16s{}), 2)
}

// ------------------------------------------------------------------------------

// Uint32s represents a slice serialized in an unsafe, non portable manner.
type Uint32s []uint32

func (s Uint32s) Len() int           { return len(s) }
func (s Uint32s) Less(i, j int) bool { return s[i] < s[j] }
func (s Uint32s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint32s) GetBinaryCodec() binary.Codec {
	return UintsCodecAs(reflect.TypeOf(Uint32s{}), 4)
}

// ------------------------------------------------------------------------------

// Int32s represents a slice serialized in an unsafe, non portable manner.
type Int32s []int32

func (s Int32s) Len() int           { return len(s) }
func (s Int32s) Less(i, j int) bool { return s[i] < s[j] }
func (s Int32s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int32s) GetBinaryCodec() binary.Codec {
	return IntsCodecAs(reflect.TypeOf(Int32s{}), 4)
}

// ------------------------------------------------------------------------------

// Uint64s represents a slice serialized in an unsafe, non portable manner.
type Uint64s []uint64

func (s Uint64s) Len() int           { return len(s) }
func (s Uint64s) Less(i, j int) bool { return s[i] < s[j] }
func (s Uint64s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint64s) GetBinaryCodec() binary.Codec {
	return UintsCodecAs(reflect.TypeOf(Uint64s{}), 8)
}

// ------------------------------------------------------------------------------

// Int64s represents a slice serialized in an unsafe, non portable manner.
type Int64s []int64

func (s Int64s) Len() int           { return len(s) }
func (s Int64s) Less(i, j int) bool { return s[i] < s[j] }
func (s Int64s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int64s) GetBinaryCodec() binary.Codec {
	return IntsCodecAs(reflect.TypeOf(Int64s{}), 8)
}

// ------------------------------------------------------------------------------

// Timestamps represents the slice of sorted timestamps
type Timestamps []uint64

// GetBinaryCodec retrieves a custom binary codec.
func (ts *Timestamps) GetBinaryCodec() binary.Codec {
	return timestampCodec{}
}

// ------------------------------------------------------------------------------

// TimeSeries represents a compressed time-series data. The implementation is based
// on Gorilla paper (https://www.vldb.org/pvldb/vol8/p1816-teller.pdf), but instead
// of bit-weaving it is byte-aligned. If you are using this, consider using snappy
// compression on the output, as it will give a significantly better compression than
// simply marshaling the time-series using this binary encoder.
type TimeSeries struct {
	Time []uint64  // Sorted timestamps compressed using delta-encoding
	Data []float64 // Corresponding float-64 values
}

// Append appends a new value into the time series.
func (ts *TimeSeries) Append(time uint64, value float64) {
	ts.Time = append(ts.Time, time)
	ts.Data = append(ts.Data, value)
}

// Len returns the length of the time-series
func (ts *TimeSeries) Len() int {
	return len(ts.Time)
}

// Less compares two elements of the time series
func (ts *TimeSeries) Less(i, j int) bool {
	return ts.Time[i] < ts.Time[j]
}

// Swap swaps two elements of the time series
func (ts *TimeSeries) Swap(i, j int) {
	ts.Time[i], ts.Time[j] = ts.Time[j], ts.Time[i]
	ts.Data[i], ts.Data[j] = ts.Data[j], ts.Data[i]
}

// GetBinaryCodec retrieves a custom binary codec.
func (ts *TimeSeries) GetBinaryCodec() binary.Codec {
	return tszCodec{}
}

// ------------------------------------------------------------------------------

// TimeCounters represents a compressed time-series data where the value
// is itself an unsigned integer. This is particularly useful for counters.
type TimeCounters struct {
	Time []uint64 // Sorted timestamps compressed using delta-encoding
	Data []uint64 // Corresponding uint64 values
}

// Append appends a new value into the time series.
func (ts *TimeCounters) Append(time, value uint64) {
	ts.Time = append(ts.Time, time)
	ts.Data = append(ts.Data, value)
}

// Len returns the length of the time-series
func (ts *TimeCounters) Len() int {
	return len(ts.Time)
}

// Less compares two elements of the time series
func (ts *TimeCounters) Less(i, j int) bool {
	return ts.Time[i] < ts.Time[j]
}

// Swap swaps two elements of the time series
func (ts *TimeCounters) Swap(i, j int) {
	ts.Time[i], ts.Time[j] = ts.Time[j], ts.Time[i]
	ts.Data[i], ts.Data[j] = ts.Data[j], ts.Data[i]
}

// GetBinaryCodec retrieves a custom binary codec.
func (ts *TimeCounters) GetBinaryCodec() binary.Codec {
	return tczCodec{}
}
