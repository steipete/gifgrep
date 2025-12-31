package tui

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
)

func TestRevealSelectedUsesExistingDownload(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "gifgrep-*.gif")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_ = tmp.Close()

	origDownload := downloadToDownloadsFn
	origReveal := revealFn
	t.Cleanup(func() {
		downloadToDownloadsFn = origDownload
		revealFn = origReveal
	})

	downloadCalled := false
	downloadToDownloadsFn = func(model.Result) (string, error) {
		downloadCalled = true
		return "", errors.New("unexpected download")
	}

	var revealed string
	revealFn = func(path string) error {
		revealed = path
		return nil
	}

	state := &appState{
		results:    []model.Result{{ID: "1", URL: "https://example.test/1.gif", Title: "one"}},
		selected:   0,
		lastRows:   24,
		lastCols:   80,
		savedPaths: map[string]string{"id:1": tmp.Name()},
	}

	out := bufio.NewWriter(bytes.NewBuffer(nil))
	_ = handleRevealSelected(state, out)

	if downloadCalled {
		t.Fatalf("expected no download")
	}
	if revealed != tmp.Name() {
		t.Fatalf("expected reveal %q, got %q", tmp.Name(), revealed)
	}
}

func TestRevealSelectedDownloadsWhenMissing(t *testing.T) {
	origDownload := downloadToDownloadsFn
	origReveal := revealFn
	t.Cleanup(func() {
		downloadToDownloadsFn = origDownload
		revealFn = origReveal
	})

	downloadCalled := false
	var downloadedPath string
	downloadToDownloadsFn = func(model.Result) (string, error) {
		downloadCalled = true
		tmp, err := os.CreateTemp(t.TempDir(), "gifgrep-*.gif")
		if err != nil {
			return "", err
		}
		_ = tmp.Close()
		downloadedPath = tmp.Name()
		return downloadedPath, nil
	}

	var revealed string
	revealFn = func(path string) error {
		revealed = path
		return nil
	}

	state := &appState{
		results:    []model.Result{{ID: "1", URL: "https://example.test/1.gif", Title: "one"}},
		selected:   0,
		lastRows:   24,
		lastCols:   80,
		savedPaths: map[string]string{},
	}

	out := bufio.NewWriter(bytes.NewBuffer(nil))
	_ = handleRevealSelected(state, out)

	if !downloadCalled {
		t.Fatalf("expected download")
	}
	if revealed != downloadedPath {
		t.Fatalf("expected reveal %q, got %q", downloadedPath, revealed)
	}
	if got := state.savedPaths["id:1"]; got != downloadedPath {
		t.Fatalf("expected state.savedPaths to contain %q, got %q", downloadedPath, got)
	}
}
