// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package binary

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Map of all the schemas we've encountered so far
var schemas = new(sync.Map)

// Scan gets a codec for the type and uses a cached schema if the type was
// previously scanned.
func scan(t reflect.Type) (c Codec, err error) {

	// Attempt to load from cache first
	if f, ok := schemas.Load(t); ok {
		c = f.(Codec)
		return
	}

	// Scan for the first time
	c, err = scanType(t)
	if err != nil {
		return
	}

	// Load or store again
	if f, ok := schemas.LoadOrStore(t, c); ok {
		c = f.(Codec)
		return
	}
	return
}

// ScanType scans the type
func scanType(t reflect.Type) (Codec, error) {
	if custom, ok := scanCustomCodec(t); ok {
		return custom, nil
	}
	if custom, ok := scanBinaryMarshaler(t); ok {
		return custom, nil
	}

	switch t.Kind() {
	case reflect.Ptr:
		return scanPointer(t)
	case reflect.Array:
		return scanArray(t)
	case reflect.Slice:
		return scanSlice(t)
	case reflect.Struct:
		return scanStructCodec(t)
	case reflect.Map:
		return scanMap(t)
	default:
		if c := scanPrimitive(t.Kind()); c != nil {
			return c, nil
		}
		return nil, errors.New("binary: unsupported type " + t.String())
	}
}

func scanPointer(t reflect.Type) (Codec, error) {
	elemCodec, err := scanType(t.Elem())
	if err != nil {
		return nil, err
	}
	return &reflectPointerCodec{elemCodec: elemCodec}, nil
}

func scanArray(t reflect.Type) (Codec, error) {
	elemCodec, err := scanType(t.Elem())
	if err != nil {
		return nil, err
	}
	return &reflectArrayCodec{elemCodec: elemCodec}, nil
}

func scanSlice(t reflect.Type) (Codec, error) {
	switch t.Elem().Kind() {
	case reflect.Uint8:
		return new(byteSliceCodec), nil
	case reflect.Bool:
		return new(boolSliceCodec), nil
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &varuintSliceCodec{elemSize: t.Elem().Size()}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &varintSliceCodec{elemSize: t.Elem().Size()}, nil
	case reflect.Ptr:
		elemCodec, err := scanType(t.Elem().Elem())
		if err != nil {
			return nil, err
		}
		return &reflectSliceOfPtrCodec{
			elemType:  t.Elem().Elem(),
			elemCodec: elemCodec,
		}, nil
	default:
		elemCodec, err := scanType(t.Elem())
		if err != nil {
			return nil, err
		}
		return &reflectSliceCodec{elemCodec: elemCodec}, nil
	}
}

func scanStructCodec(t reflect.Type) (Codec, error) {
	n := t.NumField()
	var (
		hasTagged bool
		hasPlain  bool
		arms      []unionArm
		seen      map[uint64]struct{}
		maxTag    uint64
	)

	for i := range n {
		field := t.Field(i)
		tag := field.Tag.Get("binary")
		switch {
		case field.Name == "_" || tag == "-":
			continue
		case tag == "":
			hasPlain = true
			continue
		}

		value, option, ok := strings.Cut(tag, ",")
		if !ok {
			hasPlain = true
			continue
		}
		if option != "union" {
			return nil, errors.New("binary: invalid tag " + strconv.Quote(tag) + " on " + t.String())
		}

		id, err := strconv.ParseUint(value, 10, 64)
		switch {
		case err != nil || id == 0 || id > maxUnionTag:
			return nil, errors.New("binary: invalid tag " + strconv.Quote(tag) + " on " + t.String())
		case field.Type.Kind() != reflect.Ptr:
			return nil, errors.New("binary: union arm " + field.Name + " must be a pointer")
		}
		if seen == nil {
			seen = make(map[uint64]struct{}, n)
		}
		if _, ok := seen[id]; ok {
			return nil, errors.New("binary: duplicate union tag " + tag + " on " + t.String())
		}
		seen[id] = struct{}{}
		hasTagged = true
		if id > maxTag {
			maxTag = id
		}

		elem := field.Type.Elem()
		codec, err := scanType(elem)
		if err != nil {
			return nil, err
		}
		arms = append(arms, unionArm{
			tag:    id,
			index:  i,
			offset: field.Offset,
			elem:   elem,
			codec:  codec,
		})
	}

	switch {
	case hasTagged && hasPlain:
		return nil, errors.New("binary: mixed union and sequential fields on " + t.String())
	case hasTagged:
		return newUnionCodec(arms, maxTag), nil
	}

	v := make(reflectStructCodec, n)
	hasDirect := false
	for i := range n {
		field := t.Field(i)
		tag := field.Tag.Get("binary")
		if field.Name == "_" || tag == "-" {
			continue
		}
		codec, err := scanType(field.Type)
		if err != nil {
			return nil, err
		}
		kind := reflect.Invalid
		switch codec.(type) {
		case *stringCodec, *boolCodec, *varintCodec, *varuintCodec,
			*complex64Codec, *complex128Codec, *float32Codec, *float64Codec:
			kind = field.Type.Kind()
			hasDirect = true
		case *varuintSliceCodec:
			switch codec.(*varuintSliceCodec).elemSize {
			case 2:
				kind = fieldVaruint2
			case 4:
				kind = fieldVaruint4
			case 8:
				kind = fieldVaruint8
			}
		case *byteSliceCodec:
			kind = fieldByteSlice
		}
		packed := uint64(field.Offset) |
			uint64(kind)<<fieldKindShift |
			fieldIncluded
		if field.PkgPath == "" {
			packed |= fieldWritable
		}
		v[i] = fieldCodec{
			Field: packed,
			Codec: codec,
		}
	}
	if hasDirect {
		v[0].Field |= fieldDirect
	}
	return &v, nil
}

