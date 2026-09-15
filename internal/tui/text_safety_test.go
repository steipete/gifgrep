package tui

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/termcaps"
)

func TestRenderEscapesUntrustedTextBeforeStyling(t *testing.T) {
	const payload = "\x1b]52;c;c3ludGhldGlj\a\x1b[2J\u009b31m"
	for _, useColor := range []bool{false, true} {
		for _, field := range []string{"title", "id", "query", "status", "header"} {
			t.Run(field+map[bool]string{true: " colored", false: " plain"}[useColor], func(t *testing.T) {
				state := newAppState(termcaps.InlineANSI, model.Options{})
				state.useColor = useColor
				state.results = []model.Result{{Title: "Café", ID: "fixture"}}
				switch field {
				case "title":
					state.results[0].Title = payload
				case "id":
					state.results[0].Title = ""
					state.results[0].ID = payload
				case "query":
					state.query = payload
				case "status":
					state.status = payload
				case "header":
					state.headerFlash = payload
				}
				var buf bytes.Buffer
				out := bufio.NewWriter(&buf)
				render(state, out, 24, 120)
				if err := out.Flush(); err != nil {
					t.Fatal(err)
				}
				got := buf.String()
				if strings.Contains(got, "\x1b]52;") || strings.Contains(got, "\x1b[2J") || strings.ContainsAny(got, "\a\u009b") {
					t.Fatalf("untrusted control emitted: %q", got)
				}
				if !strings.Contains(got, "\\x1b") {
					t.Fatalf("missing visible escape: %q", got)
				}
				if useColor && !strings.Contains(got, "\x1b[1m") {
					t.Fatal("lost application styling")
				}
			})
		}
	}
}
