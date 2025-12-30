package testutil

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"net/http"
	"strings"
	"testing"
)

type FakeTransport struct {
	GIFData []byte
}

func (t *FakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Host {
	case "api.tenor.com":
		body := `{"results":[{"id":"1","title":"Cat One","content_description":"","tags":["cat","fun"],"media":[{"gif":{"url":"https://example.test/full.gif","dims":[200,100]},"tinygif":{"url":"https://example.test/preview.gif","dims":[50,25]}}]}]}`
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	case "api.giphy.com":
		body := `{"data":[{"id":"g1","title":"Cat One","images":{"original":{"url":"https://example.test/full.gif","width":"200","height":"100"},"fixed_width_small":{"url":"https://example.test/preview.gif","width":"50","height":"25"}}}]}`
		return &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	case "example.test":
		if req.URL.Path == "/preview.gif" || req.URL.Path == "/full.gif" {
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{"Content-Type": []string{"image/gif"}},
				Body:       io.NopCloser(bytes.NewReader(t.GIFData)),
			}, nil
		}
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader("not found")),
		}, nil
	default:
		return nil, fmt.Errorf("unexpected host: %s", req.URL.Host)
	}
}

func WithTransport(t *testing.T, rt http.RoundTripper, fn func()) {
	t.Helper()
	prev := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() {
		http.DefaultTransport = prev
	})
	fn()
}

func MakeTestGIF() []byte {
	pal := color.Palette{color.Black, color.White}
	frame1 := image.NewPaletted(image.Rect(0, 0, 2, 2), pal)
	frame2 := image.NewPaletted(image.Rect(0, 0, 2, 2), pal)
	frame1.SetColorIndex(0, 0, 1)
	frame2.SetColorIndex(1, 1, 1)

	g := &gif.GIF{
		Image:    []*image.Paletted{frame1, frame2},
		Delay:    []int{5, 7},
		Disposal: []byte{gif.DisposalNone, gif.DisposalBackground},
		Config: image.Config{
			Width:      2,
			Height:     2,
			ColorModel: pal,
		},
	}
	var buf bytes.Buffer
	_ = gif.EncodeAll(&buf, g)
	return buf.Bytes()
}
