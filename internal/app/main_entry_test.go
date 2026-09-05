package app

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/steipete/gifgrep/internal/testutil"
	"github.com/steipete/gifgrep/internal/tui"
	"golang.org/x/term"
)

func TestRunArgs(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		if code := Run([]string{"--version"}); code != 0 {
			t.Fatalf("expected exit 0")
		}
	})

	t.Run("help", func(t *testing.T) {
		if code := Run([]string{"--help"}); code != 0 {
			t.Fatalf("expected exit 0")
		}
	})

	t.Run("empty", func(t *testing.T) {
		if code := Run(nil); code != 2 {
			t.Fatalf("expected exit 2")
		}
	})

	t.Run("bad args", func(t *testing.T) {
		if code := Run([]string{"--nope"}); code != 2 {
			t.Fatalf("expected exit 2")
		}
	})

	t.Run("bad source", func(t *testing.T) {
		if code := Run([]string{"search", "--source", "nope", "cats"}); code != 2 {
			t.Fatalf("expected exit 2")
		}
	})

	t.Run("tui", func(t *testing.T) {
		t.Cleanup(func() { tui.SetDefaultEnvForTest(nil) })
		t.Setenv("GIFGREP_INLINE", "kitty")
		tui.SetDefaultEnvForTest(func() tui.Env {
			return tui.Env{
				In:         bytes.NewReader([]byte{0x03}),
				Out:        io.Discard,
				FD:         1,
				IsTerminal: func(int) bool { return true },
				MakeRaw:    func(int) (*term.State, error) { return &term.State{}, nil },
				Restore:    func(int, *term.State) error { return nil },
				GetSize:    func(int) (int, int, error) { return 80, 24, nil },
				SignalCh:   make(chan os.Signal),
			}
		})
		if code := Run([]string{"tui"}); code != 0 {
			t.Fatalf("expected exit 0")
		}
	})

	t.Run("extract still", func(t *testing.T) {
		tmp, err := os.CreateTemp(t.TempDir(), "gifgrep-*.gif")
		if err != nil {
			t.Fatalf("temp file: %v", err)
		}
		if _, err := tmp.Write(testutil.MakeTestGIF()); err != nil {
			t.Fatalf("write gif: %v", err)
		}
		if err := tmp.Close(); err != nil {
			t.Fatalf("close gif: %v", err)
		}

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w
		t.Cleanup(func() {
			os.Stdout = oldStdout
		})

		code := Run([]string{"still", tmp.Name(), "--at", "0", "--output", "-"})
		_ = w.Close()
		if code != 0 {
			t.Fatalf("expected exit 0")
		}
		out, _ := io.ReadAll(r)
		if !bytes.HasPrefix(out, []byte{0x89, 'P', 'N', 'G'}) {
			t.Fatalf("expected png output")
		}
	})
}

func TestRunTUIExitCodes(t *testing.T) {
	t.Cleanup(func() { tui.SetDefaultEnvForTest(nil) })

	t.Run("not terminal", func(t *testing.T) {
		tui.SetDefaultEnvForTest(func() tui.Env {
			return tui.Env{
				In:         bytes.NewReader(nil),
				Out:        io.Discard,
				FD:         1,
				IsTerminal: func(int) bool { return false },
			}
		})
		if code := Run([]string{"tui"}); code != 1 {
			t.Fatalf("expected exit 1")
		}
	})

	t.Run("makeRaw fails", func(t *testing.T) {
		tui.SetDefaultEnvForTest(func() tui.Env {
			return tui.Env{
				In:         bytes.NewReader(nil),
				Out:        io.Discard,
				FD:         1,
				IsTerminal: func(int) bool { return true },
				MakeRaw: func(int) (*term.State, error) {
					return nil, errors.New("boom")
				},
			}
		})
		if code := Run([]string{"tui"}); code != 1 {
			t.Fatalf("expected exit 1")
		}
	})
}
