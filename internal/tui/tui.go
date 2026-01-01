package tui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/steipete/gifgrep/gifdecode"
	"github.com/steipete/gifgrep/internal/assets"
	"github.com/steipete/gifgrep/internal/iterm"
	"github.com/steipete/gifgrep/internal/kitty"
	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/search"
	"github.com/steipete/gifgrep/internal/termcaps"
	"golang.org/x/term"
)

type inputEvent struct {
	kind keyKind
	ch   rune
}

type keyKind int

const (
	keyRune keyKind = iota
	keyEnter
	keyBackspace
	keyEsc
	keyUp
	keyDown
	keyCtrlC
	keyUnknown
)

var ErrNotTerminal = errors.New("stdin is not a tty")

const giphyAttributionImageID uint32 = 0x67697068 // "giph"

var nowFn = time.Now

func Run(opts model.Options, query string) error {
	env := defaultEnvFn()
	return runWith(env, opts, query)
}

func errUnsupportedInline(getenv func(string) string) error {
	if getenv == nil {
		getenv = os.Getenv
	}
	termProgram := strings.TrimSpace(getenv("TERM_PROGRAM"))
	term := strings.TrimSpace(getenv("TERM"))
	return fmt.Errorf(
		"gifgrep tui needs inline image support.\n\nSupported terminals:\n  - Kitty (Kitty graphics protocol)\n  - Ghostty (Kitty graphics protocol)\n  - iTerm2 (OSC 1337 inline images)\n\nDetected:\n  TERM_PROGRAM=%q\n  TERM=%q\n\nSee: docs/kitty.md and docs/iterm.md\n\nTip: You can force detection with GIFGREP_INLINE=kitty|iterm|none",
		termProgram,
		term,
	)
}

