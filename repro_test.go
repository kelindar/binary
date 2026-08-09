//go:build repro

package binary

import (
	"bytes"
	stdbinary "encoding/binary"
	"errors"
	"reflect"
	"testing"
)

type reproPointerMarshaler struct{}

type reproNamedPointer *int

type reproMapKey string

type reproZeroWire int

type reproZeroWireStruct struct{ value int }

type reproZeroSizedMarshaler struct{}

type reproZeroWireCodec struct{}

type reproNilCodecType struct{}

type reproNilCodec struct{}

func (k reproMapKey) MarshalBinary() ([]byte, error) {
	return []byte("key:" + string(k)), nil
}

func (k *reproMapKey) UnmarshalBinary(data []byte) error {
	if !bytes.HasPrefix(data, []byte("key:")) {
		return errors.New("invalid map key")
	}
	*k = reproMapKey(string(data[4:]))
	return nil
}

func (*reproPointerMarshaler) MarshalBinary() ([]byte, error) { return nil, nil }

func (*reproPointerMarshaler) UnmarshalBinary([]byte) error { return nil }

func (reproZeroWire) GetBinaryCodec() Codec { return reproZeroWireCodec{} }

func (reproZeroWireStruct) GetBinaryCodec() Codec { return reproZeroWireCodec{} }

func (reproZeroSizedMarshaler) MarshalBinary() ([]byte, error) { return nil, nil }

func (*reproZeroSizedMarshaler) UnmarshalBinary([]byte) error { return nil }

func (reproZeroWireCodec) EncodeTo(*Encoder, reflect.Value) error { return nil }

func (reproZeroWireCodec) DecodeTo(*Decoder, reflect.Value) error { return nil }

func (*reproNilCodecType) GetBinaryCodec() Codec { return (*reproNilCodec)(nil) }

func (reproNilCodec) EncodeTo(*Encoder, reflect.Value) error { return nil }

func (reproNilCodec) DecodeTo(*Decoder, reflect.Value) error { return nil }

func TestArenaMap(t *testing.T) {
	data := []byte{
		2,
		1, 0, 'a', 1, 'x',
		1, 0, 'b', 1, 'y',
	}
	var got map[string][]byte
	if err := Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	sibling := got["b"]
	first := append(got["a"], 'z', 'q')
	if !bytes.Equal(sibling, []byte("y")) {
		t.Fatalf("appending to one decoded value changed its sibling: first=%q sibling=%q", first, sibling)
	}
}

func TestUnionArena(t *testing.T) {
	in := reproUnionMapEnvelope{
		Body: reproUnionMapContainer{
			Arm: &reproUnionMapArm{Values: map[string][]byte{"a": {'x'}}},
		},
		Tail: map[string][]byte{"t": bytes.Repeat([]byte{'y'}, 16)},
	}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("decoding a union followed by a map panicked: %v", recovered)
		}
	}()
	var got reproUnionMapEnvelope
	if err := Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
}

func TestHugeLength(t *testing.T) {
	data := stdbinary.AppendUvarint(nil, ^uint64(0))
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("malformed length panicked: %v", recovered)
		}
	}()

	var got []byte
	if err := Unmarshal(data, &got); err == nil {
		t.Fatal("expected malformed length to return an error")
	}
}

func TestSliceLengths(t *testing.T) {
	data := stdbinary.AppendUvarint(nil, ^uint64(0))
	for name, out := range map[string]any{
		"bools":    new([]bool),
		"ints":     new([]int),
		"strings":  new([]string),
		"pointers": new([]*int),
		"struct":   new(struct{ Bytes []byte }),
		"map":      new(map[int]string),
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("malformed length panicked: %v", recovered)
				}
			}()
			if err := Unmarshal(data, out); err == nil {
				t.Fatal("expected malformed length to return an error")
			}
		})
	}
}

func TestHugeMapCount(t *testing.T) {
	data := stdbinary.AppendUvarint(nil, ^uint64(0))
	got := map[string]string{"stale": "value"}
	if err := Unmarshal(data, &got); err == nil {
		t.Fatal("expected malformed map count to return an error")
	}
}

