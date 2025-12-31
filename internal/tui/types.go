package tui

import (
	"time"

	"github.com/steipete/gifgrep/gifdecode"
	"github.com/steipete/gifgrep/internal/model"
)

type mode int

const (
	modeBrowse mode = iota
	modeQuery
)

type gifAnimation struct {
	ID     uint32
	Frames []gifdecode.Frame
	Width  int
	Height int
}

type appState struct {
	query       string
	tagline     string
	results     []model.Result
	selected    int
	scroll      int
	mode        mode
	status      string
	currentAnim *gifAnimation
	cache       map[string]*gifdecode.Frames
	renderDirty bool
	lastRows    int
	lastCols    int
	previewRow  int
	previewCol  int
	lastPreview struct {
		cols int
		rows int
	}
	previewNeedsSend      bool
	previewDirty          bool
	nextImageID           uint32
	activeImageID         uint32
	manualAnim            bool
	manualFrame           int
	manualNext            time.Time
	useSoftwareAnim       bool
	useColor              bool
	opts                  model.Options
	giphyAttributionShown bool
	lastSavedPath         string
}