func runWith(env Env, opts model.Options, query string) error {
	if env.In == nil {
		env.In = os.Stdin
	}
	if env.Out == nil {
		env.Out = os.Stdout
	}
	if env.IsTerminal == nil {
		env.IsTerminal = term.IsTerminal
	}
	if env.MakeRaw == nil {
		env.MakeRaw = term.MakeRaw
	}
	if env.Restore == nil {
		env.Restore = term.Restore
	}
	if env.GetSize == nil {
		env.GetSize = term.GetSize
	}
	if env.FD == 0 {
		env.FD = int(os.Stdin.Fd())
	}
	if !env.IsTerminal(env.FD) {
		return ErrNotTerminal
	}

	inline := termcaps.DetectInlineRobust(os.Getenv)
	if inline == termcaps.InlineNone {
		return errUnsupportedInline(os.Getenv)
	}

	oldState, err := env.MakeRaw(env.FD)
	if err != nil {
		return err
	}
	if oldState != nil {
		defer func() {
			_ = env.Restore(env.FD, oldState)
		}()
	}

	out := bufio.NewWriter(env.Out)
	hideCursor(out)
	defer func() {
		showCursor(out)
		if inline == termcaps.InlineKitty {
			clearImages(out)
		}
		_ = out.Flush()
	}()

	sigs := env.SignalCh
	if sigs == nil {
		sigs = make(chan os.Signal)
	}

	inputCh := make(chan inputEvent, 16)
	stopCh := make(chan struct{})
	go readInput(env.In, inputCh, stopCh)

	state := &appState{
		mode:            modeQuery,
		status:          "Type a search and press Enter",
		tagline:         pickTagline(time.Now(), os.Getenv, nil),
		cache:           map[string]*gifCacheEntry{},
		savedPaths:      map[string]string{},
		renderDirty:     true,
		nextImageID:     1,
		inline:          inline,
		useSoftwareAnim: inline == termcaps.InlineKitty && useSoftwareAnimation(),
		useColor:        opts.Color != "never",
		opts:            opts,
	}
	if cols, rows, err := env.GetSize(env.FD); err == nil {
		state.lastRows = rows
		state.lastCols = cols
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	if strings.TrimSpace(query) != "" {
		state.query = query
		state.mode = modeBrowse
		state.status = "Searching..."
		render(state, out, state.lastRows, state.lastCols)
		_ = out.Flush()

		results, err := search.Search(query, opts)
		if err != nil {
			state.status = "Search error: " + err.Error()
		} else {
			state.results = results
			state.selected = 0
			state.scroll = 0
			if len(results) == 0 {
				state.status = "No results"
				state.currentAnim = nil
				state.previewDirty = true
			} else {
				state.status = fmt.Sprintf("%d results", len(results))
				loadSelectedImage(state)
			}
		}
		state.renderDirty = true
	}

	for {
		select {
		case <-sigs:
			close(stopCh)
			return nil
		case ev := <-inputCh:
			if handleInput(state, ev, out) {
				close(stopCh)
				return nil
			}
		case <-ticker.C:
		}

		if cols, rows, err := env.GetSize(env.FD); err == nil {
			if rows != state.lastRows || cols != state.lastCols {
				state.lastRows = rows
				state.lastCols = cols
				ensureVisible(state)
				state.renderDirty = true
				state.previewDirty = true
			}
		}

		if state.renderDirty {
			render(state, out, state.lastRows, state.lastCols)
			state.renderDirty = false
			_ = out.Flush()
		}

		advanceManualAnimation(state, out)
	}
}

func readInput(r io.Reader, ch chan<- inputEvent, stop <-chan struct{}) {
	reader := bufio.NewReader(r)
	for {
		select {
		case <-stop:
			return
		default:
		}

		b, err := reader.ReadByte()
		if err != nil {
			return
		}
		switch b {
		case 0x03:
			ch <- inputEvent{kind: keyCtrlC}
		case '\r', '\n':
			ch <- inputEvent{kind: keyEnter}
		case 0x7f, 0x08:
			ch <- inputEvent{kind: keyBackspace}
		case 0x1b:
			next, err := reader.ReadByte()
			if err != nil {
				ch <- inputEvent{kind: keyEsc}
				continue
			}
			if next == '[' {
				third, _ := reader.ReadByte()
				switch third {
				case 'A':
					ch <- inputEvent{kind: keyUp}
				case 'B':
					ch <- inputEvent{kind: keyDown}
				default:
					ch <- inputEvent{kind: keyUnknown}
				}
			} else {
				_ = reader.UnreadByte()
				ch <- inputEvent{kind: keyEsc}
			}
		default:
			if b >= 0x20 && b < 0x7f {
				ch <- inputEvent{kind: keyRune, ch: rune(b)}
			}
		}
	}
}

func handleInput(state *appState, ev inputEvent, out *bufio.Writer) bool {
	if ev.kind == keyCtrlC {
		return true
	}
	if ev.kind == keyRune && ev.ch == 'q' {
		return true
	}

	switch state.mode {
	case modeQuery:
		return handleQueryInput(state, ev, out)
	case modeBrowse:
		return handleBrowseInput(state, ev, out)
	}

	return false
}

func handleQueryInput(state *appState, ev inputEvent, out *bufio.Writer) bool {
	switch ev.kind {
	case keyRune:
		state.query += string(ev.ch)
		state.renderDirty = true
	case keyBackspace:
		if len(state.query) > 0 {
			state.query = state.query[:len(state.query)-1]
			state.renderDirty = true
		}
	case keyEnter:
		if strings.TrimSpace(state.query) == "" {
			state.status = "Empty query"
			state.renderDirty = true
			return false
		}
		state.status = "Searching..."
		render(state, out, state.lastRows, state.lastCols)
		_ = out.Flush()

		results, err := search.Search(state.query, state.opts)
		if err != nil {
			state.status = "Search error: " + err.Error()
		} else {
			state.results = results
			state.selected = 0
			state.scroll = 0
			if len(results) == 0 {
				state.status = "No results"
				state.currentAnim = nil
				state.previewDirty = true
			} else {
				state.status = fmt.Sprintf("%d results", len(results))
				loadSelectedImage(state)
			}
		}
		state.mode = modeBrowse
		state.renderDirty = true
	case keyEsc:
		if len(state.results) > 0 {
			state.mode = modeBrowse
			state.renderDirty = true
		}
	case keyCtrlC:
		return true
	case keyUp, keyDown, keyUnknown:
		// ignore
	}
	return false
}

func handleBrowseInput(state *appState, ev inputEvent, out *bufio.Writer) bool {
	switch ev.kind {
	case keyRune:
		if ev.ch == '/' {
			state.mode = modeQuery
			state.status = "Type a search and press Enter"
			state.renderDirty = true
			return false
		}
		switch ev.ch {
		case 'd':
			downloadSelected(state, out, state.opts.Reveal)
			return false
		case 'f':
			return handleRevealSelected(state, out)
		default:
		}
		if ev.ch >= 0x20 {
			state.mode = modeQuery
			state.status = "Type a search and press Enter"
			state.query = string(ev.ch)
			state.renderDirty = true
			return false
		}
	case keyUp:
		if state.selected > 0 {
			state.selected--
			ensureVisible(state)
			loadSelectedImage(state)
			state.renderDirty = true
		}
	case keyDown:
		if state.selected < len(state.results)-1 {
			state.selected++
			ensureVisible(state)
			loadSelectedImage(state)
			state.renderDirty = true
		}
	case keyEnter:
		state.mode = modeQuery
		state.status = "Type a search and press Enter"
		state.renderDirty = true
	case keyEsc:
		state.mode = modeQuery
		state.renderDirty = true
	case keyCtrlC:
		return true
	case keyBackspace, keyUnknown:
		// ignore
	}
	return false
}

func ensureVisible(state *appState) {
	listHeight := state.lastRows - 4
	if listHeight < 0 {
		listHeight = 0
	}
	if state.selected < state.scroll {
		state.scroll = state.selected
	}
	if state.selected >= state.scroll+listHeight {
		state.scroll = state.selected - listHeight + 1
	}
	if state.scroll < 0 {
		state.scroll = 0
	}
}

func render(state *appState, out *bufio.Writer, rows, cols int) {
	if rows <= 0 || cols <= 0 {
		return
	}

	layout := buildLayout(state, rows, cols)

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

type layout struct {
	rows, cols                     int
	statusRow, searchRow, hintsRow int
	contentTop, contentBottom      int
	contentHeight                  int
	listCol, listWidth             int
	listHeight                     int
	previewCols, previewRows       int
	previewRow, previewCol         int
	clearWidth                     int
	showRight                      bool
	hasContent                     bool
}

func buildLayout(state *appState, rows, cols int) layout {
	layout := layout{rows: rows, cols: cols}
	layout.searchRow = rows - 2
	layout.statusRow = rows - 1
	layout.hintsRow = rows
	if layout.searchRow < 2 {
		return layout
	}

	layout.contentTop = 2
	layout.contentBottom = layout.searchRow - 1
	if layout.contentBottom < layout.contentTop {
		return layout
	}
	layout.contentHeight = layout.contentBottom - layout.contentTop + 1
	layout.hasContent = true

	showRight := cols >= 80 && rows >= 14 && state.currentAnim != nil
	minListWidth := 28
	gapCols := 1
	maxPreviewCols := cols
	if showRight {
		maxPreviewCols = cols - minListWidth - gapCols
		if maxPreviewCols < 10 {
			showRight = false
		}
	}
	layout.showRight = showRight

	if showRight {
		// iTerm inline images are part of the text grid. Keep the split boundary stable
		// (don't resize the preview box per GIF aspect ratio), otherwise old text can
		// "leak" when the list column shifts.
		if state.inline == termcaps.InlineIterm {
			layout.previewCols = maxPreviewCols
			layout.previewRows = layout.contentHeight
		} else {
			layout.previewCols, layout.previewRows = fitPreviewSize(maxPreviewCols, layout.contentHeight, state.currentAnim)
		}
	} else {
		availRows := layout.contentHeight / 2
		if availRows < 6 {
			availRows = minInt(6, layout.contentHeight)
		}
		if availRows > layout.contentHeight-2 {
			availRows = maxInt(0, layout.contentHeight-2)
		}
		layout.previewCols, layout.previewRows = fitPreviewSize(cols, availRows, state.currentAnim)
	}
	if state.currentAnim == nil {
		layout.previewCols = 0
		layout.previewRows = 0
	}

	layout.listCol = 1
	layout.listWidth = cols
	layout.listHeight = layout.contentHeight
	if showRight && layout.previewCols > 0 {
		layout.listCol = layout.previewCols + gapCols + 1
		layout.listWidth = maxInt(0, cols-layout.listCol+1)
		layout.clearWidth = layout.listCol - gapCols
		layout.previewRow = layout.contentTop + maxInt(0, (layout.contentHeight-layout.previewRows)/2)
		layout.previewCol = 1
	} else if !showRight && layout.previewRows > 0 {
		layout.listHeight = layout.contentHeight - layout.previewRows - 1
		if layout.listHeight < 0 {
			layout.listHeight = 0
		}
		layout.previewRow = layout.contentTop + layout.listHeight + 1
		layout.previewCol = 1
	}

	return layout
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
	showGiphyIcon := showGiphyAttribution && state.inline == termcaps.InlineKitty
	logoCols := 2
	logoRows := 1
	statusWidth := layout.cols
	if showGiphyIcon {
		statusWidth = maxInt(0, layout.cols-(logoCols+1))
	}
	line := formatStatusLine(state.useColor, status)
	if showGiphyAttribution {
		line += styleIf(state.useColor, " · Powered by GIPHY", "\x1b[90m")
	}
	writeLineAt(out, layout.statusRow, 1, line, statusWidth)
	if showGiphyIcon && layout.cols >= logoCols {
		moveCursor(out, layout.statusRow, maxInt(1, layout.cols-logoCols+1))
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
	pill := "[Search]"
	query := state.query
	if state.useColor {
		bg := "\x1b[48;5;236m"
		if state.mode == modeQuery {
			pill = styleIf(true, " Search ", bg, "\x1b[1m", "\x1b[33m")
			query += styleIf(true, "▍", "\x1b[36m")
		} else {
			pill = styleIf(true, " Search ", bg, "\x1b[90m")
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
		return styleIf(true, key, "\x1b[1m", "\x1b[36m") + " " + styleIf(true, label, "\x1b[90m")
	}
	hints := strings.Join([]string{
		formatHint("Enter", "Search"),
		formatHint("/", "Edit"),
		formatHint("Up/Down", "Select"),
		formatHint("d", "Download"),
		formatHint("f", "Reveal"),
		formatHint("q", "Quit"),
	}, "  ")
	// Hints live below the content area; center across the full terminal width,
	// even when the content is split (preview left / list right).
	pad := maxInt(0, (layout.cols-visibleRuneLen(hints))/2)
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

func availablePreviewSize(rows, cols, leftWidth int, showRight bool) (int, int) {
	if rows <= 0 || cols <= 0 {
		return 0, 0
	}
	if showRight {
		availCols := cols - leftWidth - 2
		availRows := rows - 4
		if availCols < 10 || availRows < 6 {
			return 0, 0
		}
		return availCols, availRows
	}
	availCols := cols
	availRows := rows / 3
	if availRows < 6 {
		availRows = 6
	}
	maxRows := rows - 6
	if availRows > maxRows {
		availRows = maxRows
	}
	if availCols < 10 || availRows <= 0 {
		return 0, 0
	}
	return availCols, availRows
}

func fitPreviewSize(availCols, availRows int, anim *gifAnimation) (int, int) {
	if availCols <= 0 || availRows <= 0 {
		return 0, 0
	}
	if anim == nil || anim.Width <= 0 || anim.Height <= 0 {
		return availCols, availRows
	}
	aspect := cellAspectRatio()
	targetCols := availCols
	targetRows := int(math.Round(float64(targetCols) * aspect * float64(anim.Height) / float64(anim.Width)))
	if targetRows > availRows {
		targetRows = availRows
		targetCols = int(math.Round(float64(targetRows) / aspect * float64(anim.Width) / float64(anim.Height)))
	}
	if targetCols < 1 {
		targetCols = 1
	}
	if targetRows < 1 {
		targetRows = 1
	}
	return minInt(targetCols, availCols), minInt(targetRows, availRows)
}

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
		if hasPrev {
			clearItermRectFn(out, state.itermLast.row, state.itermLast.col, state.itermLast.cols, state.itermLast.rows)
			// If the new box is larger or moved, also clear the new rect once to avoid
			// old list text "bleeding" into the preview area (iTerm images are in-grid).
			if row != state.itermLast.row || col != state.itermLast.col || cols > state.itermLast.cols || rows > state.itermLast.rows {
				clearItermRectFn(out, row, col, cols, rows)
			}
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
	if state.useSoftwareAnim && len(state.currentAnim.Frames) > 1 {
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

func writeLineAt(out *bufio.Writer, row, col int, text string, width int) {
	moveCursor(out, row, col)
	if width <= 0 {
		_, _ = fmt.Fprint(out, "\x1b[K")
		return
	}
	text = truncateANSI(text, width)
	_, _ = fmt.Fprint(out, text)
	_, _ = fmt.Fprint(out, "\x1b[K")
}

func drawPreviewSoftware(state *appState, out *bufio.Writer, cols, rows int, row, col int) {
	if state.currentAnim == nil || len(state.currentAnim.Frames) == 0 {
		return
	}
	if state.activeImageID != 0 && state.activeImageID != state.currentAnim.ID {
		kitty.DeleteImage(out, state.activeImageID)
	}
	state.activeImageID = state.currentAnim.ID
	if state.previewNeedsSend {
		state.manualAnim = true
		state.manualFrame = 0
		frame := state.currentAnim.Frames[state.manualFrame]
		saveCursor(out)
		moveCursor(out, row, col)
		kitty.SendFrame(out, state.activeImageID, frame, cols, rows)
		restoreCursor(out)
		state.manualNext = time.Now().Add(frame.Delay)
		state.previewNeedsSend = false
		state.previewDirty = false
		state.lastPreview.cols = cols
		state.lastPreview.rows = rows
		return
	}
	if state.previewDirty || state.lastPreview.cols != cols || state.lastPreview.rows != rows {
		frame := state.currentAnim.Frames[state.manualFrame]
		saveCursor(out)
		moveCursor(out, row, col)
		kitty.SendFrame(out, state.activeImageID, frame, cols, rows)
		restoreCursor(out)
		state.previewDirty = false
		state.lastPreview.cols = cols
		state.lastPreview.rows = rows
	}
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
	saveCursor(out)
	moveCursor(out, state.previewRow, state.previewCol)
	kitty.SendFrame(out, state.activeImageID, frame, state.lastPreview.cols, state.lastPreview.rows)
	restoreCursor(out)
	state.manualNext = now.Add(frame.Delay)
	_ = out.Flush()
}

func writeLine(out *bufio.Writer, text string, width int) {
	if width <= 0 {
		_, _ = fmt.Fprint(out, "\r\n")
		return
	}
	text = truncateANSI(text, width)
	_, _ = fmt.Fprint(out, text)
	_, _ = fmt.Fprint(out, "\x1b[K\r\n")
}

func moveCursor(out *bufio.Writer, row, col int) {
	if row < 1 {
		row = 1
	}
	if col < 1 {
		col = 1
	}
	_, _ = fmt.Fprintf(out, "\x1b[%d;%dH", row, col)
}

func saveCursor(out *bufio.Writer) {
	_, _ = fmt.Fprint(out, "\x1b7")
}

func restoreCursor(out *bufio.Writer) {
	_, _ = fmt.Fprint(out, "\x1b8")
}

func hideCursor(out *bufio.Writer) {
	_, _ = fmt.Fprint(out, "\x1b[?25l")
}

func showCursor(out *bufio.Writer) {
	_, _ = fmt.Fprint(out, "\x1b[?25h")
}

func clearImages(out *bufio.Writer) {
	_, _ = fmt.Fprint(out, "\x1b_Ga=d\x1b\\")
}

func clearItermRect(out *bufio.Writer, row, col, cols, rows int) {
	if out == nil || cols <= 0 || rows <= 0 {
		return
	}
	if row < 1 {
		row = 1
	}
	if col < 1 {
		col = 1
	}
	blank := strings.Repeat(" ", cols)
	saveCursor(out)
	for i := 0; i < rows; i++ {
		moveCursor(out, row+i, col)
		_, _ = fmt.Fprint(out, blank)
	}
	restoreCursor(out)
}

var clearItermRectFn = clearItermRect

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

func clearItermGapColumn(out *bufio.Writer, layout layout) {
	if out == nil || !layout.showRight || layout.previewCols <= 0 || layout.contentHeight <= 0 {
		return
	}
	col := layout.previewCols + 1
	if col < 1 || col > layout.cols {
		return
	}
	saveCursor(out)
	for row := layout.contentTop; row <= layout.contentBottom; row++ {
		moveCursor(out, row, col)
		_, _ = fmt.Fprint(out, " ")
	}
	restoreCursor(out)
}