const maxUnionTag = 255 // wire tags are uvarint; arms must fit a dense 1..255 table (0 = none)

func newUnionCodec(arms []unionArm, maxTag uint64) *reflectUnionCodec {
	byTag := make([]int, maxTag+1)
	for i := range byTag {
		byTag[i] = -1
	}
	for i := range arms {
		byTag[arms[i].tag] = i
	}
	return &reflectUnionCodec{arms: arms, byTag: byTag}
}

func scanMap(t reflect.Type) (Codec, error) {
	switch t {
	case reflect.TypeFor[map[string][]byte]():
		return new(stringBytesMapCodec), nil
	case reflect.TypeFor[map[string]string]():
		return new(stringStringMapCodec), nil
	}

	key, err := scanType(t.Key())
	if err != nil {
		return nil, err
	}
	val, err := scanType(t.Elem())
	if err != nil {
		return nil, err
	}
	return &reflectMapCodec{key: key, val: val}, nil
}

func scanPrimitive(kind reflect.Kind) Codec {
	switch kind {
	case reflect.String:
		return new(stringCodec)
	case reflect.Bool:
		return new(boolCodec)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int, reflect.Int64:
		return new(varintCodec)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint, reflect.Uint64:
		return new(varuintCodec)
	case reflect.Complex64:
		return new(complex64Codec)
	case reflect.Complex128:
		return new(complex128Codec)
	case reflect.Float32:
		return new(float32Codec)
	case reflect.Float64:
		return new(float64Codec)
	default:
		return nil
	}
}

// scanBinaryMarshaler scans whether a type has a custom binary marshaling implemented.
func scanBinaryMarshaler(t reflect.Type) (Codec, bool) {
	out := new(customCodec)
	if m, ok := t.MethodByName("MarshalBinary"); ok {
		out.marshaler = &m
	} else if m, ok := reflect.PtrTo(t).MethodByName("MarshalBinary"); ok {
		out.ptrMarshaler = &m
	}

	if m, ok := t.MethodByName("UnmarshalBinary"); ok {
		out.unmarshaler = &m
	} else if m, ok := reflect.PtrTo(t).MethodByName("UnmarshalBinary"); ok {
		out.ptrUnmarshaler = &m
	}

	// Checks whether we have both marshaler and unmarshaler attached
	if (out.marshaler != nil || out.ptrMarshaler != nil) &&
		(out.unmarshaler != nil || out.ptrUnmarshaler != nil) {
		return out, true
	}

	return nil, false
}

// scanCustomCodec scans whether a type has a custom codec implemented.
func scanCustomCodec(t reflect.Type) (out Codec, ok bool) {
	if m, ok := reflect.PtrTo(t).MethodByName("GetBinaryCodec"); ok {
		callable := reflect.New(t).Method(m.Index)
		result := callable.Call([]reflect.Value{})
		if len(result) == 1 && !result[0].IsNil() {
			out, ok = result[0].Interface().(Codec)
			return out, ok
		}
	}
	return
}
