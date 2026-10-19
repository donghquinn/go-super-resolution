package main

// Run with: go test sr-backend.go types.go onnx_engine.go onnx_engine_test.go

import (
	"math"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
)

func TestFloat16RoundTrip(t *testing.T) {
	for h := 0; h <= 0xffff; h++ {
		f := float16ToFloat32(uint16(h))
		if math.IsNaN(float64(f)) {
			continue
		}
		if got := float32ToFloat16(f); got != uint16(h) {
			t.Fatalf("round trip %#04x -> %v -> %#04x", h, f, got)
		}
	}
}

func TestFloat32ToFloat16(t *testing.T) {
	cases := []struct {
		in   float32
		want uint16
	}{
		{0, 0x0000},
		{1, 0x3c00},
		{0.5, 0x3800},
		{-2, 0xc000},
		{0.1, 0x2e66},
		{65504, 0x7bff},
		{1e6, 0x7c00},
		{5.9604645e-8, 0x0001}, // smallest subnormal
		{1e-9, 0x0000},
		{float32(math.Inf(-1)), 0xfc00},
	}
	for _, c := range cases {
		if got := float32ToFloat16(c.in); got != c.want {
			t.Errorf("float32ToFloat16(%v) = %#04x, want %#04x", c.in, got, c.want)
		}
	}
}

func TestReflectIndex(t *testing.T) {
	want := map[int]int{-3: 3, -1: 1, 0: 0, 4: 4, 5: 3, 7: 1, 12: 4}
	for i, w := range want {
		if got := reflectIndex(i, 5); got != w {
			t.Errorf("reflectIndex(%d, 5) = %d, want %d", i, got, w)
		}
	}
	if got := reflectIndex(-7, 1); got != 0 {
		t.Errorf("reflectIndex(-7, 1) = %d, want 0", got)
	}
}

// onnxModel builds a minimal ModelProto whose graph has one input with the given dims;
// a string dim is a named (dynamic) dimension.
func onnxModel(dims ...any) []byte {
	var shape []byte
	for _, d := range dims {
		var dim []byte
		switch v := d.(type) {
		case int:
			dim = protowire.AppendTag(dim, 1, protowire.VarintType)
			dim = protowire.AppendVarint(dim, uint64(v))
		case string:
			dim = protowire.AppendTag(dim, 2, protowire.BytesType)
			dim = protowire.AppendString(dim, v)
		}
		shape = protowire.AppendTag(shape, 1, protowire.BytesType)
		shape = protowire.AppendBytes(shape, dim)
	}
	wrap := func(num protowire.Number, msg []byte) []byte {
		return protowire.AppendBytes(protowire.AppendTag(nil, num, protowire.BytesType), msg)
	}
	elemType := protowire.AppendVarint(protowire.AppendTag(nil, 1, protowire.VarintType), 1)
	tensor := append(elemType, wrap(2, shape)...)
	input := append(protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), "input"), wrap(2, wrap(1, tensor))...)
	irVersion := protowire.AppendVarint(protowire.AppendTag(nil, 1, protowire.VarintType), 8)
	return append(irVersion, wrap(7, wrap(11, input))...)
}

func TestWithStaticInputShape(t *testing.T) {
	dims := []int64{1, 3, 256, 256}

	got, err := withStaticInputShape(onnxModel("batch", 3, "height", "width"), dims)
	if err != nil {
		t.Fatal(err)
	}
	if want := onnxModel(1, 3, 256, 256); string(got) != string(want) {
		t.Errorf("dynamic input not pinned:\n got %x\nwant %x", got, want)
	}

	// Fixed-size inputs and non-4D inputs are left alone.
	for _, model := range [][]byte{onnxModel(1, 3, 512, 512), onnxModel("batch", 3)} {
		got, err := withStaticInputShape(model, dims)
		if err != nil || string(got) != string(model) {
			t.Errorf("model changed unexpectedly (err %v)", err)
		}
	}

	if _, err := withStaticInputShape([]byte{0xff}, dims); err == nil {
		t.Error("expected an error for malformed protobuf")
	}
}