func TestShortRead(t *testing.T) {
	input := []byte("payload")
	data, err := Marshal(input)
	if err != nil {
		t.Fatal(err)
	}

	var got []byte
	if err := NewDecoder(&oneByteReader{content: data}).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, input) {
		t.Fatalf("short read silently corrupted payload: got=%q want=%q", got, input)
	}
}

func TestLargeLengths(t *testing.T) {
	data := stdbinary.AppendUvarint(nil, 1<<62)
	tests := map[string]any{
		"bytes":    new([]byte),
		"bools":    new([]bool),
		"floats":   new([]float64),
		"integers": new([]int64),
		"strings":  new([]string),
	}
	reads := map[string]func([]byte, any) error{
		"slice":  Unmarshal,
		"stream": func(data []byte, out any) error { return NewDecoder(bytes.NewReader(data)).Decode(out) },
	}
	for name, out := range tests {
		for mode, read := range reads {
			t.Run(name+"/"+mode, func(t *testing.T) {
				defer func() {
					if recovered := recover(); recovered != nil {
						t.Fatalf("large truncated length panicked: %v", recovered)
					}
				}()
				if err := read(data, out); err == nil {
					t.Fatal("expected large truncated length to return an error")
				}
			})
		}
	}
}

func TestLongMapKey(t *testing.T) {
	key := string(bytes.Repeat([]byte{'k'}, 1<<16))
	for name, input := range map[string]any{
		"specialized": map[string]string{key: "value"},
		"generic":     map[string]uint32{key: 1},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Marshal(input); err == nil {
				t.Fatal("expected an oversized map key to be rejected")
			}
		})
	}
}

func TestZeroWire(t *testing.T) {
	type item struct{ _ int }
	data, err := Marshal([]item{{}, {}})
	if err != nil {
		t.Fatal(err)
	}

	var got []item
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("zero-wire elements should decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("decoded %d elements, want 2", len(got))
	}
}

func TestNestedZeroWire(t *testing.T) {
	type empty struct{ _ int }
	type item [2]empty
	data, err := Marshal([]item{{{}, {}}})
	if err != nil {
		t.Fatal(err)
	}
	var got []item
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("nested zero-wire elements should decode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("decoded %d elements, want 1", len(got))
	}
}

func TestNamedBoolSlice(t *testing.T) {
	type values []bool
	data, err := Marshal(values{true, false})
	if err != nil {
		t.Fatal(err)
	}

	var got values
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("named bool slice should decode: %v", err)
	}
	if len(got) != 2 || !got[0] || got[1] {
		t.Fatalf("decoded named bool slice incorrectly: %v", got)
	}
}

func TestNamedByteElements(t *testing.T) {
	type element byte
	type values []element
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("named byte elements panicked: %v", recovered)
		}
	}()
	data, err := Marshal(values{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	var got values
	if err := Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("decoded named byte elements incorrectly: %v", got)
	}
	got = nil
	if err := NewDecoder(bytes.NewReader(data)).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("stream decoded named byte elements incorrectly: %v", got)
	}
}

func TestHugeZeroWire(t *testing.T) {
	data := stdbinary.AppendUvarint(nil, 1<<62)
	var got []struct{}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("huge zero-wire length panicked: %v", recovered)
		}
	}()
	if err := Unmarshal(data, &got); err == nil {
		t.Fatal("expected huge zero-wire length to return an error")
	}
}

func TestLargeStreamLengths(t *testing.T) {
	data := stdbinary.AppendUvarint(nil, 1<<61)
	tests := map[string]any{
		"bytes":    new([]byte),
		"bools":    new([]bool),
		"floats":   new([]float64),
		"integers": new([]int64),
		"strings":  new([]string),
		"pointers": new([]*int),
		"structs":  new([]struct{ Value int }),
		"map":      new(map[int]string),
		"custom":   new(s2),
	}
	for name, out := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("large truncated stream length panicked: %v", recovered)
				}
			}()
			if err := NewDecoder(bytes.NewReader(data)).Decode(out); err == nil {
				t.Fatal("expected large truncated stream length to return an error")
			}
		})
	}
}

