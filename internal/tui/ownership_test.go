package tui

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/termcaps"
)

func TestRevealFailureDoesNotOpenPreviousDownload(t *testing.T) {
	oldDownload, oldReveal := downloadToDownloadsFn, revealFn
	t.Cleanup(func() { downloadToDownloadsFn = oldDownload; revealFn = oldReveal })
	downloadToDownloadsFn = func(model.Result, model.CacheOptions) (string, error) { return "previous.gif", nil }
	revealed := false
	revealFn = func(string) error { revealed = true; return nil }
	state := &appState{results: []model.Result{{ID: "old", URL: "https://example.test/previous.gif"}}}
	downloadSelected(state, bufio.NewWriter(io.Discard), false)
	state.results = []model.Result{{ID: "new", URL: "https://example.test/new.gif"}}
	downloadToDownloadsFn = func(model.Result, model.CacheOptions) (string, error) { return "", errors.New("download failed") }
	handleRevealSelected(state, bufio.NewWriter(io.Discard))
	if revealed {
		t.Fatal("revealed an unrelated previous download")
	}
	if !strings.Contains(state.headerFlash, "Download error") {
		t.Fatalf("lost download error: %s", state.headerFlash)
	}
}

func TestResultIdentityIncludesAssetURL(t *testing.T) {
	a := model.Result{ID: "same", URL: "https://giphy.test/a.gif"}
	b := model.Result{ID: "same", URL: "https://klipy.test/b.gif"}
	if resultKey(a) == resultKey(b) {
		t.Fatal("different provider assets share saved/prefetched state")
	}
}

func TestStalePrefetchKeepsCurrentRequest(t *testing.T) {
	state := &appState{prefetchGen: 2, prefetching: map[string]bool{"current": true}}
	acceptPrefetchResult(state, prefetchResult{key: "current", gen: 1, err: errors.New("cancelled")})
	if !state.prefetching["current"] {
		t.Fatal("old generation removed current request")
	}
}

func TestCleanupCancelsPrefetch(t *testing.T) {
	t.Setenv("GIFGREP_TUI_PREFETCH_MAX_BYTES", "1024")
	started, cancelled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
			close(cancelled)
		case <-release:
		}
	}))
	defer srv.Close()
	defer close(release)
	state := newAppState(termcaps.InlineANSI, model.Options{})
	startPrefetch(state, []model.Result{{URL: srv.URL}}, make(chan prefetchResult))
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	cleanupTempDir(state)
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("cleanup left HTTP request running")
	}
}

func TestRenderKeepsSelectionInVisibleList(t *testing.T) {
	state := &appState{lastRows: 12, lastCols: 60, currentAnim: &gifAnimation{Width: 2, Height: 2}, selected: 5}
	for range 20 {
		state.results = append(state.results, model.Result{Title: "result"})
	}
	render(state, bufio.NewWriter(io.Discard), 12, 60)
	l := buildLayout(state, 12, 60)
	if state.selected < state.scroll || state.selected >= state.scroll+l.listHeight {
		t.Fatalf("selection %d outside scroll=%d height=%d", state.selected, state.scroll, l.listHeight)
	}
}

func TestRenderUsesServingProvider(t *testing.T) {
	t.Setenv("GIPHY_API_KEY", "synthetic")
	state := &appState{source: "klipy", opts: model.Options{Source: "auto"}}
	var b strings.Builder
	out := bufio.NewWriter(&b)
	render(state, out, 24, 80)
	if err := out.Flush(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "Powered by KLIPY") || strings.Contains(b.String(), "Powered by GIPHY") {
		t.Fatal("attribution ignored the provider that served results")
	}
}
