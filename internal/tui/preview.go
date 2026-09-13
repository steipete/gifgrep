package tui

import (
	"encoding/binary"
	"os"
	"time"

	"github.com/steipete/gifgrep/gifdecode"
	"github.com/steipete/gifgrep/internal/termcaps"
)

func gifSize(raw []byte) (w, h int) {
	if len(raw) < 10 {
		return 0, 0
	}
	hdr := string(raw[:6])
	if hdr != "GIF87a" && hdr != "GIF89a" {
		return 0, 0
	}
	w = int(binary.LittleEndian.Uint16(raw[6:8]))
	h = int(binary.LittleEndian.Uint16(raw[8:10]))
	return w, h
}

func loadSelectedImage(state *appState) {
	if state.cache == nil {
		state.cache = map[string]*gifCacheEntry{}
	}
	if state.selected < 0 || state.selected >= len(state.results) {
		state.currentAnim = nil
		state.previewDirty = true
		return
	}
	item := state.results[state.selected]
	source := item.PreviewURL
	localPath, localOK := savedPathForResult(state, item)
	if localOK {
		source = localPath
	} else if tempPath, ok := tempPathForResult(state, item); ok {
		source = tempPath
		localPath = tempPath
		localOK = true
	}
	if source == "" {
		state.currentAnim = nil
		state.previewDirty = true
		return
	}
	entry, ok := state.cache[source]
	if !ok {
		var data []byte
		var err error
		if localOK {
			data, err = os.ReadFile(localPath)
		} else {
			data, err = fetchGIF(source)
		}
		if err != nil {
			state.status = "Image error: " + err.Error()
			state.currentAnim = nil
			return
		}
		w, h := gifSize(data)
		entry = &gifCacheEntry{RawGIF: data, Width: w, Height: h}
	}
	if entry.Frames == nil && inlineNeedsDecodedFrames(state.inline) {
		decoded, err := gifdecode.Decode(entry.RawGIF, gifdecode.DefaultOptions())
		if err != nil {
			state.status = "Image error: " + err.Error()
			state.currentAnim = nil
			return
		}
		entry.Frames = decoded
		entry.Width = decoded.Width
		entry.Height = decoded.Height
	}

	state.cache[source] = entry

	var frames []gifdecode.Frame
	if entry.Frames != nil {
		frames = entry.Frames.Frames
	}
	state.currentAnim = &gifAnimation{
		ID:     state.nextImageID,
		RawGIF: entry.RawGIF,
		Frames: frames,
		Width:  entry.Width,
		Height: entry.Height,
	}
	state.nextImageID++
	state.ansiFrames = nil
	state.manualAnim = false
	state.manualFrame = 0
	state.manualNext = time.Time{}
	state.previewNeedsSend = true
	state.previewDirty = true
}

func inlineNeedsDecodedFrames(inline termcaps.InlineProtocol) bool {
	return inline == termcaps.InlineKitty || inline == termcaps.InlineSixel || inline == termcaps.InlineANSI
}