func TestNamedByteStream(t *testing.T) {
	type values []byte
	data, err := Marshal(values("payload"))
	if err != nil {
		t.Fatal(err)
	}
	var got values
	if err := NewDecoder(bytes.NewReader(data)).Decode(&got); err != nil {
		t.Fatalf("named byte slice stream should decode: %v", err)
	}
	if !bytes.Equal(got, []byte("payload")) {
		t.Fatalf("decoded named byte slice incorrectly: %q", got)
	}
}

func TestUnexportedField(t *testing.T) {
	type item struct {
		hidden  int
		Visible int
	}
	in := item{hidden: 7, Visible: 42}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var got item
	if err := Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Visible != in.Visible {
		t.Fatalf("unexported field shifted exported field: got %d want %d", got.Visible, in.Visible)
	}
	if got.hidden != 0 {
		t.Fatalf("unexported field was serialized: got %d", got.hidden)
	}
}

func TestPointerMarshalerValue(t *testing.T) {
	var input reproPointerMarshaler
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("non-addressable pointer marshaler panicked: %v", recovered)
		}
	}()
	if _, err := Marshal(input); err == nil {
		t.Fatal("expected a non-addressable pointer marshaler to return an error")
	}
}

func TestNilEncode(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("nil encode panicked: %v", recovered)
		}
	}()
	var value *int
	if _, err := Marshal(value); err == nil {
		t.Fatal("expected nil encode to return an error")
	}
}

func TestNilEncoderWriter(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("nil encoder writer panicked: %v", recovered)
		}
	}()
	for _, input := range []any{1, complex64(1), complex128(1)} {
		if err := MarshalTo(input, nil); err == nil {
			t.Fatalf("expected nil encoder writer to return an error for %T", input)
		}
	}
}

func TestNamedPointerDecode(t *testing.T) {
	type item struct {
		Value reproNamedPointer
	}
	value := 42
	data, err := Marshal(item{Value: reproNamedPointer(&value)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("named pointer decode panicked: %v", recovered)
		}
	}()
	var got item
	if err := Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Value == nil || *got.Value != value {
		t.Fatalf("decoded named pointer incorrectly: %v", got.Value)
	}
}

func TestCustomPointerReuse(t *testing.T) {
	type item struct {
		Value *s2
	}
	data, err := Marshal(item{Value: &s2{b: []byte{1}}})
	if err != nil {
		t.Fatal(err)
	}
	previous := &s2{}
	got := item{Value: previous}
	if err := Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Value != previous {
		t.Fatal("custom pointer decoder discarded reusable storage")
	}
}

func TestZeroWireMap(t *testing.T) {
	type key struct{ hidden int }
	type value struct{ hidden int }
	input := map[key]value{{hidden: 1}: {hidden: 2}, {hidden: 3}: {hidden: 4}}
	data, err := Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var got map[key]value
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("decoder rejected its own zero-wire map: %v", err)
	}
}

func TestCustomMapKey(t *testing.T) {
	data := []byte{1, 8, 'k', 'e', 'y', ':', 'n', 'a', 'm', 'e', 1}
	var got map[reproMapKey]uint64
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("custom map key was not decoded through its codec: %v", err)
	}
	if got[reproMapKey("name")] != 1 {
		t.Fatalf("decoded custom map key incorrectly: %#v", got)
	}
}

func TestWriterNil(t *testing.T) {
	var writer *bytes.Buffer
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("typed-nil writer panicked: %v", recovered)
		}
	}()
	if err := MarshalTo([]int{1}, writer); err == nil {
		t.Fatal("expected typed-nil writer to return an error")
	}
}

func TestReaderNil(t *testing.T) {
	var reader *bytes.Reader
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("typed-nil reader panicked: %v", recovered)
		}
	}()
	var value int
	if err := NewDecoder(reader).Decode(&value); err == nil {
		t.Fatal("expected typed-nil reader to return an error")
	}
}

