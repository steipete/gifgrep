package tui

import (
	"bufio"
	"io"
	"strings"
	"time"
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

type inputRune struct {
	value rune
	size  int
}

func readInput(r io.Reader, ch chan<- inputEvent, stop <-chan struct{}) {
	runes := make(chan inputRune)
	done := make(chan struct{})
	defer close(done)
	go func() {
		defer close(runes)
		reader := bufio.NewReader(r)
		for {
			value, size, err := reader.ReadRune()
			if err != nil {
				return
			}
			select {
			case runes <- inputRune{value, size}:
			case <-stop:
				return
			case <-done:
				return
			}
		}
	}()

	var pending *inputRune
	for {
		var next inputRune
		if pending != nil {
			next = *pending
			pending = nil
		} else {
			var ok bool
			next, ok = nextInputRune(runes, stop, nil)
			if !ok {
				return
			}
		}
		var ev inputEvent
		switch next.value {
		case 0x03:
			ev.kind = keyCtrlC
		case '\r', '\n':
			ev.kind = keyEnter
		case 0x7f, 0x08:
			ev.kind = keyBackspace
		case 0x1b:
			ev, pending = readEscape(runes, stop)
		default:
			if unicode.IsControl(next.value) || (next.value == utf8.RuneError && next.size == 1) {
				continue
			}
			ev = inputEvent{kind: keyRune, ch: next.value}
		}
		select {
		case ch <- ev:
		case <-stop:
			return
		}
	}
}

func nextInputRune(runes <-chan inputRune, stop <-chan struct{}, timeout <-chan time.Time) (inputRune, bool) {
	select {
	case r, ok := <-runes:
		return r, ok
	case <-stop:
		return inputRune{}, false
	case <-timeout:
		return inputRune{}, false
	}
}

func readEscape(runes <-chan inputRune, stop <-chan struct{}) (inputEvent, *inputRune) {
	// A lone Escape must complete without waiting indefinitely for a CSI sequence.
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	next, ok := nextInputRune(runes, stop, timer.C)
	if !ok {
		return inputEvent{kind: keyEsc}, nil
	}
	if next.value != '[' {
		return inputEvent{kind: keyEsc}, &next
	}
	third, ok := nextInputRune(runes, stop, timer.C)
	if ok {
		switch third.value {
		case 0x03:
			return inputEvent{kind: keyCtrlC}, nil
		case 'A':
			return inputEvent{kind: keyUp}, nil
		case 'B':
			return inputEvent{kind: keyDown}, nil
		}
	}
	return inputEvent{kind: keyUnknown}, nil
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
			loadSelectedImage(state)
			state.renderDirty = true
		}
	case keyDown:
		if state.selected < len(state.results)-1 {
			state.selected++
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
