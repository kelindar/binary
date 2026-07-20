// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/kelindar/bench"
	"github.com/kelindar/binary"
	"github.com/kelindar/binary/nocopy"
	"github.com/kelindar/binary/sorted"
	binunsafe "github.com/kelindar/binary/unsafe"
)

func main() {
	bench.Run(func(b *bench.B) {
		runBinary(b)
		runNocopy(b)
		runSorted(b)
		runUnsafe(b)
	},
		bench.WithSamples(100),
		bench.WithDuration(10*time.Millisecond),
	)
}

// ------------------------------------------------------------------------------
// Core binary package
// ------------------------------------------------------------------------------

type msg struct {
	Name      string
	Timestamp int64
	Payload   []byte
	Ssid      []uint32
}

type nested struct {
	Meta  msg
	Tags  map[string]string
	Items []msg
}

func runBinary(b *bench.B) {
	runBinaryMsg(b)
	runBinaryMap(b)
	runBinarySlice(b)
	runBinaryNested(b)
	runBinaryBytes(b)
	runBinaryUint64s(b)
	runBinaryReuse(b)
}

func runBinaryMsg(b *bench.B) {
	v := msg{
		Name:      "Roman",
		Timestamp: 1242345235,
		Payload:   []byte("hi"),
		Ssid:      []uint32{1, 2, 3},
	}
	enc, _ := binary.Marshal(&v)

	var buffer bytes.Buffer
	var out msg

	b.Run("binary/enc", func(int) { binary.Marshal(&v) })
	b.Run("binary/enc-to", func(int) {
		buffer.Reset()
		binary.MarshalTo(&v, &buffer)
	})
	b.Run("binary/dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runBinaryMap(b *bench.B) {
	v := make(map[string][]byte, 100)
	for i := 0; i < 100; i++ {
		v[fmt.Sprintf("key-%d", i)] = makeBytes(64)
	}
	enc, _ := binary.Marshal(&v)
	var out map[string][]byte

	b.Run("binary/map-enc", func(int) { binary.Marshal(&v) })
	b.Run("binary/map-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runBinarySlice(b *bench.B) {
	v := make([]msg, 100)
	for i := range v {
		v[i] = msg{
			Name:      fmt.Sprintf("msg-%d", i),
			Timestamp: int64(1500000000 + i),
			Payload:   makeBytes(32),
			Ssid:      []uint32{uint32(i), uint32(i + 1), uint32(i + 2)},
		}
	}
	enc, _ := binary.Marshal(&v)
	var out []msg

	b.Run("binary/slice-enc", func(int) { binary.Marshal(&v) })
	b.Run("binary/slice-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runBinaryNested(b *bench.B) {
	items := make([]msg, 50)
	for i := range items {
		items[i] = msg{
			Name:      fmt.Sprintf("item-%d", i),
			Timestamp: int64(1500000000 + i),
			Payload:   makeBytes(16),
			Ssid:      []uint32{uint32(i), 1, 2},
		}
	}
	v := nested{
		Meta: msg{
			Name:      "root",
			Timestamp: 1500000000,
			Payload:   []byte("meta"),
			Ssid:      []uint32{1, 2, 3},
		},
		Tags: map[string]string{
			"env":     "prod",
			"region":  "us-east",
			"service": "broker",
		},
		Items: items,
	}
	enc, _ := binary.Marshal(&v)
	var out nested

	b.Run("binary/nest-enc", func(int) { binary.Marshal(&v) })
	b.Run("binary/nest-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runBinaryBytes(b *bench.B) {
	v := makeBytes(defaultSize)
	enc, _ := binary.Marshal(&v)
	var out []byte

	b.Run("binary/bytes-enc", func(int) { binary.Marshal(&v) })
	b.Run("binary/bytes-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runBinaryUint64s(b *bench.B) {
	v := makeUint64s(defaultSize)
	enc, _ := binary.Marshal(&v)
	var out []uint64

	b.Run("binary/u64-enc", func(int) { binary.Marshal(&v) })
	b.Run("binary/u64-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runBinaryReuse(b *bench.B) {
	v := msg{
		Name:      "Roman",
		Timestamp: 1242345235,
		Payload:   makeBytes(256),
		Ssid:      []uint32{1, 2, 3, 4, 5},
	}
	enc, _ := binary.Marshal(&v)

	var buf bytes.Buffer
	encoder := binary.NewEncoder(&buf)
	_ = encoder.Encode(&v) // warm schema cache

	var out msg
	reader := bytes.NewReader(enc)
	decoder := binary.NewDecoder(reader)
	_ = decoder.Decode(&out) // warm schema cache

	b.Run("binary/reuse-enc", func(int) {
		buf.Reset()
		encoder.Reset(&buf)
		_ = encoder.Encode(&v)
	})
	b.Run("binary/stream-dec", func(int) {
		reader.Reset(enc)
		_ = decoder.Decode(&out)
	})
}

// ------------------------------------------------------------------------------
// nocopy package
// ------------------------------------------------------------------------------

const (
	testString  = "Donec egestas enim vitae turpis imperdiet ultricies. Vivamus sollicitudin in felis quis euismod. Nunc at tellus lectus."
	defaultSize = 10000
)

type nocopyComposite map[string]nocopyColumn

type nocopyColumn struct {
	Varchar nocopyColumnVarchar
	Float64 nocopyColumnFloat64
}

type nocopyColumnVarchar struct {
	Nulls nocopy.Bools
	Sizes nocopy.Uint32s
	Bytes nocopy.Bytes
}

type nocopyColumnFloat64 struct {
	Nulls  nocopy.Bools
	Floats nocopy.Float64s
}

type nocopyMessage struct {
	A nocopy.Bytes
	B nocopy.Bytes
	C nocopy.Bytes
	D nocopy.Bytes
}

func runNocopy(b *bench.B) {
	runNocopyString(b)
	runNocopyDictionary(b)
	runNocopyByteMap(b)
	runNocopyHashMap(b)
	runNocopyBytes(b)
	runNocopyUint64s(b)
	runNocopyColumnar(b)
	runNocopyStruct(b)
}

func runNocopyString(b *bench.B) {
	v := nocopy.String(testString)
	enc, _ := binary.Marshal(&v)
	var out nocopy.String

	b.Run("nocopy/str-enc", func(int) { binary.Marshal(&v) })
	b.Run("nocopy/str-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runNocopyDictionary(b *bench.B) {
	v := nocopy.Dictionary{
		"name":   "Roman",
		"race":   "human",
		"status": "happy",
	}
	enc, _ := binary.Marshal(&v)
	var out nocopy.Dictionary

	b.Run("nocopy/dict-enc", func(int) { binary.Marshal(&v) })
	b.Run("nocopy/dict-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runNocopyByteMap(b *bench.B) {
	v := nocopy.ByteMap{
		"name":   []byte(testString),
		"race":   []byte(testString),
		"status": []byte(testString),
	}
	enc, _ := binary.Marshal(&v)
	var out nocopy.ByteMap

	b.Run("nocopy/bmap-enc", func(int) { binary.Marshal(&v) })
	b.Run("nocopy/bmap-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runNocopyHashMap(b *bench.B) {
	v := nocopy.HashMap{
		1: []byte(testString),
		2: []byte(testString),
		3: []byte(testString),
	}
	enc, _ := binary.Marshal(&v)
	var out nocopy.HashMap

	b.Run("nocopy/hmap-enc", func(int) { binary.Marshal(&v) })
	b.Run("nocopy/hmap-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runNocopyBytes(b *bench.B) {
	v := nocopy.Bytes(makeBytes(defaultSize))
	enc, _ := binary.Marshal(&v)
	var out nocopy.Bytes

	b.Run("nocopy/bytes-enc", func(int) { binary.Marshal(&v) })
	b.Run("nocopy/bytes-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runNocopyUint64s(b *bench.B) {
	v := nocopy.Uint64s(makeUint64s(defaultSize))
	enc, _ := binary.Marshal(&v)
	var out nocopy.Uint64s

	b.Run("nocopy/u64-enc", func(int) { binary.Marshal(&v) })
	b.Run("nocopy/u64-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runNocopyColumnar(b *bench.B) {
	v := nocopyComposite{}
	v["a"] = nocopyColumn{
		Varchar: nocopyColumnVarchar{
			Nulls: nocopy.Bools{false, false, false, true, false, false, false, false, true, false, false, false, false, true, false},
			Sizes: nocopy.Uint32s{2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2, 2, 2, 2, 0, 2},
			Bytes: nocopy.Bytes{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
		},
	}
	v["b"] = nocopyColumn{
		Float64: nocopyColumnFloat64{
			Nulls:  nocopy.Bools{false, false, false, true, false},
			Floats: nocopy.Float64s{1.1, 2.2, 3.3, 0, 4.4},
		},
	}
	enc, _ := binary.Marshal(&v)
	var out nocopyComposite

	b.Run("nocopy/col-enc", func(int) { binary.Marshal(&v) })
	b.Run("nocopy/col-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runNocopyStruct(b *bench.B) {
	v := nocopyMessage{
		A: nocopy.Bytes{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
		B: nocopy.Bytes{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
		C: nocopy.Bytes{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
		D: nocopy.Bytes{10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10, 10},
	}
	enc, _ := binary.Marshal(&v)
	var out nocopyMessage

	b.Run("nocopy/struct-enc", func(int) { binary.Marshal(&v) })
	b.Run("nocopy/struct-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func makeUint64s(n int) (arr []uint64) {
	for i := 0; i < n; i++ {
		arr = append(arr, uint64(i))
	}
	return
}

func makeBytes(n int) (arr []byte) {
	for i := 0; i < n; i++ {
		arr = append(arr, byte(i%255))
	}
	return
}

// ------------------------------------------------------------------------------
// sorted package
// ------------------------------------------------------------------------------

func runSorted(b *bench.B) {
	runSortedSlice(b)
	runSortedTimestamps(b)
	runSortedTimeSeries(b)
	runSortedTimeCounters(b)
}

func runSortedSlice(b *bench.B) {
	ints := makeSortedInt32s(defaultSize)
	intsEnc, _ := binary.Marshal(&ints)
	uints := makeSortedUint32s(defaultSize)
	uintsEnc, _ := binary.Marshal(&uints)

	var intsOut sorted.Int32s
	var uintsOut sorted.Uint32s

	b.Run("sorted/i32-enc", func(int) { binary.Marshal(&ints) })
	b.Run("sorted/i32-dec", func(int) { binary.Unmarshal(intsEnc, &intsOut) })
	b.Run("sorted/u32-enc", func(int) { binary.Marshal(&uints) })
	b.Run("sorted/u32-dec", func(int) { binary.Unmarshal(uintsEnc, &uintsOut) })
}

func makeSortedInt32s(n int) sorted.Int32s {
	out := make(sorted.Int32s, n)
	for i := 0; i < n; i++ {
		out[i] = int32((i * 37) % n) // unsorted; codec sorts on encode
	}
	return out
}

func makeSortedUint32s(n int) sorted.Uint32s {
	out := make(sorted.Uint32s, n)
	for i := 0; i < n; i++ {
		out[i] = uint32((i * 37) % n)
	}
	return out
}

func runSortedTimestamps(b *bench.B) {
	var times sorted.Timestamps
	for i := uint64(0); i < 10000; i++ {
		times = append(times, uint64(time.Now().Unix())+i)
	}
	enc, _ := binary.Marshal(&times)
	var out sorted.Timestamps

	b.Run("sorted/ts-enc", func(int) { binary.Marshal(&times) })
	b.Run("sorted/ts-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runSortedTimeSeries(b *bench.B) {
	series := makeTimeSeries(20000)
	enc, _ := binary.Marshal(series)
	var out sorted.TimeSeries

	b.Run("sorted/tsz-enc", func(int) { binary.Marshal(series) })
	b.Run("sorted/tsz-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func runSortedTimeCounters(b *bench.B) {
	series := makeTimeCounters(20000)
	enc, _ := binary.Marshal(series)
	var out sorted.TimeCounters

	b.Run("sorted/tcz-enc", func(int) { binary.Marshal(series) })
	b.Run("sorted/tcz-dec", func(int) { binary.Unmarshal(enc, &out) })
}

func makeTimeSeries(count int) *sorted.TimeSeries {
	var ts sorted.TimeSeries
	for i := count - 1; i >= 0; i-- {
		ts.Append(uint64(1500000000+i), float64(i))
	}
	return &ts
}

func makeTimeCounters(count int) *sorted.TimeCounters {
	var ts sorted.TimeCounters
	for i := count - 1; i >= 0; i-- {
		ts.Append(uint64(1500000000+i), uint64(i))
	}
	return &ts
}

// ------------------------------------------------------------------------------
// unsafe package
// ------------------------------------------------------------------------------

var uint64Arr = []uint64{4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6,
	4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6, 4, 5, 6, 1, 2, 3, 5, 3, 2, 6, 1, 6, 7, 6, 1, 2, 6}

func runUnsafe(b *bench.B) {
	v := binunsafe.Uint64s(uint64Arr)
	enc, _ := binary.Marshal(&v)
	var out binunsafe.Uint64s

	b.Run("unsafe/u64-enc", func(int) { binary.Marshal(&v) })
	b.Run("unsafe/u64-dec", func(int) { binary.Unmarshal(enc, &out) })
}
