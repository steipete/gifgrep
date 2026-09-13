package app

import (
	"errors"
	"io"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/testutil"
)

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

func TestSearchReportsOutputFailure(t *testing.T) {
	t.Setenv("GIPHY_API_KEY", "synthetic")
	failure := errors.New("output unavailable")
	testutil.WithTransport(t, &testutil.FakeTransport{}, func() {
		for _, format := range []string{"json", "url", "plain", "md", "tsv", "comment"} {
			err := runSearch(errorWriter{failure}, io.Discard, model.Options{Source: "giphy", Format: format}, "cats")
			if !errors.Is(err, failure) {
				t.Errorf("%s: got %v, want output failure", format, err)
			}
		}
	})
}
