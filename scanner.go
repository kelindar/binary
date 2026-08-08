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

var schemas = new(sync.Map)

func scan(t reflect.Type) (c Codec, err error) {
	if f, ok := schemas.Load(t); ok {
		return f.(Codec), nil
	}
	c, err = scanType(t)
	if err != nil {
		return nil, err
	}
	if f, ok := schemas.LoadOrStore(t, c); ok {
		c = f.(Codec)
	}
	return c, nil
}

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
	if codec := scanFixedWidth(t.Elem(), true); codec != nil {
		return codec, nil
	}
	elemCodec, err := scanType(t.Elem())
	if err != nil {
		return nil, err
	}
	return &reflectCollectionCodec{elemCodec: elemCodec, array: true}, nil
}

func scanSlice(t reflect.Type) (Codec, error) {
	if codec := scanFixedWidth(t.Elem(), false); codec != nil {
		return codec, nil
	}
	switch t.Elem().Kind() {
	case reflect.Uint8:
		return new(byteSliceCodec), nil
	case reflect.Bool:
		return new(boolSliceCodec), nil
	case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &varSliceCodec{elemSize: t.Elem().Size()}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &varSliceCodec{elemSize: t.Elem().Size(), signed: true}, nil
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
		return &reflectCollectionCodec{elemCodec: elemCodec}, nil
	}
}

func scanFixedWidth(elem reflect.Type, array bool) Codec {
	switch elem {
	case reflect.TypeFor[string]():
		return &stringSliceCodec{array: array}
	case reflect.TypeFor[float32](), reflect.TypeFor[float64]():
		return &fixedSliceCodec{elemSize: elem.Size(), array: array}
	case reflect.TypeFor[complex64](), reflect.TypeFor[complex128]():
		return &fixedSliceCodec{elemSize: elem.Size(), array: array, complex: true}
	default:
		return nil
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
		switch codec := codec.(type) {
		case *primitiveCodec:
			kind = field.Type.Kind()
			hasDirect = true
		case *varSliceCodec:
			if codec.signed {
				break
			}
			switch codec.elemSize {
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
		return stringMapCodec[[]byte]{}, nil
	case reflect.TypeFor[map[string]string]():
		return stringMapCodec[string]{}, nil
	case reflect.TypeFor[map[uint64]uint64]():
		return new(uint64MapCodec), nil
	case reflect.TypeFor[map[string]uint64]():
		return stringMapCodec[uint64]{}, nil
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
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Complex64, reflect.Complex128, reflect.Float32, reflect.Float64:
		return new(primitiveCodec)
	default:
		return nil
	}
}

func scanBinaryMarshaler(t reflect.Type) (Codec, bool) {
	out := new(customCodec)
	out.marshaler, out.ptrMarshaler = findMethod(t, "MarshalBinary")
	out.unmarshaler, out.ptrUnmarshaler = findMethod(t, "UnmarshalBinary")
	if (out.marshaler != nil || out.ptrMarshaler != nil) &&
		(out.unmarshaler != nil || out.ptrUnmarshaler != nil) {
		return out, true
	}
	return nil, false
}

func findMethod(t reflect.Type, name string) (value, pointer *reflect.Method) {
	if method, ok := t.MethodByName(name); ok {
		return &method, nil
	}
	if method, ok := reflect.PtrTo(t).MethodByName(name); ok {
		return nil, &method
	}
	return nil, nil
}

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
