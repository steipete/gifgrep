package tui

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/termcaps"
	"github.com/steipete/gifgrep/internal/testutil"
)

func TestCopySelectedUsesTempPath(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "gifgrep-*.gif")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_ = tmp.Close()

	orig := copyToClipboardFn
	t.Cleanup(func() { copyToClipboardFn = orig })

	var copied string
	copyToClipboardFn = func(path string) error {
		copied = path
		return nil
	}

	state := &appState{
		results:   []model.Result{{ID: "1", URL: "https://example.test/1.gif", Title: "one"}},
		selected:  0,
		lastRows:  24,
		lastCols:  80,
		tempPaths: map[string]string{"id:1": tmp.Name()},
		cache:     map[string]*gifCacheEntry{},
	}

	out := bufio.NewWriter(bytes.NewBuffer(nil))
	copySelected(state, out)

	if copied != tmp.Name() {
		t.Fatalf("expected copy %q, got %q", tmp.Name(), copied)
	}
	if state.headerFlash != "Copied to clipboard" {
		t.Fatalf("expected flash 'Copied to clipboard', got %q", state.headerFlash)
	}
}

func TestCopySelectedUsesSavedPath(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "gifgrep-*.gif")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_ = tmp.Close()

	orig := copyToClipboardFn
	t.Cleanup(func() { copyToClipboardFn = orig })

	var copied string
	copyToClipboardFn = func(path string) error {
		copied = path
		return nil
	}

	state := &appState{
		results:    []model.Result{{ID: "1", URL: "https://example.test/1.gif", Title: "one"}},
		selected:   0,
		lastRows:   24,
		lastCols:   80,
		savedPaths: map[string]string{"id:1": tmp.Name()},
		tempPaths:  map[string]string{},
		cache:      map[string]*gifCacheEntry{},
	}

	out := bufio.NewWriter(bytes.NewBuffer(nil))
	copySelected(state, out)

	if copied != tmp.Name() {
		t.Fatalf("expected copy %q, got %q", tmp.Name(), copied)
	}
}

func TestCopySelectedWritesCacheToTemp(t *testing.T) {
	data := testutil.MakeTestGIF()
	orig := copyToClipboardFn
	t.Cleanup(func() { copyToClipboardFn = orig })

	var copied string
	copyToClipboardFn = func(path string) error {
		copied = path
		return nil
	}

	state := &appState{
		results:    []model.Result{{ID: "1", URL: "https://example.test/1.gif", PreviewURL: "https://example.test/preview.gif", Title: "one"}},
		selected:   0,
		lastRows:   24,
		lastCols:   80,
		savedPaths: map[string]string{},
		tempPaths:  map[string]string{},
		tempDir:    t.TempDir(),
		inline:     termcaps.InlineKitty,
	}
	testutil.WithTransport(t, &testutil.FakeTransport{GIFData: data}, func() {
		loadSelectedImage(state)
	})
	if state.currentAnim == nil {
		t.Fatal("expected a loaded preview")
	}

	out := bufio.NewWriter(bytes.NewBuffer(nil))
	copySelected(state, out)

	if copied == "" {
		t.Fatal("expected non-empty path")
	}
	copiedData, err := os.ReadFile(copied)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(copiedData, data) {
		t.Fatal("clipboard did not receive the loaded preview")
	}
	if state.headerFlash != "Copied to clipboard" {
		t.Fatalf("expected flash 'Copied to clipboard', got %q", state.headerFlash)
	}
}

func TestCopySelectedHandlesError(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "gifgrep-*.gif")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_ = tmp.Close()

	orig := copyToClipboardFn
	t.Cleanup(func() { copyToClipboardFn = orig })

	copyToClipboardFn = func(string) error {
		return errors.New("clipboard broken")
	}

	state := &appState{
		results:   []model.Result{{ID: "1", URL: "https://example.test/1.gif", Title: "one"}},
		selected:  0,
		lastRows:  24,
		lastCols:  80,
		tempPaths: map[string]string{"id:1": tmp.Name()},
		cache:     map[string]*gifCacheEntry{},
	}

	out := bufio.NewWriter(bytes.NewBuffer(nil))
	copySelected(state, out)

	if state.headerFlash != "Copy failed: clipboard broken" {
		t.Fatalf("expected error flash, got %q", state.headerFlash)
	}
}

func TestCopySelectedNoSelection(t *testing.T) {
	state := &appState{
		results:  []model.Result{},
		selected: 0,
		lastRows: 24,
		lastCols: 80,
		cache:    map[string]*gifCacheEntry{},
	}

	out := bufio.NewWriter(bytes.NewBuffer(nil))
	copySelected(state, out)

	if state.headerFlash != "No selection" {
		t.Fatalf("expected 'No selection' flash, got %q", state.headerFlash)
	}
}

func TestCopySelectedNoURL(t *testing.T) {
	state := &appState{
		results:  []model.Result{{ID: "1", Title: "one"}},
		selected: 0,
		lastRows: 24,
		lastCols: 80,
		cache:    map[string]*gifCacheEntry{},
	}

	out := bufio.NewWriter(bytes.NewBuffer(nil))
	copySelected(state, out)

	if state.headerFlash != "No URL" {
		t.Fatalf("expected 'No URL' flash, got %q", state.headerFlash)
	}
}

func TestCopySelectedNoAvailablePath(t *testing.T) {
	state := &appState{
		results:    []model.Result{{ID: "1", URL: "https://example.test/1.gif", Title: "one"}},
		selected:   0,
		lastRows:   24,
		lastCols:   80,
		savedPaths: map[string]string{},
		tempPaths:  map[string]string{},
		cache:      map[string]*gifCacheEntry{},
	}

	out := bufio.NewWriter(bytes.NewBuffer(nil))
	copySelected(state, out)

	if state.headerFlash != "GIF not available" {
		t.Fatalf("expected 'GIF not available' flash, got %q", state.headerFlash)
	}
}
