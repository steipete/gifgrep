package tui

import (
	"bufio"
	"os"

	"github.com/steipete/gifgrep/internal/clipboard"
	"github.com/steipete/gifgrep/internal/model"
)

var copyToClipboardFn = clipboard.CopyFile

func copySelected(state *appState, out *bufio.Writer) {
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
	flashHeader(state, "Copying…")
	state.renderDirty = true
	render(state, out, state.lastRows, state.lastCols)
	_ = out.Flush()

	filePath := gifPathForResult(state, item)
	if filePath == "" {
		flashHeader(state, "GIF not available")
		state.renderDirty = true
		return
	}

	if err := copyToClipboardFn(filePath); err != nil {
		flashHeader(state, "Copy failed: "+err.Error())
		state.renderDirty = true
		return
	}
	flashHeader(state, "Copied to clipboard")
	state.renderDirty = true
}

func gifPathForResult(state *appState, item model.Result) string {
	if p, ok := savedPathForResult(state, item); ok {
		return p
	}
	if p, ok := tempPathForResult(state, item); ok {
		return p
	}

	// Write cached raw GIF bytes to a temp file.
	key := resultKey(item)
	if entry, ok := state.cache[item.PreviewURL]; ok && len(entry.RawGIF) > 0 {
		dir, err := ensureTempDir(state)
		if err != nil {
			return ""
		}
		tmp, err := os.CreateTemp(dir, "gifgrep-*.gif")
		if err != nil {
			return ""
		}
		if _, err := tmp.Write(entry.RawGIF); err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			return ""
		}
		_ = tmp.Close()
		if state.tempPaths == nil {
			state.tempPaths = map[string]string{}
		}
		state.tempPaths[key] = tmp.Name()
		return tmp.Name()
	}

	return ""
}
