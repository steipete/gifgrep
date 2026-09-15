package imageinput

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/steipete/gifgrep/gifdecode"
)

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestReadLimit(t *testing.T) {
	limit := gifdecode.DefaultOptions().MaxBytes
	for _, size := range []int64{0, limit - 1, limit, limit + 1} {
		data, err := Read(io.LimitReader(zeroReader{}, size))
		if size > limit {
			if !errors.Is(err, gifdecode.ErrTooLarge) || data != nil {
				t.Fatalf("size %d: len=%d err=%v", size, len(data), err)
			}
		} else if err != nil || int64(len(data)) != size {
			t.Fatalf("size %d: len=%d err=%v", size, len(data), err)
		}
	}
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }
func TestReadPropagatesFailure(t *testing.T) {
	failure := errors.New("stream interrupted")
	data, err := Read(io.MultiReader(strings.NewReader("GIF89a"), failingReader{failure}))
	if !errors.Is(err, failure) || data != nil {
		t.Fatalf("data=%q err=%v", data, err)
	}
}