func TestMapZeroWire(t *testing.T) {
	input := map[reproZeroWire]reproZeroWire{1: 2, 3: 4}
	data, err := Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var got map[reproZeroWire]reproZeroWire
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("custom zero-wire map was rejected: %v", err)
	}
}

func TestSliceZeroWire(t *testing.T) {
	sliceData, err := Marshal([]reproZeroWireStruct{{value: 1}, {value: 3}})
	if err != nil {
		t.Fatal(err)
	}
	var gotSlice []reproZeroWireStruct
	if err := Unmarshal(sliceData, &gotSlice); err != nil {
		t.Fatalf("custom zero-wire slice was rejected: %v", err)
	}
}

func TestSliceCodec(t *testing.T) {
	data, err := Marshal([]reproZeroWire{1, 3})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte{2}) {
		t.Fatalf("custom element codec was bypassed: wire=%v", data)
	}
}

func TestCodecNil(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("typed-nil custom codec panicked: %v", recovered)
		}
	}()
	if _, err := Marshal(reproNilCodecType{}); err == nil {
		t.Fatal("expected typed-nil custom codec to return an error")
	}
}

func TestMapArenaCopy(t *testing.T) {
	type envelope struct {
		Values  map[string]string
		Payload []byte
	}
	data, err := Marshal(envelope{Payload: bytes.Repeat([]byte{'x'}, 1<<20)})
	if err != nil {
		t.Fatal(err)
	}
	decoder := NewDecoder(bytes.NewBuffer(data))
	var got envelope
	if err := decoder.Decode(&got); err != nil {
		t.Fatal(err)
	}
	if decoder.arena != nil {
		t.Fatalf("empty map retained an arena for unrelated trailing payload: %d bytes", len(decoder.arena))
	}
}

func TestMapArenaBounds(t *testing.T) {
	type envelope struct {
		Values  map[string]string
		Payload []byte
	}
	data, err := Marshal(envelope{
		Values:  map[string]string{"key": "value"},
		Payload: bytes.Repeat([]byte{'x'}, 1<<20),
	})
	if err != nil {
		t.Fatal(err)
	}
	decoder := NewDecoder(bytes.NewBuffer(data))
	var got envelope
	if err := decoder.Decode(&got); err != nil {
		t.Fatal(err)
	}
	if cap(decoder.arena) >= len(data)/2 {
		t.Fatalf("map arena retained unrelated trailing payload: cap=%d input=%d", cap(decoder.arena), len(data))
	}
}

func TestMapKeyArena(t *testing.T) {
	type envelope struct {
		Values  map[reproMapKey]uint64
		Payload []byte
	}
	data, err := Marshal(envelope{
		Values:  map[reproMapKey]uint64{"name": 1},
		Payload: bytes.Repeat([]byte{'x'}, 1<<20),
	})
	if err != nil {
		t.Fatal(err)
	}
	decoder := NewDecoder(bytes.NewBuffer(data))
	var got envelope
	if err := decoder.Decode(&got); err != nil {
		t.Fatal(err)
	}
	if decoder.arena != nil {
		t.Fatalf("custom map key retained an unused arena: %d bytes", len(decoder.arena))
	}
}

func TestZeroWireStream(t *testing.T) {
	type empty struct{ hidden int }
	data := stdbinary.AppendUvarint(nil, 1<<59)
	var got []empty
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("huge zero-wire stream length panicked: %v", recovered)
		}
	}()
	if err := NewDecoder(bytes.NewReader(data)).Decode(&got); err == nil {
		t.Fatal("expected huge zero-wire stream length to return an error")
	}
}

func TestSliceEOF(t *testing.T) {
	data := stdbinary.AppendUvarint(nil, 1<<30)
	var got []reproZeroSizedMarshaler
	if err := Unmarshal(data, &got); err == nil {
		t.Fatal("expected truncated custom slice return an error")
	}
	if len(got) != 0 {
		t.Fatalf("truncated custom slice was resized to %d elements", len(got))
	}
}
