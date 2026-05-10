package search

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/testutil"
)

func TestFetchKlipyAndGIF(t *testing.T) {
	t.Setenv("KLIPY_API_KEY", "test-key")
	gifData := testutil.MakeTestGIF()
	testutil.WithTransport(t, &testutil.FakeTransport{GIFData: gifData}, func() {
		if _, err := Search("cats", model.Options{Source: "nope"}); err == nil {
			t.Fatalf("expected unknown source error")
		}
		out, err := fetchKlipyV2("cats", model.Options{Limit: 1})
		if err != nil {
			t.Fatalf("fetchKlipyV2 failed: %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("expected 1 result")
		}
		if out[0].PreviewURL == "" || out[0].URL == "" {
			t.Fatalf("missing URLs")
		}
	})
}

type klipyRequestTransport struct {
	t *testing.T
}

func (rt *klipyRequestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.t.Helper()
	if req.URL.Scheme != "https" || req.URL.Host != "api.klipy.com" || req.URL.Path != "/v2/search" {
		rt.t.Fatalf("unexpected Klipy URL: %s", req.URL.String())
	}
	q := req.URL.Query()
	if q.Get("key") != "test-key" || q.Get("q") != "cats" || q.Get("limit") != "7" {
		rt.t.Fatalf("unexpected Klipy query: %s", req.URL.RawQuery)
	}
	if q.Get("contentfilter") != "low" {
		rt.t.Fatalf("expected contentfilter=low, got %q", q.Get("contentfilter"))
	}
	if q.Get("media_filter") == "" {
		rt.t.Fatalf("expected media_filter")
	}

	body := `{"results":[{"id":"1","content_description":"Cat Desc","tags":["cat"],"media_formats":{"mediumgif":{"url":"https://example.test/full.gif","dims":[200,100]},"nanogif":{"url":"https://example.test/preview.gif","dims":[50,25]}}}]}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func TestFetchKlipyRequestAndFallbacks(t *testing.T) {
	t.Setenv("KLIPY_API_KEY", "test-key")
	testutil.WithTransport(t, &klipyRequestTransport{t: t}, func() {
		results, err := fetchKlipyV2("cats", model.Options{Limit: 7})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected one result, got %d", len(results))
		}
		if results[0].Title != "Cat Desc" {
			t.Fatalf("expected content description title fallback, got %q", results[0].Title)
		}
		if results[0].URL != "https://example.test/full.gif" || results[0].PreviewURL != "https://example.test/preview.gif" {
			t.Fatalf("unexpected URLs: %#v", results[0])
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

type giphyUnauthorizedTransport struct {
	fallback testutil.FakeTransport
}

func (t *giphyUnauthorizedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "api.giphy.com" {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(strings.NewReader("unauthorized")),
		}, nil
	}
	return t.fallback.RoundTrip(req)
}

func TestAutoFallsBackToKlipyWhenGiphyFails(t *testing.T) {
	t.Setenv("GIPHY_API_KEY", "bad-key")
	t.Setenv("KLIPY_API_KEY", "test-key")
	gifData := testutil.MakeTestGIF()
	testutil.WithTransport(t, &giphyUnauthorizedTransport{fallback: testutil.FakeTransport{GIFData: gifData}}, func() {
		out, err := Search("cats", model.Options{Limit: 1, Source: "auto"})
		if err != nil {
			t.Fatalf("Search auto failed: %v", err)
		}
		if len(out) != 1 || !strings.Contains(out[0].URL, "example.test") {
			t.Fatalf("expected Klipy fallback result, got %#v", out)
		}

		if _, err := Search("cats", model.Options{Limit: 1, Source: "giphy"}); err == nil {
			t.Fatalf("expected explicit giphy to return auth error")
		}
	})
}

type badKlipyTransport struct{}

func (t *badKlipyTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	body := "not-json"
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

type statusKlipyTransport struct{}

func (t *statusKlipyTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("oops")),
	}, nil
}

func TestFetchKlipyErrors(t *testing.T) {
	testutil.WithTransport(t, &badKlipyTransport{}, func() {
		t.Setenv("KLIPY_API_KEY", "test-key")
		if _, err := fetchKlipyV2("cats", model.Options{Limit: 1}); err == nil {
			t.Fatalf("expected json error")
		}
	})
	testutil.WithTransport(t, &statusKlipyTransport{}, func() {
		t.Setenv("KLIPY_API_KEY", "test-key")
		if _, err := fetchKlipyV2("cats", model.Options{Limit: 1}); err == nil {
			t.Fatalf("expected status error")
		}
	})
}

type noMediaTransport struct{}

func (t *noMediaTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	body := `{"results":[{"id":"1","title":"No Media","media_formats":{}},{"id":"2","title":"Gif Only","media_formats":{"gif":{"url":"https://example.test/full.gif","dims":[10,5]}}}]}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func TestFetchKlipyMediaFallbacks(t *testing.T) {
	t.Setenv("KLIPY_API_KEY", "test-key")
	testutil.WithTransport(t, &noMediaTransport{}, func() {
		results, err := fetchKlipyV2("cats", model.Options{Limit: 2})
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

func TestFetchKlipyMissingKey(t *testing.T) {
	t.Setenv("KLIPY_API_KEY", "")
	if _, err := fetchKlipyV2("cats", model.Options{Limit: 1}); err == nil || !strings.Contains(err.Error(), "KLIPY_API_KEY") {
		t.Fatalf("expected missing KLIPY_API_KEY error, got %v", err)
	}
}

func TestResolveSource(t *testing.T) {
	t.Setenv("GIPHY_API_KEY", "")
	if got := ResolveSource("tenor"); got != "klipy" {
		t.Fatalf("expected tenor alias to resolve to klipy, got %q", got)
	}
	if got := ResolveSource("auto"); got != "klipy" {
		t.Fatalf("expected auto fallback to klipy, got %q", got)
	}

	t.Setenv("GIPHY_API_KEY", "test-key")
	if got := ResolveSource("auto"); got != "giphy" {
		t.Fatalf("expected auto with giphy key to resolve to giphy, got %q", got)
	}
}
