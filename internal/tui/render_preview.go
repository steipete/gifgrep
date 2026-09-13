package tui

import (
	"bufio"
	"time"

	"github.com/steipete/gifgrep/gifdecode"
	"github.com/steipete/gifgrep/internal/ansi"
	"github.com/steipete/gifgrep/internal/iterm"
	"github.com/steipete/gifgrep/internal/kitty"
	sixelgfx "github.com/steipete/gifgrep/internal/sixel"
	"github.com/steipete/gifgrep/internal/termcaps"
)

func drawPreview(state *appState, out *bufio.Writer, cols, rows int, row, col int) {
	if state.currentAnim == nil {
		return
	}
	if state.inline == termcaps.InlineIterm {
		if len(state.currentAnim.RawGIF) == 0 {
			return
		}
		if !state.previewNeedsSend && !state.previewDirty && state.lastPreview.cols == cols && state.lastPreview.rows == rows {
			return
		}

		hasPrev := state.itermLast.cols > 0 && state.itermLast.rows > 0
		sameRect := hasPrev &&
			row == state.itermLast.row &&
			col == state.itermLast.col &&
			cols == state.itermLast.cols &&
			rows == state.itermLast.rows

		// iTerm inline images are in the text grid. Always clear the target rect before
		// sending so letterboxing doesn't show stale text.
		if sameRect {
			clearItermRectFn(out, row, col, cols, rows)
		} else {
			if hasPrev {
				clearItermRectFn(out, state.itermLast.row, state.itermLast.col, state.itermLast.cols, state.itermLast.rows)
			}
			clearItermRectFn(out, row, col, cols, rows)
		}
		saveCursor(out)
		moveCursor(out, row, col)
		iterm.SendInlineFile(out, iterm.File{
			Name:        "gifgrep.gif",
			Data:        state.currentAnim.RawGIF,
			WidthCells:  cols,
			HeightCells: rows,
		})
		restoreCursor(out)
		state.previewNeedsSend = false
		state.previewDirty = false
		state.lastPreview.cols = cols
		state.lastPreview.rows = rows
		state.itermLast.row = row
		state.itermLast.col = col
		state.itermLast.cols = cols
		state.itermLast.rows = rows
		return
	}
	if len(state.currentAnim.Frames) == 0 {
		return
	}
	if state.useSoftwareAnim {
		drawPreviewSoftware(state, out, cols, rows, row, col)
		return
	}
	if state.previewNeedsSend {
		if state.activeImageID != 0 {
			kitty.DeleteImage(out, state.activeImageID)
		}
		state.activeImageID = state.currentAnim.ID
		kitty.SendAnimation(out, state.currentAnim.ID, state.currentAnim.Frames, cols, rows)
		state.previewNeedsSend = false
		state.previewDirty = false
		state.lastPreview.cols = cols
		state.lastPreview.rows = rows
		return
	}
	if state.previewDirty || state.lastPreview.cols != cols || state.lastPreview.rows != rows {
		kitty.PlaceImage(out, state.activeImageID, cols, rows)
		state.previewDirty = false
		state.lastPreview.cols = cols
		state.lastPreview.rows = rows
	}
}

func drawPreviewSoftware(state *appState, out *bufio.Writer, cols, rows int, row, col int) {
	if state.currentAnim == nil || len(state.currentAnim.Frames) == 0 {
		return
	}
	if state.inline == termcaps.InlineKitty && state.activeImageID != 0 && state.activeImageID != state.currentAnim.ID {
		kitty.DeleteImage(out, state.activeImageID)
	}
	state.activeImageID = state.currentAnim.ID
	if state.previewNeedsSend {
		state.manualAnim = true
		state.manualFrame = 0
		frame := state.currentAnim.Frames[state.manualFrame]
		sendPreviewFrame(state, out, frame, state.manualFrame, cols, rows, row, col)
		state.manualNext = time.Now().Add(frame.Delay)
		state.previewNeedsSend = false
		state.previewDirty = false
		state.lastPreview.cols = cols
		state.lastPreview.rows = rows
		return
	}
	if state.previewDirty || state.lastPreview.cols != cols || state.lastPreview.rows != rows {
		frame := state.currentAnim.Frames[state.manualFrame]
		sendPreviewFrame(state, out, frame, state.manualFrame, cols, rows, row, col)
		state.previewDirty = false
		state.lastPreview.cols = cols
		state.lastPreview.rows = rows
	}
}

var (
	sendSixelFrameFn  = sixelgfx.SendFrame
	renderANSIFrameFn = ansi.RenderFrame
)

func sendPreviewFrame(state *appState, out *bufio.Writer, frame gifdecode.Frame, frameIndex int, cols, rows int, row, col int) {
	if state.inline == termcaps.InlineSixel || state.inline == termcaps.InlineANSI {
		clearItermRectFn(out, row, col, cols, rows)
	}
	saveCursor(out)
	moveCursor(out, row, col)
	switch state.inline {
	case termcaps.InlineSixel:
		_ = sendSixelFrameFn(out, frame, cols, rows)
	case termcaps.InlineANSI:
		data, err := cachedANSIFrame(state, frame, frameIndex, cols, rows)
		if err == nil {
			_, _ = out.Write(data)
		}
	case termcaps.InlineKitty:
		kitty.SendFrame(out, state.activeImageID, frame, cols, rows)
	case termcaps.InlineNone, termcaps.InlineIterm:
	}
	restoreCursor(out)
}

func cachedANSIFrame(state *appState, frame gifdecode.Frame, frameIndex int, cols, rows int) ([]byte, error) {
	if state.ansiFrames == nil {
		state.ansiFrames = map[ansiFrameKey][]byte{}
	}
	key := ansiFrameKey{animID: state.activeImageID, frame: frameIndex, cols: cols, rows: rows}
	if data, ok := state.ansiFrames[key]; ok {
		return data, nil
	}
	data, err := renderANSIFrameFn(frame.PNG, cols, rows)
	if err != nil {
		return nil, err
	}
	state.ansiFrames[key] = data
	return data, nil
}

func advanceManualAnimation(state *appState, out *bufio.Writer) {
	if !state.manualAnim || state.currentAnim == nil {
		return
	}
	if len(state.currentAnim.Frames) <= 1 {
		return
	}
	if state.lastPreview.cols == 0 || state.lastPreview.rows == 0 {
		return
	}
	if state.manualNext.IsZero() || state.previewRow == 0 || state.previewCol == 0 {
		return
	}
	now := time.Now()
	if now.Before(state.manualNext) {
		return
	}
	state.manualFrame = (state.manualFrame + 1) % len(state.currentAnim.Frames)
	frame := state.currentAnim.Frames[state.manualFrame]
	sendPreviewFrame(state, out, frame, state.manualFrame, state.lastPreview.cols, state.lastPreview.rows, state.previewRow, state.previewCol)
	state.manualNext = now.Add(frame.Delay)
	_ = out.Flush()
}
