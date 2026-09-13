package tui

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestHelpers(t *testing.T) {
	if truncateANSI("héllö", 3) != "hél" {
		t.Fatalf("truncateANSI failed")
	}
	if truncateANSI("hello", 0) != "" {
		t.Fatalf("truncateANSI width 0 failed")
	}
	if visibleRuneLen("héllö") != 5 {
		t.Fatalf("visibleRuneLen failed")
	}
	var buf bytes.Buffer
	out := bufio.NewWriter(&buf)
	moveCursor(out, 2, 3)
	saveCursor(out)
	restoreCursor(out)
	hideCursor(out)
	showCursor(out)
	clearImages(out)
	_ = out.Flush()
	s := buf.String()
	if !strings.Contains(s, "\x1b[2;3H") || !strings.Contains(s, "\x1b[?25l") || !strings.Contains(s, "\x1b[?25h") {
		t.Fatalf("cursor sequences missing")
	}
	if !strings.Contains(s, "\x1b_Ga=d\x1b\\") {
		t.Fatalf("clear images missing")
	}
	buf.Reset()
	moveCursor(out, 0, 0)
	_ = out.Flush()
	if !strings.Contains(buf.String(), "\x1b[1;1H") {
		t.Fatalf("expected clamped cursor move")
	}

	t.Setenv("GIFGREP_CELL_ASPECT", "0.7")
	if cellAspectRatio() != 0.7 {
		t.Fatalf("cellAspectRatio env override failed")
	}

	t.Setenv("GIFGREP_SOFTWARE_ANIM", "true")
	if !useSoftwareAnimation() {
		t.Fatalf("expected software animation")
	}

	t.Setenv("GIFGREP_SOFTWARE_ANIM", "")
	t.Setenv("TERM_PROGRAM", "xterm")
	t.Setenv("TERM", "xterm-256color")
	if useSoftwareAnimation() {
		t.Fatalf("expected software animation to be false")
	}

	if got := truncateANSI("\x1b[31mhello\x1b[0m", 3); got != "\x1b[31mhel\x1b[0m" {
		t.Fatalf("truncateANSI failed: %q", got)
	}
	if visibleRuneLen("\x1b[31mhi\x1b[0m") != 2 {
		t.Fatalf("visibleRuneLen failed")
	}
	// Ensure wide runes don't break centering/truncation.
	if visibleRuneLen("🧲") != 2 {
		t.Fatalf("expected magnet emoji to be width 2")
	}
}
