package app

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/steipete/gifgrep/internal/model"
)

func TestExtractUsesCompleteAnimation(t *testing.T) {
	palette := make(color.Palette, 61)
	for i := range palette {
		palette[i] = color.RGBA{R: uint8(i * 4), A: 255}
	}
	animation := &gif.GIF{Config: image.Config{Width: 2, Height: 2, ColorModel: palette}}
	for i := range palette {
		frame := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
		for j := range frame.Pix {
			frame.Pix[j] = uint8(i)
		}
		animation.Image = append(animation.Image, frame)
		animation.Delay = append(animation.Delay, 10)
	}
	var data bytes.Buffer
	if err := gif.EncodeAll(&data, animation); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(t.TempDir(), "long.gif")
	if err := os.WriteFile(input, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, still := range []bool{true, false} {
		output := filepath.Join(t.TempDir(), "output.png")
		opts := model.Options{GifInput: input, OutPath: output, StillSet: still}
		if still {
			opts.StillAt = 10 * time.Second
		} else {
			opts.StillsCount = 61
			opts.StillsCols = 61
		}
		if err := runExtract(opts); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		x := 0
		if !still {
			x = 120
			if img.Bounds().Dx() != 122 {
				t.Fatalf("sheet omitted frames: %v", img.Bounds())
			}
		}
		r, _, _, _ := img.At(x, 0).RGBA()
		if r != 240*257 {
			t.Fatalf("still=%v: final frame red=%d, want %d", still, r, 240*257)
		}
	}
}

func TestDurationRejectsInvalidNumbers(t *testing.T) {
	for _, raw := range []string{"NaN", "Inf", "+Inf", "-Inf", "1e100", "9223372037", "-1", "-1s"} {
		t.Run(raw, func(t *testing.T) {
			d := DurationValue(time.Second)
			if err := d.UnmarshalText([]byte(raw)); err == nil {
				t.Fatalf("accepted %q as %v", raw, time.Duration(d))
			}
			if d != DurationValue(time.Second) {
				t.Fatal("invalid input changed duration")
			}
		})
	}
}

func TestExtractPreservesLongFrameDelays(t *testing.T) {
	palette := color.Palette{color.RGBA{R: 255, A: 255}, color.RGBA{G: 255, A: 255}, color.RGBA{B: 255, A: 255}, color.White}
	animation := &gif.GIF{Delay: []int{300, 100, 100, 100}}
	for i := range palette {
		frame := image.NewPaletted(image.Rect(0, 0, 1, 1), palette)
		frame.SetColorIndex(0, 0, uint8(i))
		animation.Image = append(animation.Image, frame)
	}
	var data bytes.Buffer
	if err := gif.EncodeAll(&data, animation); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(t.TempDir(), "pauses.gif")
	if err := os.WriteFile(input, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		at    time.Duration
		sheet bool
		want  []color.Color
	}{
		{name: "during pause", at: 1500 * time.Millisecond, want: []color.Color{palette[0]}},
		{name: "exact boundary", at: 3 * time.Second, want: []color.Color{palette[1]}},
		{name: "past end", at: 10 * time.Second, want: []color.Color{palette[3]}},
		{name: "sheet", sheet: true, want: []color.Color{palette[0], palette[1], palette[3]}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "output.png")
			opts := model.Options{GifInput: input, OutPath: output, StillSet: !tc.sheet, StillAt: tc.at}
			if tc.sheet {
				opts.StillsCount = 3
				opts.StillsCols = 3
			}
			if err := runExtract(opts); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(bytes.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}
			for x, want := range tc.want {
				if got := color.NRGBAModel.Convert(img.At(x, 0)); got != color.NRGBAModel.Convert(want) {
					t.Errorf("pixel %d = %v, want %v", x, got, want)
				}
			}
		})
	}
}
