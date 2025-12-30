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
	"github.com/steipete/gifgrep/internal/kitty"
	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/search"
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

func Run(opts model.Options, query string) error {
	env := defaultEnvFn()
	return runWith(env, opts, query)
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
		clearImages(out)
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
		cache:           map[string]*gifdecode.Frames{},
		renderDirty:     true,
		nextImageID:     1,
		useSoftwareAnim: useSoftwareAnimation(),
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
			results, err = search.FilterResults(results, query, opts)
			if err != nil {
				state.status = "Filter error: " + err.Error()
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
				results, err = search.FilterResults(results, state.query, state.opts)
				if err != nil {
					state.status = "Filter error: " + err.Error()
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
			}
			state.mode = modeBrowse
			state.renderDirty = true
		case keyEsc:
			if len(state.results) > 0 {
				state.mode = modeBrowse
				state.renderDirty = true
			}
		}
	case modeBrowse:
		switch ev.kind {
		case keyRune:
			if ev.ch == '/' {
				state.mode = modeQuery
				state.status = "Type a search and press Enter"
				state.renderDirty = true
				return false
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
		}
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

	showRight := cols >= 80 && rows >= 14
	rightWidth := cols
	leftWidth := cols
	if showRight {
		rightWidth = maxInt(28, cols/3)
		if rightWidth > cols-2 {
			rightWidth = cols - 2
		}
		leftWidth = cols - rightWidth - 2
	}

	if state.currentAnim == nil && state.activeImageID != 0 {
		kitty.DeleteImage(out, state.activeImageID)
		state.activeImageID = 0
	}

	header := styleIf(state.useColor, "gifgrep", "\x1b[1m", "\x1b[36m")
	header = header + styleIf(state.useColor, " — GIF search", "\x1b[90m")
	writeLineAt(out, 1, 1, header, cols)

	statusRow := rows - 2
	searchRow := rows - 1
	hintsRow := rows
	if statusRow < 2 {
		for row := 1; row <= rows; row++ {
			writeLineAt(out, row, 1, "", cols)
		}
		return
	}

	contentTop := 2
	contentBottom := statusRow - 1
	if contentBottom < contentTop {
		for row := 1; row <= rows; row++ {
			writeLineAt(out, row, 1, "", cols)
		}
		return
	}
	contentHeight := contentBottom - contentTop + 1

	var previewCols, previewRows int
	if showRight {
		previewCols, previewRows = fitPreviewSize(leftWidth, contentHeight, state.currentAnim)
	} else {
		availRows := contentHeight / 2
		if availRows < 6 {
			availRows = minInt(6, contentHeight)
		}
		if availRows > contentHeight-2 {
			availRows = maxInt(0, contentHeight-2)
		}
		previewCols, previewRows = fitPreviewSize(cols, availRows, state.currentAnim)
	}
	if state.currentAnim == nil {
		previewCols = 0
		previewRows = 0
	}

	listCol := 1
	listWidth := cols
	if showRight {
		listCol = leftWidth + 2
		listWidth = rightWidth
	}

	listHeight := contentHeight
	if !showRight && previewRows > 0 {
		listHeight = contentHeight - previewRows - 1
	}
	if listHeight < 0 {
		listHeight = 0
	}

	if showRight {
		for i := 0; i < contentHeight; i++ {
			writeLineAt(out, contentTop+i, 1, "", leftWidth)
		}
	}

	for i := 0; i < listHeight; i++ {
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
			writeLineAt(out, contentTop+i, listCol, prefix+label, listWidth)
		} else {
			writeLineAt(out, contentTop+i, listCol, "", listWidth)
		}
	}

	if state.currentAnim != nil && previewCols > 0 && previewRows > 0 {
		if showRight {
			state.previewCol = 1
			state.previewRow = contentTop + maxInt(0, (contentHeight-previewRows)/2)
			moveCursor(out, state.previewRow, state.previewCol)
			drawPreview(state, out, previewCols, previewRows, state.previewRow, state.previewCol)
		} else {
			label := styleIf(state.useColor, "Preview", "\x1b[90m")
			writeLineAt(out, contentTop+listHeight, 1, label, cols)
			state.previewRow = contentTop + listHeight + 1
			state.previewCol = 1
			for i := 0; i < previewRows; i++ {
				writeLineAt(out, state.previewRow+i, 1, "", cols)
			}
			moveCursor(out, state.previewRow, state.previewCol)
			drawPreview(state, out, previewCols, previewRows, state.previewRow, state.previewCol)
		}
	}

	status := state.status
	if status == "" {
		status = fmt.Sprintf("%d results", len(state.results))
	}
	status = styleIf(state.useColor, status, "\x1b[90m")
	writeLineAt(out, statusRow, 1, status, cols)

	searchLabel := "Search: "
	if state.mode == modeQuery {
		searchLabel = styleIf(state.useColor, "Search: ", "\x1b[1m", "\x1b[33m")
	} else {
		searchLabel = styleIf(state.useColor, "Search: ", "\x1b[90m")
	}
	searchLine := searchLabel + state.query
	writeLineAt(out, searchRow, 1, searchLine, cols)

	hints := "Enter search  / edit  Up/Down select  q quit"
	hints = styleIf(state.useColor, hints, "\x1b[90m")
	writeLineAt(out, hintsRow, 1, hints, cols)

	for row := 1; row <= rows; row++ {
		if row == 1 || (row >= contentTop && row <= contentBottom) || row == statusRow || row == searchRow || row == hintsRow {
			continue
		}
		writeLineAt(out, row, 1, "", cols)
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
	if state.currentAnim == nil || len(state.currentAnim.Frames) == 0 {
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
