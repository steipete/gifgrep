package gifdecode

import (
	"bytes"
	"errors"
	"math"
	"testing"
)

func TestDecodeChecksDimensionsBeforeImageData(t *testing.T) {
	// A complete GIF header/palette followed by no frames: dimensions already exceed the limit.
	data := []byte{'G', 'I', 'F', '8', '9', 'a', 255, 255, 255, 255, 0x80, 0, 0, 0, 0, 0, 255, 255, 255}
	if _, err := Decode(data, DefaultOptions()); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("got %v, want dimension rejection before frame decoding", err)
	}
}

func TestReadAllLimitMaxInt64(t *testing.T) {
	want := []byte("GIF89a")
	got, err := readAllLimit(bytes.NewReader(want), math.MaxInt64)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("got %q, %v", got, err)
	}
}
