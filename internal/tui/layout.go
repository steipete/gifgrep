package tui

import (
	"math"
)

func ensureVisible(state *appState, listHeight int) {
	listHeight = max(1, listHeight)
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
		layout.previewCols, layout.previewRows = fitPreviewSize(maxPreviewCols, layout.contentHeight, state.currentAnim)
	} else {
		availRows := layout.contentHeight / 2
		if availRows < 6 {
			availRows = min(6, layout.contentHeight)
		}
		if availRows > layout.contentHeight-2 {
			availRows = max(0, layout.contentHeight-2)
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
		layout.listWidth = max(0, cols-layout.listCol+1)
		layout.clearWidth = layout.listCol - gapCols
		layout.previewRow = layout.contentTop + max(0, (layout.contentHeight-layout.previewRows)/2)
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
	return min(targetCols, availCols), min(targetRows, availRows)
}
