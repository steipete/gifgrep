package tui

import (
	"bufio"
	"io"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"unicode/utf8"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/termcaps"
)

func TestReadInputUnicode(t *testing.T) {
	query := "quiet café 漢字 🦞 e\u0301 👩‍💻 \ufffd"
	data := query + "\x00\x1f\u0085\xff\x1b[A\x1b[B\x1bé\x7f\r\x03"
	want := make([]inputEvent, 0, len(data))
	for _, r := range query {
		want = append(want, inputEvent{kind: keyRune, ch: r})
	}
	want = append(want,
		inputEvent{kind: keyUp}, inputEvent{kind: keyDown},
		inputEvent{kind: keyEsc}, inputEvent{kind: keyRune, ch: 'é'},
		inputEvent{kind: keyBackspace}, inputEvent{kind: keyEnter}, inputEvent{kind: keyCtrlC},
	)
	ch := make(chan inputEvent, len(data))
	readInput(iotest.OneByteReader(strings.NewReader(data)), ch, make(chan struct{}))
	close(ch)
	var got []inputEvent
	for ev := range ch {
		got = append(got, ev)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("input events = %#v, want %#v", got, want)
	}
}

func TestQueryBackspaceUnicode(t *testing.T) {
	for _, query := range []string{"", "q", "café", "漢字", "quiet 🦞", "e\u0301", "👩‍💻"} {
		t.Run(query, func(t *testing.T) {
			state := newAppState(termcaps.InlineANSI, model.Options{})
			state.query = query
			out := bufio.NewWriter(io.Discard)
			remaining := []rune(query)
			for {
				state.renderDirty = false
				if handleInput(state, inputEvent{kind: keyBackspace}, out, nil) {
					t.Fatal("backspace quit the TUI")
				}
				changed := len(remaining) > 0
				if changed {
					remaining = remaining[:len(remaining)-1]
				}
				if !utf8.ValidString(state.query) || state.query != string(remaining) {
					t.Fatalf("query after backspace = %q, want %q", state.query, string(remaining))
				}
				if state.renderDirty != changed {
					t.Fatalf("renderDirty = %v, want %v", state.renderDirty, changed)
				}
				if !changed {
					break
				}
			}
		})
	}
}
