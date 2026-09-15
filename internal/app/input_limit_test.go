package app

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/steipete/gifgrep/gifdecode"
	"github.com/steipete/gifgrep/internal/testutil"
)

type oversizedInput struct {
	remaining int64
	read      int64
	closed    bool
}

func (r *oversizedInput) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := min(int64(len(p)), r.remaining)
	clear(p[:n])
	r.remaining -= n
	r.read += n
	return int(n), nil
}
func (r *oversizedInput) Close() error { r.closed = true; return nil }

type inputTransport struct{ body *oversizedInput }

func (tr inputTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: tr.body, ContentLength: -1}, nil
}

func TestReadInputRejectsOversizedHTTPWhileReading(t *testing.T) {
	limit := gifdecode.DefaultOptions().MaxBytes
	body := &oversizedInput{remaining: limit * 2}
	testutil.WithTransport(t, inputTransport{body}, func() {
		data, err := readInput("https://example.test/large.gif")
		if !errors.Is(err, gifdecode.ErrTooLarge) {
			t.Errorf("error=%v, want ErrTooLarge", err)
		}
		if len(data) != 0 {
			t.Errorf("returned %d bytes after oversized input", len(data))
		}
	})
	if body.read > limit+1 {
		t.Errorf("read %d bytes, limit is %d", body.read, limit)
	}
	if !body.closed {
		t.Error("response body not closed")
	}
}

func TestReadInputRejectsOversizedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.gif")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(gifdecode.DefaultOptions().MaxBytes + 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{path, "file://" + path} {
		if _, err := readInput(input); !errors.Is(err, gifdecode.ErrTooLarge) {
			t.Errorf("error=%v, want ErrTooLarge", err)
		}
	}
}
