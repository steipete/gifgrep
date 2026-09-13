package tui

import (
	"bufio"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
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

func readInput(r io.Reader, ch chan<- inputEvent, stop <-chan struct{}) {
	reader := bufio.NewReader(r)
	for {
		select {
		case <-stop:
			return
		default:
		}

		r, size, err := reader.ReadRune()
		if err != nil {
			return
		}
		switch r {
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
			if !unicode.IsControl(r) && (r != utf8.RuneError || size > 1) {
				ch <- inputEvent{kind: keyRune, ch: r}
			}
		}
	}
}

func handleInput(state *appState, ev inputEvent, out *bufio.Writer, prefetchCh chan<- prefetchResult) bool {
	if ev.kind == keyCtrlC {
		return true
	}
	switch state.mode {
	case modeQuery:
		return handleQueryInput(state, ev, out, prefetchCh)
	case modeBrowse:
		return handleBrowseInput(state, ev, out)
	}

	return false
}

func handleQueryInput(state *appState, ev inputEvent, out *bufio.Writer, prefetchCh chan<- prefetchResult) bool {
	switch ev.kind {
	case keyRune:
		state.query += string(ev.ch)
		state.renderDirty = true
	case keyBackspace:
		if state.query != "" {
			_, size := utf8.DecodeLastRuneInString(state.query)
			state.query = state.query[:len(state.query)-size]
			state.renderDirty = true
		}
	case keyEnter:
		if strings.TrimSpace(state.query) == "" {
			state.status = "Empty query"
			state.renderDirty = true
			return false
		}
		searchResults(state, out, prefetchCh)
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
		case 'q':
			return true
		case 'c':
			copySelected(state, out)
			return false
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
