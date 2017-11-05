package binary

import (
	"errors"
	"reflect"
	"sync"
)

// Map of all the schemas we've encountered so far
var schemas = new(sync.Map)

// Codec represents a single part codec, which can encode something.
type codec interface {
	EncodeTo(*Encoder, reflect.Value) error
}

// Encode encodes the value to the binary format.
func scan(t reflect.Type) (c codec, err error) {

	// Attempt to load from cache first
	if f, ok := schemas.Load(t); ok {
		c = f.(codec)
		return
	}

	// Scan for the first time
	c, err = scanType(t)
	if err != nil {
		return
	}

	// Load or store again
	if f, ok := schemas.LoadOrStore(t, c); ok {
		c = f.(codec)
		return
	}
	return
}

func scanType(t reflect.Type) (codec, error) {
	switch t.Kind() {
	case reflect.Array:
		elemCodec, err := scanType(t.Elem())
		if err != nil {
			return nil, err
		}

		return &reflectArrayCodec{
			elemCodec: elemCodec,
		}, nil

	case reflect.Slice:

		// Fast-paths for simple numeric slices and string slices
		switch t.Elem().Kind() {
		case reflect.Int8:
			fallthrough
		case reflect.Uint8:
			return new(byteSliceCodec), nil

		case reflect.Uint:
			fallthrough
		case reflect.Uint16:
			fallthrough
		case reflect.Uint32:
			fallthrough
		case reflect.Uint64:
			return new(varuintSliceCodec), nil

		case reflect.Int:
			fallthrough
		case reflect.Int16:
			fallthrough
		case reflect.Int32:
			fallthrough
		case reflect.Int64:
			return new(varintSliceCodec), nil

		default:
			elemCodec, err := scanType(t.Elem())
			if err != nil {
				return nil, err
			}

			return &reflectSliceCodec{
				elemCodec: elemCodec,
			}, nil
		}

	case reflect.Ptr:
		println("reflect.Ptr")

	case reflect.Struct:
		meta := getMetadata(t)

		if meta.IsCustom() {
			return new(customMarshalCodec), nil
		}

		var v reflectStructCodec
		for _, i := range meta.fields {
			if c, err := scanType(t.Field(i).Type); err == nil {
				v.fields = append(v.fields, fieldCodec{index: i, codec: c})
			} else {
				return nil, err
			}
		}

		return &v, nil

	case reflect.Map:
		key, err := scanType(t.Key())
		if err != nil {
			return nil, err
		}

		val, err := scanType(t.Elem())
		if err != nil {
			return nil, err
		}

		return &reflectMapCodec{
			key: key,
			val: val,
		}, nil

	case reflect.String:
		return new(stringCodec), nil

	case reflect.Bool:
		return new(boolCodec), nil

	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int:
		fallthrough
	case reflect.Int64:
		return new(varintCodec), nil

	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint:
		fallthrough
	case reflect.Uint64:
		return new(varuintCodec), nil

	case reflect.Complex64:
		return new(complex64Codec), nil

	case reflect.Complex128:
		return new(complex128Codec), nil

	case reflect.Float32:
		return new(float32Codec), nil

	case reflect.Float64:
		return new(float64Codec), nil
	}

	return nil, errors.New("binary: unsupported type " + t.String())
}
