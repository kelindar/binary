// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package unsafe

import (
	"reflect"

	"github.com/kelindar/binary"
)

// ------------------------------------------------------------------------------

// Bools represents a slice serialized in an unsafe, non portable manner.
type Bools []bool

// GetBinaryCodec retrieves a custom binary codec.
func (s *Bools) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Bools{}),
		sizeOfInt: 1,
	}
}

// ------------------------------------------------------------------------------

// Uint16s represents a slice serialized in an unsafe, non portable manner.
type Uint16s []uint16

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint16s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Uint16s{}),
		sizeOfInt: 2,
	}
}

// ------------------------------------------------------------------------------

// Int16s represents a slice serialized in an unsafe, non portable manner.
type Int16s []int16

func (s Int16s) Len() int           { return len(s) }
func (s Int16s) Less(i, j int) bool { return s[i] < s[j] }
func (s Int16s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int16s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Int16s{}),
		sizeOfInt: 2,
	}
}

// ------------------------------------------------------------------------------

// Uint32s represents a slice serialized in an unsafe, non portable manner.
type Uint32s []uint32

func (s Uint32s) Len() int           { return len(s) }
func (s Uint32s) Less(i, j int) bool { return s[i] < s[j] }
func (s Uint32s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint32s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Uint32s{}),
		sizeOfInt: 4,
	}
}

// ------------------------------------------------------------------------------

// Int32s represents a slice serialized in an unsafe, non portable manner.
type Int32s []int32

func (s Int32s) Len() int           { return len(s) }
func (s Int32s) Less(i, j int) bool { return s[i] < s[j] }
func (s Int32s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int32s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Int32s{}),
		sizeOfInt: 4,
	}
}

// ------------------------------------------------------------------------------

// Uint64s represents a slice serialized in an unsafe, non portable manner.
type Uint64s []uint64

func (s Uint64s) Len() int           { return len(s) }
func (s Uint64s) Less(i, j int) bool { return s[i] < s[j] }
func (s Uint64s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Uint64s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Uint64s{}),
		sizeOfInt: 8,
	}
}

// ------------------------------------------------------------------------------

// Int64s represents a slice serialized in an unsafe, non portable manner.
type Int64s []int64

func (s Int64s) Len() int           { return len(s) }
func (s Int64s) Less(i, j int) bool { return s[i] < s[j] }
func (s Int64s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Int64s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Int64s{}),
		sizeOfInt: 8,
	}
}

// ------------------------------------------------------------------------------

// Float32s represents a slice serialized in an unsafe, non portable manner.
type Float32s []float32

func (s Float32s) Len() int           { return len(s) }
func (s Float32s) Less(i, j int) bool { return s[i] < s[j] }
func (s Float32s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Float32s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Float32s{}),
		sizeOfInt: 4,
	}
}

// ------------------------------------------------------------------------------

// Float64s represents a slice serialized in an unsafe, non portable manner.
type Float64s []float64

func (s Float64s) Len() int           { return len(s) }
func (s Float64s) Less(i, j int) bool { return s[i] < s[j] }
func (s Float64s) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// GetBinaryCodec retrieves a custom binary codec.
func (s *Float64s) GetBinaryCodec() binary.Codec {
	return &integerSliceCodec{
		sliceType: reflect.TypeOf(Float64s{}),
		sizeOfInt: 8,
	}
}
