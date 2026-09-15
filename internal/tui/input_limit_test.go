package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steipete/gifgrep/gifdecode"
	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/termcaps"
	"github.com/steipete/gifgrep/internal/testutil"
)

func TestPreviewRejectsOversizedInputs(t *testing.T) {
	data := make([]byte, gifdecode.DefaultOptions().MaxBytes+1)
	testutil.WithTransport(t, &testutil.FakeTransport{GIFData: data}, func() {
		if _, err := fetchGIF("https://example.test/preview.gif"); !errors.Is(err, gifdecode.ErrTooLarge) {
			t.Fatalf("fetch error=%v", err)
		}
	})
	path := filepath.Join(t.TempDir(), "large.gif")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	item := model.Result{URL: "https://example.test/full.gif", PreviewURL: "https://example.test/preview.gif"}
	state := newAppState(termcaps.InlineIterm, model.Options{})
	state.results = []model.Result{item}
	trackSavedPath(state, item, path)
	loadSelectedImage(state)
	if state.currentAnim != nil || !strings.Contains(state.status, gifdecode.ErrTooLarge.Error()) {
		t.Fatalf("oversized local iTerm image accepted: status=%q", state.status)
	}
}
