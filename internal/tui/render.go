package tui

import (
	"bufio"
	"fmt"
	"strings"
	"time"

	"github.com/steipete/gifgrep/gifdecode"
	"github.com/steipete/gifgrep/internal/assets"
	"github.com/steipete/gifgrep/internal/kitty"
	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/search"
	"github.com/steipete/gifgrep/internal/termcaps"
)

const giphyAttributionImageID uint32 = 0x67697068 // "giph"

func render(state *appState, out *bufio.Writer, rows, cols int) {
	if rows <= 0 || cols <= 0 {
		return
	}

	layout := buildLayout(state, rows, cols)

	if state.inline == termcaps.InlineIterm && layout.showRight && shouldSendItermPreview(state, layout) {
		if shouldHardClearIterm(state, layout) {
			clearItermScreenFn(out)
			state.itermLast = struct {
				row  int
				col  int
				cols int
				rows int
			}{}
		} else {
			eraseItermContentAreaFn(out, layout)
		}
	}

	if state.currentAnim == nil && state.activeImageID != 0 {
		if state.inline == termcaps.InlineKitty {
			kitty.DeleteImage(out, state.activeImageID)
		}
		state.activeImageID = 0
	}

	if !state.headerFlashAt.IsZero() && nowFn().After(state.headerFlashAt) {
		state.headerFlash = ""
		state.headerFlashAt = time.Time{}
	}
	headerTagline := state.tagline
	if strings.TrimSpace(state.headerFlash) != "" {
		headerTagline = state.headerFlash
	}
	drawHeader(out, state.useColor, cols, headerTagline)

	if !layout.hasContent {
		clearAll(out, rows, cols)
		return
	}

	if layout.clearWidth > 0 {
		// When switching from bottom-preview to split-preview, clear the left area once
		// so old list rows don't show through. For Kitty, it's cheap to clear every render;
		// for iTerm images (inline in the text grid), clearing would erase the image.
		if state.inline == termcaps.InlineKitty || !state.lastShowRight {
			clearPreviewAreaFn(out, layout)
		}
	}

	state.lastShowRight = layout.showRight

	drawList(out, state, layout)
	if state.inline == termcaps.InlineIterm && layout.showRight {
		clearItermGapColumn(out, layout)
	}
	drawPreviewIfNeeded(out, state, layout)
	drawStatus(out, state, layout)
	drawSearch(out, state, layout)
	drawHints(out, state, layout)
	clearUnused(out, layout)
}

func drawHeader(out *bufio.Writer, useColor bool, cols int, tagline string) {
	header := styleIf(useColor, "gifgrep", "\x1b[1m", "\x1b[36m")
	if strings.TrimSpace(tagline) == "" {
		tagline = model.Tagline
	}
	codes := []string{"\x1b[90m"}
	if useColor && strings.TrimSpace(tagline) != "" && tagline != model.Tagline {
		// Likely an action flash; make it pop a bit.
		codes = []string{"\x1b[33m"}
	}
	header += styleIf(useColor, " — "+tagline, codes...)
	writeLineAt(out, 1, 1, header, cols)
}

func clearAll(out *bufio.Writer, rows, cols int) {
	for row := 1; row <= rows; row++ {
		writeLineAt(out, row, 1, "", cols)
	}
}

func clearPreviewArea(out *bufio.Writer, layout layout) {
	for i := 0; i < layout.contentHeight; i++ {
		writeLineAt(out, layout.contentTop+i, 1, "", layout.clearWidth)
	}
}

var clearPreviewAreaFn = clearPreviewArea

func drawList(out *bufio.Writer, state *appState, layout layout) {
	for i := 0; i < layout.listHeight; i++ {
		idx := state.scroll + i
		if idx >= 0 && idx < len(state.results) {
			item := state.results[idx]
			label := item.Title
			if label == "" {
				label = item.ID
			}
			prefix := "  "
			if idx == state.selected {
				prefix = styleIf(state.useColor, "> ", "\x1b[1m", "\x1b[36m")
				label = styleIf(state.useColor, label, "\x1b[1m")
			}
			writeLineAt(out, layout.contentTop+i, layout.listCol, prefix+label, layout.listWidth)
		} else {
			writeLineAt(out, layout.contentTop+i, layout.listCol, "", layout.listWidth)
		}
	}
}

func drawPreviewIfNeeded(out *bufio.Writer, state *appState, layout layout) {
	if state.currentAnim == nil || layout.previewCols <= 0 || layout.previewRows <= 0 {
		return
	}
	if layout.showRight {
		state.previewCol = 1
		state.previewRow = layout.previewRow
		moveCursor(out, state.previewRow, state.previewCol)
		drawPreview(state, out, layout.previewCols, layout.previewRows, state.previewRow, state.previewCol)
		return
	}

	label := styleIf(state.useColor, "Preview", "\x1b[90m")
	writeLineAt(out, layout.contentTop+layout.listHeight, 1, label, layout.cols)
	state.previewRow = layout.previewRow
	state.previewCol = layout.previewCol
	for i := 0; i < layout.previewRows; i++ {
		writeLineAt(out, state.previewRow+i, 1, "", layout.cols)
	}
	moveCursor(out, state.previewRow, state.previewCol)
	drawPreview(state, out, layout.previewCols, layout.previewRows, state.previewRow, state.previewCol)
}

