package gifdecode

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	_ "image/jpeg" // allow image.Decode fallback for stills
	"image/png"
	"io"
	"sync"
	"time"
)

type Frame struct {
	PNG   []byte
	Delay time.Duration
}

type Frames struct {
	Frames []Frame
	Width  int
	Height int
}

var (
	pngEncoder = png.Encoder{CompressionLevel: png.BestSpeed}
	pngPool    = sync.Pool{New: func() any { return new(bytes.Buffer) }}
)

func Decode(data []byte, opts Options) (*Frames, error) {
	return DecodeReader(bytes.NewReader(data), opts)
}

func DecodeReader(r io.Reader, opts Options) (*Frames, error) {
	opts = opts.withDefaults()
	data, err := readAllLimit(r, opts.MaxBytes)
	if err != nil {
		return nil, err
	}
	return decodeBytes(data, opts)
}

func decodeBytes(data []byte, opts Options) (*Frames, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		if opts.StrictGIF {
			return nil, err
		}
		img, _, imgErr := image.Decode(bytes.NewReader(data))
		if imgErr != nil {
			return nil, err
		}
		return singleFrame(img, opts)
	}
	return decodeGIF(g, opts)
}

func decodeGIF(g *gif.GIF, opts Options) (*Frames, error) {
	if len(g.Image) == 0 {
		return nil, ErrNoFrames
	}
	width := g.Config.Width
	height := g.Config.Height
	if width <= 0 || height <= 0 {
		b := g.Image[0].Bounds()
		width, height = b.Dx(), b.Dy()
	}
	if width <= 0 || height <= 0 {
		return nil, ErrInvalidSize
	}
	if exceedsPixels(width, height, opts.MaxPixels) {
		return nil, fmt.Errorf("%w: pixels=%d limit=%d", ErrTooLarge, width*height, opts.MaxPixels)
	}

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	prev := image.NewRGBA(canvas.Bounds())
	bg := backgroundColor(g)

	limit := len(g.Image)
	if opts.MaxFrames > 0 && opts.MaxFrames < limit {
		limit = opts.MaxFrames
	}
	frames := make([]Frame, 0, limit)

	for i := 0; i < limit; i++ {
		frame := g.Image[i]
		disposal := gif.DisposalNone
		if i < len(g.Disposal) {
			disposal = int(g.Disposal[i])
		}
		if disposal == gif.DisposalPrevious {
			copy(prev.Pix, canvas.Pix)
		}

		draw.Draw(canvas, frame.Bounds(), frame, image.Point{}, draw.Over)
		pngData, err := encodePNG(canvas)
		if err != nil {
			return nil, err
		}
		frames = append(frames, Frame{PNG: pngData, Delay: frameDelay(g, i, opts)})

		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, frame.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			copy(canvas.Pix, prev.Pix)
		}
	}

	return &Frames{Frames: frames, Width: width, Height: height}, nil
}

func singleFrame(img image.Image, opts Options) (*Frames, error) {
	b := img.Bounds()
	width, height := b.Dx(), b.Dy()
	if width <= 0 || height <= 0 {
		return nil, ErrInvalidSize
	}
	if exceedsPixels(width, height, opts.MaxPixels) {
		return nil, fmt.Errorf("%w: pixels=%d limit=%d", ErrTooLarge, width*height, opts.MaxPixels)
	}
	pngData, err := encodePNG(img)
	if err != nil {
		return nil, err
	}
	return &Frames{
		Frames: []Frame{{PNG: pngData, Delay: clampDelay(opts.DefaultDelay, opts)}},
		Width:  width,
		Height: height,
	}, nil
}

func frameDelay(g *gif.GIF, idx int, opts Options) time.Duration {
	var delay time.Duration
	if idx < len(g.Delay) {
		delay = time.Duration(g.Delay[idx]) * 10 * time.Millisecond
	}
	if delay <= 0 {
		delay = opts.DefaultDelay
	}
	return clampDelay(delay, opts)
}

func clampDelay(delay time.Duration, opts Options) time.Duration {
	if delay < opts.MinDelay {
		return opts.MinDelay
	}
	if delay > opts.MaxDelay {
		return opts.MaxDelay
	}
	return delay
}

func backgroundColor(g *gif.GIF) color.Color {
	pal, ok := g.Config.ColorModel.(color.Palette)
	if !ok || len(pal) == 0 {
		return color.Transparent
	}
	idx := int(g.BackgroundIndex)
	if idx < 0 || idx >= len(pal) {
		return color.Transparent
	}
	return pal[idx]
}

func encodePNG(img image.Image) ([]byte, error) {
	buf := pngPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer pngPool.Put(buf)
	if err := pngEncoder.Encode(buf, img); err != nil {
		return nil, err
	}
	out := make([]byte, buf.Len())
	copy(out, buf.Bytes())
	return out, nil
}

func readAllLimit(r io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return io.ReadAll(r)
	}
	lr := &io.LimitedReader{R: r, N: maxBytes + 1}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, ErrTooLarge
	}
	return data, nil
}

func exceedsPixels(width, height, maxPixels int) bool {
	if maxPixels <= 0 {
		return false
	}
	pixels := int64(width) * int64(height)
	return pixels > int64(maxPixels)
}
