// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
package sorted

import (
	"github.com/kelindar/binary"
	"reflect"
)

// ------------------------------------------------------------------------------

type Uint16s []uint16

func (s Uint16s) Len() int                      { return len(s) }
func (s Uint16s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Uint16s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Uint16s) GetBinaryCodec() binary.Codec { return UintsCodecAs(reflect.TypeFor[Uint16s](), 2) }

// ------------------------------------------------------------------------------

type Int16s []int16

func (s Int16s) Len() int                      { return len(s) }
func (s Int16s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Int16s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Int16s) GetBinaryCodec() binary.Codec { return IntsCodecAs(reflect.TypeFor[Int16s](), 2) }

// ------------------------------------------------------------------------------

type Uint32s []uint32

func (s Uint32s) Len() int                      { return len(s) }
func (s Uint32s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Uint32s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Uint32s) GetBinaryCodec() binary.Codec { return UintsCodecAs(reflect.TypeFor[Uint32s](), 4) }

// ------------------------------------------------------------------------------

type Int32s []int32

func (s Int32s) Len() int                      { return len(s) }
func (s Int32s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Int32s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Int32s) GetBinaryCodec() binary.Codec { return IntsCodecAs(reflect.TypeFor[Int32s](), 4) }

// ------------------------------------------------------------------------------

type Uint64s []uint64

func (s Uint64s) Len() int                      { return len(s) }
func (s Uint64s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Uint64s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Uint64s) GetBinaryCodec() binary.Codec { return UintsCodecAs(reflect.TypeFor[Uint64s](), 8) }

// ------------------------------------------------------------------------------

type Int64s []int64

func (s Int64s) Len() int                      { return len(s) }
func (s Int64s) Less(i, j int) bool            { return s[i] < s[j] }
func (s Int64s) Swap(i, j int)                 { s[i], s[j] = s[j], s[i] }
func (s *Int64s) GetBinaryCodec() binary.Codec { return IntsCodecAs(reflect.TypeFor[Int64s](), 8) }

// ------------------------------------------------------------------------------

type Timestamps []uint64

func (ts *Timestamps) GetBinaryCodec() binary.Codec { return timestampCodec{} }

// ------------------------------------------------------------------------------

type TimeSeries struct {
	Time []uint64  // Sorted timestamps compressed using delta-encoding
	Data []float64 // Corresponding float-64 values
}

func (ts *TimeSeries) Append(time uint64, value float64) {
	ts.Time = append(ts.Time, time)
	ts.Data = append(ts.Data, value)
}
func (ts *TimeSeries) Len() int           { return len(ts.Time) }
func (ts *TimeSeries) Less(i, j int) bool { return ts.Time[i] < ts.Time[j] }
func (ts *TimeSeries) Swap(i, j int) {
	ts.Time[i], ts.Time[j] = ts.Time[j], ts.Time[i]
	ts.Data[i], ts.Data[j] = ts.Data[j], ts.Data[i]
}
func (ts *TimeSeries) GetBinaryCodec() binary.Codec { return tszCodec{} }

// ------------------------------------------------------------------------------

type TimeCounters struct {
	Time []uint64 // Sorted timestamps compressed using delta-encoding
	Data []uint64 // Corresponding uint64 values
}

func (ts *TimeCounters) Append(time, value uint64) {
	ts.Time = append(ts.Time, time)
	ts.Data = append(ts.Data, value)
}
func (ts *TimeCounters) Len() int           { return len(ts.Time) }
func (ts *TimeCounters) Less(i, j int) bool { return ts.Time[i] < ts.Time[j] }
func (ts *TimeCounters) Swap(i, j int) {
	ts.Time[i], ts.Time[j] = ts.Time[j], ts.Time[i]
	ts.Data[i], ts.Data[j] = ts.Data[j], ts.Data[i]
}
func (ts *TimeCounters) GetBinaryCodec() binary.Codec { return tczCodec{} }
