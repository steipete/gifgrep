package tui

import (
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/steipete/gifgrep/gifdecode"
	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/termcaps"
	"github.com/steipete/gifgrep/internal/testutil"
)

func TestLoadSelectedImageEdges(t *testing.T) {
	state := &appState{
		results: []model.Result{},
		inline:  termcaps.InlineKitty,
		cache:   map[string]*gifCacheEntry{},
	}
	loadSelectedImage(state)
	if state.currentAnim != nil {
		t.Fatalf("expected nil animation for empty results")
	}

	state.results = []model.Result{{Title: "no preview"}}
	state.selected = 0
	loadSelectedImage(state)
	if state.currentAnim != nil {
		t.Fatalf("expected nil animation for empty preview url")
	}

	state.cache["https://example.test/preview.gif"] = &gifCacheEntry{
		RawGIF: []byte("GIF89a\x01\x00\x01\x00"),
		Frames: &gifdecode.Frames{
			Frames: []gifdecode.Frame{{PNG: []byte{1, 2, 3}, Delay: 80 * time.Millisecond}},
			Width:  1,
			Height: 1,
		},
		Width:  1,
		Height: 1,
	}
	state.results = []model.Result{{Title: "cached", PreviewURL: "https://example.test/preview.gif"}}
	loadSelectedImage(state)
	if state.currentAnim == nil || !state.previewNeedsSend {
		t.Fatalf("expected cached animation")
	}

	badTransport := &testutil.FakeTransport{GIFData: []byte("not-a-gif")}
	testutil.WithTransport(t, badTransport, func() {
		state.cache = map[string]*gifCacheEntry{}
		state.results = []model.Result{{Title: "bad", PreviewURL: "https://example.test/preview.gif"}}
		state.selected = 0
		loadSelectedImage(state)
		if state.currentAnim != nil {
			t.Fatalf("expected nil animation on decode error")
		}
	})
}

type errTransport struct{}

func (t *errTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, errors.New("network")
}

func TestFetchGIFError(t *testing.T) {
	testutil.WithTransport(t, &errTransport{}, func() {
		if _, err := fetchGIF("https://example.test/preview.gif"); err == nil {
			t.Fatalf("expected fetch error")
		}
	})
}

func TestLoadSelectedImageUsesDownloadedFile(t *testing.T) {
	data := testutil.MakeTestGIF()
	tmp, err := os.CreateTemp(t.TempDir(), "gifgrep-*.gif")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	if _, err := tmp.Write(data); err != nil {
		t.Fatalf("write temp gif: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close temp gif: %v", err)
	}

	state := &appState{
		results: []model.Result{{
			ID:         "id1",
			Title:      "cached",
			URL:        "https://example.test/full.gif",
			PreviewURL: "https://example.test/preview.gif",
		}},
		selected:   0,
		inline:     termcaps.InlineKitty,
		cache:      map[string]*gifCacheEntry{},
		savedPaths: map[string]string{"url:https://example.test/full.gif": tmp.Name()},
	}
	testutil.WithTransport(t, &errTransport{}, func() {
		loadSelectedImage(state)
	})
	if state.currentAnim == nil || len(state.currentAnim.RawGIF) == 0 {
		t.Fatalf("expected animation from downloaded file")
	}
}

func TestLoadSelectedImageDecodesFramesForSixel(t *testing.T) {
	data := testutil.MakeTestGIF()
	state := &appState{
		results: []model.Result{{
			ID:         "id1",
			Title:      "cached",
			PreviewURL: "https://example.test/preview.gif",
		}},
		selected: 0,
		inline:   termcaps.InlineSixel,
		cache:    map[string]*gifCacheEntry{},
	}
	testutil.WithTransport(t, &testutil.FakeTransport{GIFData: data}, func() {
		loadSelectedImage(state)
	})
	if state.currentAnim == nil || len(state.currentAnim.Frames) == 0 {
		t.Fatalf("expected decoded Sixel frames")
	}
	if !state.previewNeedsSend || !state.previewDirty {
		t.Fatalf("expected Sixel preview to need send")
	}
}

func TestLoadSelectedImageUsesTempFile(t *testing.T) {
	data := testutil.MakeTestGIF()
	tmp, err := os.CreateTemp(t.TempDir(), "gifgrep-*.gif")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	if _, err := tmp.Write(data); err != nil {
		t.Fatalf("write temp gif: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close temp gif: %v", err)
	}

	state := &appState{
		results: []model.Result{{
			ID:         "id1",
			Title:      "cached",
			URL:        "https://example.test/full.gif",
			PreviewURL: "https://example.test/preview.gif",
		}},
		selected:   0,
		inline:     termcaps.InlineKitty,
		cache:      map[string]*gifCacheEntry{},
		tempPaths:  map[string]string{"url:https://example.test/full.gif": tmp.Name()},
		savedPaths: map[string]string{},
	}
	testutil.WithTransport(t, &errTransport{}, func() {
		loadSelectedImage(state)
	})
	if state.currentAnim == nil || len(state.currentAnim.RawGIF) == 0 {
		t.Fatalf("expected animation from temp file")
	}
}
