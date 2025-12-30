package search

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/testutil"
)

func TestFetchTenorAndGIF(t *testing.T) {
	gifData := testutil.MakeTestGIF()
	testutil.WithTransport(t, &testutil.FakeTransport{GIFData: gifData}, func() {
		if _, err := Search("cats", model.Options{Source: "nope"}); err == nil {
			t.Fatalf("expected unknown source error")
		}
		out, err := fetchTenorV1("cats", model.Options{Limit: 1})
		if err != nil {
			t.Fatalf("fetchTenorV1 failed: %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("expected 1 result")
		}
		if out[0].PreviewURL == "" || out[0].URL == "" {
			t.Fatalf("missing URLs")
		}
	})
}

func TestFetchGiphy(t *testing.T) {
	t.Setenv("GIPHY_API_KEY", "test-key")
	gifData := testutil.MakeTestGIF()
	testutil.WithTransport(t, &testutil.FakeTransport{GIFData: gifData}, func() {
		out, err := fetchGiphyV1("cats", model.Options{Limit: 1, Source: "giphy"})
		if err != nil {
			t.Fatalf("fetchGiphyV1 failed: %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("expected 1 result")
		}
		if out[0].PreviewURL == "" || out[0].URL == "" {
			t.Fatalf("missing URLs")
		}

		_, err = Search("cats", model.Options{Limit: 1, Source: "giphy"})
		if err != nil {
			t.Fatalf("Search giphy failed: %v", err)
		}
	})
}

type badTenorTransport struct{}

func (t *badTenorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body := "not-json"
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

type statusTenorTransport struct{}

func (t *statusTenorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(strings.NewReader("oops")),
	}, nil
}

func TestFetchTenorErrors(t *testing.T) {
	testutil.WithTransport(t, &badTenorTransport{}, func() {
		if _, err := fetchTenorV1("cats", model.Options{Limit: 1}); err == nil {
			t.Fatalf("expected json error")
		}
	})
	testutil.WithTransport(t, &statusTenorTransport{}, func() {
		if _, err := fetchTenorV1("cats", model.Options{Limit: 1}); err == nil {
			t.Fatalf("expected status error")
		}
	})
}

type noMediaTransport struct{}

func (t *noMediaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body := `{"results":[{"id":"1","title":"No Media","media":[]},{"id":"2","title":"Gif Only","media":[{"gif":{"url":"https://example.test/full.gif","dims":[10,5]}}]}]}`
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func TestFetchTenorMediaFallbacks(t *testing.T) {
	testutil.WithTransport(t, &noMediaTransport{}, func() {
		results, err := fetchTenorV1("cats", model.Options{Limit: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected one result, got %d", len(results))
		}
		if results[0].PreviewURL == "" {
			t.Fatalf("expected preview fallback")
		}
	})
}