func drawStatus(out *bufio.Writer, state *appState, layout layout) {
	status := state.status
	if status == "" {
		status = fmt.Sprintf("%d results", len(state.results))
	}
	source := search.ResolveSource(state.opts.Source)
	showGiphyAttribution := source == "giphy"
	showKlipyAttribution := source == "klipy"
	showGiphyIcon := showGiphyAttribution && state.inline == termcaps.InlineKitty
	logoCols := 2
	logoRows := 1
	statusWidth := layout.cols
	if showGiphyIcon {
		statusWidth = max(0, layout.cols-(logoCols+1))
	}
	line := formatStatusLine(state.useColor, status)
	if showGiphyAttribution {
		line += styleIf(state.useColor, " · Powered by GIPHY", "\x1b[90m")
	} else if showKlipyAttribution {
		line += styleIf(state.useColor, " · Powered by KLIPY", "\x1b[90m")
	}
	writeLineAt(out, layout.statusRow, 1, line, statusWidth)
	if showGiphyIcon && layout.cols >= logoCols {
		moveCursor(out, layout.statusRow, max(1, layout.cols-logoCols+1))
		kitty.SendFrame(out, giphyAttributionImageID, gifdecode.Frame{PNG: assets.GiphyIcon32PNG()}, logoCols, logoRows)
		state.giphyAttributionShown = true
	} else if state.giphyAttributionShown && state.inline == termcaps.InlineKitty {
		kitty.DeleteImage(out, giphyAttributionImageID)
		state.giphyAttributionShown = false
	}
}

func formatStatusLine(useColor bool, status string) string {
	if !useColor {
		return status
	}
	i := 0
	for i < len(status) && status[i] >= '0' && status[i] <= '9' {
		i++
	}
	if i > 0 && strings.HasPrefix(status[i:], " results") {
		num := status[:i]
		rest := status[i:]
		return styleIf(true, num, "\x1b[1m", "\x1b[36m") + styleIf(true, rest, "\x1b[90m")
	}
	return styleIf(true, status, "\x1b[90m")
}

func drawSearch(out *bufio.Writer, state *appState, layout layout) {
	label := "Search"
	if search.ResolveSource(state.opts.Source) == "klipy" {
		label = "Search KLIPY"
	}
	pill := "[" + label + "]"
	query := state.query
	if state.useColor {
		bg := "\x1b[48;5;236m"
		if state.mode == modeQuery {
			pill = styleIf(true, " "+label+" ", bg, "\x1b[1m", "\x1b[33m")
			query += styleIf(true, "▍", "\x1b[36m")
		} else {
			pill = styleIf(true, " "+label+" ", bg, "\x1b[90m")
		}
	}
	searchLine := pill + " " + query
	writeLineAt(out, layout.searchRow, 1, searchLine, layout.cols)
}

func drawHints(out *bufio.Writer, state *appState, layout layout) {
	formatHint := func(key, label string) string {
		if !state.useColor {
			return key + " " + label
		}
		return styleIf(true, key, "\x1b[1m", "\x1b[36m") + " " + styleIf(true, label, "\x1b[37m")
	}
	quitKey := "q"
	if state.mode == modeQuery {
		quitKey = "Ctrl-C"
	}
	hints := strings.Join([]string{
		formatHint("⏎", "Search"),
		formatHint("/", "Edit"),
		formatHint("↑↓", "Select"),
		formatHint("d", "Download"),
		formatHint("c", "Copy"),
		formatHint("f", "Reveal"),
		formatHint(quitKey, "Quit"),
	}, "  ")
	// Hints live below the content area; center across the full terminal width,
	// even when the content is split (preview left / list right).
	pad := max(0, (layout.cols-visibleRuneLen(hints))/2)
	line := strings.Repeat(" ", pad) + hints
	writeLineAt(out, layout.hintsRow, 1, line, layout.cols)
}

func clearUnused(out *bufio.Writer, layout layout) {
	for row := 1; row <= layout.rows; row++ {
		if row == 1 || (row >= layout.contentTop && row <= layout.contentBottom) || row == layout.statusRow || row == layout.searchRow || row == layout.hintsRow {
			continue
		}
		writeLineAt(out, row, 1, "", layout.cols)
	}
}

func flashHeader(state *appState, msg string) {
	if state == nil {
		return
	}
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}
	state.headerFlash = msg
	state.headerFlashAt = nowFn().Add(3 * time.Second)
}
