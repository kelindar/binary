package binary

import (
	"encoding/binary"
	"errors"
	"reflect"
)

// ------------------------------------------------------------------------------

type reflectArrayCodec struct {
	elemCodec codec // The codec of the array's elements
}

// Encode encodes a value into the encoder.
func (c *reflectArrayCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Type().Len()
	for i := 0; i < l; i++ {
		v := reflect.Indirect(rv.Index(i).Addr())
		if err = c.elemCodec.EncodeTo(e, v); err != nil {
			return
		}
	}
	return
}

// ------------------------------------------------------------------------------

type reflectSliceCodec struct {
	elemCodec codec // The codec of the slice's elements
}

// Encode encodes a value into the encoder.
func (c *reflectSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.writeUint64(uint64(l))
	for i := 0; i < l; i++ {
		v := reflect.Indirect(rv.Index(i).Addr())
		if err = c.elemCodec.EncodeTo(e, v); err != nil {
			return
		}
	}
	return
}

// ------------------------------------------------------------------------------

type byteSliceCodec struct{}

// Encode encodes a value into the encoder.
func (c *byteSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.writeUint64(uint64(l))
	_, err = e.Write(rv.Bytes())
	return
}

// ------------------------------------------------------------------------------

type varintSliceCodec struct{}

// Encode encodes a value into the encoder.
func (c *varintSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.writeUint64(uint64(l))
	for i := 0; i < l; i++ {
		e.writeInt64(rv.Index(i).Int())
	}
	return
}

// ------------------------------------------------------------------------------

type varuintSliceCodec struct{}

// Encode encodes a value into the encoder.
func (c *varuintSliceCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	l := rv.Len()
	e.writeUint64(uint64(l))
	for i := 0; i < l; i++ {
		e.writeUint64(rv.Index(i).Uint())
	}
	return
}

// ------------------------------------------------------------------------------

type reflectStructCodec struct {
	fields []fieldCodec // Codecs for all of the fields of the struct
}

type fieldCodec struct {
	index int
	codec codec
}

// Encode encodes a value into the encoder.
func (c *reflectStructCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	for _, i := range c.fields {
		if err = i.codec.EncodeTo(e, rv.Field(i.index)); err != nil {
			return
		}
	}
	return
}

// ------------------------------------------------------------------------------

type customMarshalCodec struct{}

// Encode encodes a value into the encoder.
func (c *customMarshalCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	m := getMetadata(rv.Type()).GetMarshalBinary(rv)
	if m == nil {
		return errors.New("MarshalBinary not found on " + rv.Type().String())
	}

	ret := m.Call([]reflect.Value{})
	if !ret[1].IsNil() {
		err = ret[1].Interface().(error)
		return
	}

	// Write the marshaled byte slice
	buffer := ret[0].Bytes()
	e.writeUint64(uint64(len(buffer)))
	_, err = e.Write(buffer)
	return
}

// ------------------------------------------------------------------------------

type reflectMapCodec struct {
	key codec // Codec for the key
	val codec // Codec for the value
}

// Encode encodes a value into the encoder.
func (c *reflectMapCodec) EncodeTo(e *Encoder, rv reflect.Value) (err error) {
	e.writeUint64(uint64(rv.Len()))

	for _, key := range rv.MapKeys() {
		value := rv.MapIndex(key)
		if err = c.key.EncodeTo(e, key); err != nil {
			return err
		}
		if err = c.val.EncodeTo(e, value); err != nil {
			return err
		}
	}

	return
}

// ------------------------------------------------------------------------------

type stringCodec struct{}

// Encode encodes a value into the encoder.
func (c *stringCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.writeString(rv.String())
	return nil
}

// ------------------------------------------------------------------------------

type boolCodec struct{}

// Encode encodes a value into the encoder.
func (c *boolCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.writeBool(rv.Bool())
	return nil
}

// ------------------------------------------------------------------------------

type varintCodec struct{}

// Encode encodes a value into the encoder.
func (c *varintCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.writeInt64(rv.Int())
	return nil
}

// ------------------------------------------------------------------------------

type varuintCodec struct{}

// Encode encodes a value into the encoder.
func (c *varuintCodec) EncodeTo(e *Encoder, rv reflect.Value) error {
	e.writeUint64(rv.Uint())
	return nil
}

// ------------------------------------------------------------------------------

type complex64Codec struct{}

// Encode encodes a value into the encoder.
func (c *complex64Codec) EncodeTo(e *Encoder, rv reflect.Value) error {
	return binary.Write(e, e.Order, complex64(rv.Complex()))
}

// ------------------------------------------------------------------------------

type complex128Codec struct{}

// Encode encodes a value into the encoder.
func (c *complex128Codec) EncodeTo(e *Encoder, rv reflect.Value) error {
	return binary.Write(e, e.Order, rv.Complex())
}

// ------------------------------------------------------------------------------

type float32Codec struct{}

// Encode encodes a value into the encoder.
func (c *float32Codec) EncodeTo(e *Encoder, rv reflect.Value) error {
	return binary.Write(e, e.Order, float32(rv.Float()))
}

// ------------------------------------------------------------------------------

type float64Codec struct{}

// Encode encodes a value into the encoder.
func (c *float64Codec) EncodeTo(e *Encoder, rv reflect.Value) error {
	return binary.Write(e, e.Order, rv.Float())
}
