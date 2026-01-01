package tui

import (
	"bufio"
	"os"

	"github.com/steipete/gifgrep/internal/download"
	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/reveal"
)

var (
	downloadToDownloadsFn = download.ToDownloads
	revealFn              = reveal.Reveal
)

func downloadSelected(state *appState, out *bufio.Writer, revealAfter bool) {
	if state.selected < 0 || state.selected >= len(state.results) {
		flashHeader(state, "No selection")
		state.renderDirty = true
		return
	}
	item := state.results[state.selected]
	if item.URL == "" {
		flashHeader(state, "No URL")
		state.renderDirty = true
		return
	}
	flashHeader(state, "Downloading…")
	state.renderDirty = true
	render(state, out, state.lastRows, state.lastCols)
	_ = out.Flush()

	filePath, err := downloadToDownloadsFn(item)
	if err != nil {
		flashHeader(state, "Download error: "+err.Error())
		state.renderDirty = true
		return
	}
	state.lastSavedPath = filePath
	trackSavedPath(state, item, filePath)
	if revealAfter {
		if err := revealFn(filePath); err != nil {
			flashHeader(state, "Saved (reveal failed)")
			state.renderDirty = true
			return
		}
		flashHeader(state, "Saved (revealed)")
		state.renderDirty = true
		return
	}
	flashHeader(state, "Saved")
	state.renderDirty = true
}

func handleRevealSelected(state *appState, out *bufio.Writer) bool {
	if state.selected < 0 || state.selected >= len(state.results) {
		flashHeader(state, "No selection")
		state.renderDirty = true
		return false
	}
	item := state.results[state.selected]
	if item.URL == "" {
		flashHeader(state, "No URL")
		state.renderDirty = true
		return false
	}

	filePath, ok := savedPathForResult(state, item)
	if !ok {
		downloadSelected(state, out, false)
		filePath = state.lastSavedPath
	}
	if filePath == "" {
		// downloadSelected already set a useful status
		return false
	}

	if err := revealFn(filePath); err != nil {
		flashHeader(state, "Reveal failed: "+err.Error())
		state.renderDirty = true
		return false
	}
	flashHeader(state, "Revealed")
	state.renderDirty = true
	return false
}

func savedPathForResult(state *appState, item model.Result) (string, bool) {
	if state.savedPaths == nil {
		return "", false
	}
	if p, ok := state.savedPaths[resultKey(item)]; ok && p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

func trackSavedPath(state *appState, item model.Result, path string) {
	if path == "" {
		return
	}
	if state.savedPaths == nil {
		state.savedPaths = map[string]string{}
	}
	state.savedPaths[resultKey(item)] = path
}

func resultKey(item model.Result) string {
	if item.ID != "" {
		return "id:" + item.ID
	}
	if item.URL != "" {
		return "url:" + item.URL
	}
	if item.Title != "" {
		return "title:" + item.Title
	}
	return "unknown"
}
