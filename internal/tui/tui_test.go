package tui

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/steipete/gifgrep/gifdecode"
	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/testutil"
)

func TestPreviewSize(t *testing.T) {
	c, r := availablePreviewSize(40, 120, 40, true)
	if c != 78 || r != 36 {
		t.Fatalf("unexpected preview size: %d %d", c, r)
	}
	c, r = availablePreviewSize(30, 80, 80, false)
	if c != 80 || r != 10 {
		t.Fatalf("unexpected preview size: %d %d", c, r)
	}
	c, r = availablePreviewSize(10, 20, 20, false)
	if c == 0 || r == 0 {
		t.Fatalf("expected preview size for small terminal")
	}

	anim := &gifAnimation{Width: 200, Height: 100}
	fc, fr := fitPreviewSize(78, 36, anim)
	if fc != 78 || fr != 20 {
		t.Fatalf("unexpected fit size: %d %d", fc, fr)
	}
}

func TestEnsureVisible(t *testing.T) {
	state := &appState{lastRows: 10, selected: 8, scroll: 0}
	ensureVisible(state)
	if state.scroll == 0 {
		t.Fatalf("scroll should advance")
	}
	state.selected = 0
	ensureVisible(state)
	if state.scroll != 0 {
		t.Fatalf("scroll should reset")
	}
}

func TestHandleInputAndLoad(t *testing.T) {
	gifData := testutil.MakeTestGIF()
	testutil.WithTransport(t, &testutil.FakeTransport{GIFData: gifData}, func() {
		state := &appState{
			query: "cats",
			mode:  modeQuery,
			cache: map[string]*gifdecode.Frames{},
			opts:  model.Options{Limit: 1, Source: "tenor"},
		}
		var buf bytes.Buffer
		out := bufio.NewWriter(&buf)

		handleInput(state, inputEvent{kind: keyEnter}, out)

		if state.mode != modeBrowse {
			t.Fatalf("expected browse mode")
		}
		if len(state.results) != 1 {
			t.Fatalf("expected results")
		}
		if state.currentAnim == nil {
			t.Fatalf("expected current animation")
		}

		state.results = append(state.results, model.Result{Title: "Second", PreviewURL: state.results[0].PreviewURL})
		state.selected = 0
		handleInput(state, inputEvent{kind: keyDown}, out)
		if state.selected != 1 {
			t.Fatalf("expected selection to move down")
		}

		state.mode = modeBrowse
		handleInput(state, inputEvent{kind: keyUp}, out)
		if state.selected != 0 {
			t.Fatalf("expected selection to move up")
		}

		state.mode = modeBrowse
		handleInput(state, inputEvent{kind: keyEnter}, out)
		if state.mode != modeQuery {
			t.Fatalf("expected query mode on enter")
		}

		handleInput(state, inputEvent{kind: keyRune, ch: '/'}, out)
		if state.mode != modeQuery {
			t.Fatalf("expected query mode")
		}

		state.results = []model.Result{{Title: "A"}}
		handleInput(state, inputEvent{kind: keyEsc}, out)
		if state.mode != modeBrowse {
			t.Fatalf("expected browse mode on esc")
		}

		state.mode = modeBrowse
		handleInput(state, inputEvent{kind: keyEsc}, out)
		if state.mode != modeQuery {
			t.Fatalf("expected query mode on esc")
		}

		state.mode = modeQuery
		state.query = "c"
		handleInput(state, inputEvent{kind: keyBackspace}, out)
		if state.query != "" {
			t.Fatalf("expected backspace to remove query")
		}

		state.mode = modeQuery
		if !handleInput(state, inputEvent{kind: keyRune, ch: 'q'}, out) {
			t.Fatalf("expected quit on q")
		}

		state.mode = modeQuery
		state.results = []model.Result{{Title: "A"}}
		handleInput(state, inputEvent{kind: keyEsc}, out)
		if state.mode != modeBrowse {
			t.Fatalf("expected browse mode on esc with results")
		}

		state.mode = modeBrowse
		state.selected = len(state.results) - 1
		handleInput(state, inputEvent{kind: keyDown}, out)
		if state.selected != len(state.results)-1 {
			t.Fatalf("expected selection to stay at end")
		}
	})
}

func TestReadInput(t *testing.T) {
	data := []byte{0x03, 'a', 0x7f, '\r', 0x1b, '[', 'A', 0x1b, '[', 'B', 0x1b, '[', 'C', 0x1b}
	r := bytes.NewReader(data)
	ch := make(chan inputEvent, 10)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		readInput(r, ch, stop)
		close(done)
	}()
	<-done
	close(ch)

	kinds := make([]keyKind, 0, 8)
	for ev := range ch {
		kinds = append(kinds, ev.kind)
	}
	if len(kinds) < 6 {
		t.Fatalf("expected events, got %d", len(kinds))
	}
}

func TestDrawPreview(t *testing.T) {
	frames := &gifdecode.Frames{
		Frames: []gifdecode.Frame{{PNG: []byte{1, 2, 3}, Delay: 80 * time.Millisecond}},
		Width:  2,
		Height: 2,
	}
	state := &appState{
		currentAnim: &gifAnimation{
			ID:     1,
			Frames: frames.Frames,
			Width:  frames.Width,
			Height: frames.Height,
		},
		previewNeedsSend: true,
		previewDirty:     true,
	}
	var buf bytes.Buffer
	out := bufio.NewWriter(&buf)
	drawPreview(state, out, 10, 5, 1, 1)
	_ = out.Flush()
	if !strings.Contains(buf.String(), "a=T") {
		t.Fatalf("expected kitty data")
	}
}

func TestRenderAndLines(t *testing.T) {
	state := &appState{
		query:   "cats",
		mode:    modeBrowse,
		results: []model.Result{{Title: "A cat"}, {Title: "B cat"}},
		status:  "10 results",
		opts:    model.Options{Source: "tenor"},
	}
	var buf bytes.Buffer
	out := bufio.NewWriter(&buf)
	render(state, out, 10, 40)
	_ = out.Flush()
	text := buf.String()
	if !strings.Contains(text, "gifgrep") {
		t.Fatalf("missing header")
	}
	if strings.Contains(text, "\x1b_G") {
		t.Fatalf("unexpected kitty graphics data")
	}

	buf.Reset()
	render(state, out, 20, 90)
	_ = out.Flush()
	if !strings.Contains(buf.String(), "[Search]") {
		t.Fatalf("expected search line")
	}

	buf.Reset()
	writeLine(out, "hello world", 5)
	_ = out.Flush()
	if buf.String() != "hello\x1b[K\r\n" {
		t.Fatalf("unexpected writeLine output: %q", buf.String())
	}

	buf.Reset()
	writeLine(out, "x", 0)
	_ = out.Flush()
	if buf.String() != "\r\n" {
		t.Fatalf("expected newline for width 0")
	}
}
