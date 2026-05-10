package tui

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/testutil"
	"golang.org/x/term"
)

func TestRunTUIWithQuit(t *testing.T) {
	t.Setenv("GIFGREP_INLINE", "kitty")
	var restored bool
	env := Env{
		In:  bytes.NewReader([]byte("q")),
		Out: io.Discard,
		FD:  1,
		IsTerminal: func(int) bool {
			return true
		},
		MakeRaw: func(int) (*term.State, error) {
			return &term.State{}, nil
		},
		Restore: func(int, *term.State) error {
			restored = true
			return nil
		},
		GetSize: func(int) (int, int, error) {
			return 80, 24, nil
		},
		SignalCh: make(chan os.Signal),
	}

	if err := runWith(env, model.Options{Source: "tenor"}, ""); err != nil {
		t.Fatalf("runTUIWith failed: %v", err)
	}
	if !restored {
		t.Fatalf("expected restore to be called")
	}
}

func TestRunTUIWithSearch(t *testing.T) {
	t.Setenv("GIFGREP_INLINE", "kitty")
	t.Setenv("GIFGREP_TUI_PREFETCH_MAX_BYTES", "0")
	t.Setenv("KLIPY_API_KEY", "test-key")
	gifData := testutil.MakeTestGIF()
	testutil.WithTransport(t, &testutil.FakeTransport{GIFData: gifData}, func() {
		env := Env{
			In:  bytes.NewReader([]byte("q")),
			Out: io.Discard,
			FD:  1,
			IsTerminal: func(int) bool {
				return true
			},
			MakeRaw: func(int) (*term.State, error) {
				return &term.State{}, nil
			},
			Restore: func(int, *term.State) error {
				return nil
			},
			GetSize: func(int) (int, int, error) {
				return 80, 24, nil
			},
			SignalCh: make(chan os.Signal),
		}

		if err := runWith(env, model.Options{Source: "tenor", Limit: 1}, "cats"); err != nil {
			t.Fatalf("runTUIWith search failed: %v", err)
		}
	})
}

func TestRunTUIWithNotTerminal(t *testing.T) {
	env := Env{
		In:  bytes.NewReader(nil),
		Out: io.Discard,
		FD:  1,
		IsTerminal: func(int) bool {
			return false
		},
	}

	if err := runWith(env, model.Options{}, ""); !errors.Is(err, ErrNotTerminal) {
		t.Fatalf("expected errNotTerminal, got %v", err)
	}
}
