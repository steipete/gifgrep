package search

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/testutil"
)

type failingTransport struct{ err error }

func (t failingTransport) RoundTrip(*http.Request) (*http.Response, error) { return nil, t.err }

func TestTransportErrorDoesNotExposeSearchCredentials(t *testing.T) {
	const key = "synthetic-secret-value"
	t.Setenv("GIPHY_API_KEY", key)
	t.Setenv("KLIPY_API_KEY", key)
	failure := errors.New("connection refused")
	testutil.WithTransport(t, failingTransport{failure}, func() {
		for _, source := range []string{"giphy", "klipy", "auto"} {
			_, _, err := Search("private-query", model.Options{Source: source})
			if !errors.Is(err, failure) {
				t.Fatalf("%s: lost transport cause: %v", source, err)
			}
			if strings.Contains(err.Error(), key) || strings.Contains(err.Error(), "private-query") {
				t.Errorf("%s: error exposes query parameters", source)
			}
		}
	})
}
