package tui

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/steipete/gifgrep/internal/termcaps"
)

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

func shouldSendItermPreview(state *appState, layout layout) bool {
	if state == nil || state.inline != termcaps.InlineIterm || !layout.showRight {
		return false
	}
	if state.currentAnim == nil || layout.previewCols <= 0 || layout.previewRows <= 0 {
		return false
	}
	if state.previewNeedsSend || state.previewDirty {
		return true
	}
	if state.lastPreview.cols != layout.previewCols || state.lastPreview.rows != layout.previewRows {
		return true
	}
	return false
}

func eraseItermContentArea(out *bufio.Writer, layout layout) {
	if out == nil || !layout.hasContent || layout.contentHeight <= 0 {
		return
	}
	saveCursor(out)
	for row := layout.contentTop; row <= layout.contentBottom; row++ {
		moveCursor(out, row, 1)
		_, _ = fmt.Fprint(out, "\x1b[2K")
	}
	restoreCursor(out)
}

var eraseItermContentAreaFn = eraseItermContentArea

func shouldHardClearIterm(state *appState, layout layout) bool {
	if state == nil || state.inline != termcaps.InlineIterm || !layout.showRight {
		return false
	}
	if state.itermLast.cols <= 0 || state.itermLast.rows <= 0 {
		return false
	}
	newRow := layout.previewRow
	newCol := 1
	newCols := layout.previewCols
	newRows := layout.previewRows
	if newCols <= 0 || newRows <= 0 {
		return false
	}
	return !rectCovers(
		itermRect{row: newRow, col: newCol, cols: newCols, rows: newRows},
		itermRect{row: state.itermLast.row, col: state.itermLast.col, cols: state.itermLast.cols, rows: state.itermLast.rows},
	)
}

type itermRect struct {
	row  int
	col  int
	cols int
	rows int
}

func rectCovers(outer, inner itermRect) bool {
	if outer.cols <= 0 || outer.rows <= 0 || inner.cols <= 0 || inner.rows <= 0 {
		return false
	}
	outerRight := outer.col + outer.cols - 1
	outerBottom := outer.row + outer.rows - 1
	innerRight := inner.col + inner.cols - 1
	innerBottom := inner.row + inner.rows - 1
	return outer.col <= inner.col && outer.row <= inner.row && outerRight >= innerRight && outerBottom >= innerBottom
}

func clearItermScreen(out *bufio.Writer) {
	if out == nil {
		return
	}
	_, _ = fmt.Fprint(out, "\x1b[2J\x1b[H")
}

var clearItermScreenFn = clearItermScreen
