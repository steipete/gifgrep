package stills

import (
	"errors"
	"image/color"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/steipete/gifgrep/gifdecode"
)

func TestContactSheetRejectsOversizedDimensions(t *testing.T) {
	decoded := &gifdecode.Frames{Width: 2, Height: 2, Frames: []gifdecode.Frame{{PNG: makeSolidPNG(color.White, 2, 2)}, {PNG: makeSolidPNG(color.White, 2, 2)}}}
	for _, opts := range []SheetOptions{
		{Count: 1, Columns: math.MaxInt},
		{Count: 2, Columns: 2, Padding: math.MaxInt},
		{Count: 2, Columns: 1, Padding: math.MaxInt},
		{Count: 1, Columns: 40_000_000},
	} {
		if _, err := ContactSheet(decoded, opts); !errors.Is(err, ErrInvalidSheet) {
			t.Fatalf("%+v: got %v", opts, err)
		}
	}
}

func TestSampleEveryFrameWithUnevenDelays(t *testing.T) {
	frames := []gifdecode.Frame{{Delay: time.Second}, {Delay: time.Millisecond}, {Delay: time.Millisecond}, {Delay: time.Millisecond}}
	if got := sampleIndices(frames, len(frames)); !slices.Equal(got, []int{0, 1, 2, 3}) {
		t.Fatalf("all-frame sampling duplicated frames: %v", got)
	}
}
