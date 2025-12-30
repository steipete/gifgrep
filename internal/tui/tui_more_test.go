package tui

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/testutil"
	"golang.org/x/term"
)

func TestRunTUIWithDefaultsNotTerminal(t *testing.T) {
	if err := runWith(Env{}, model.Options{}, ""); !errors.Is(err, ErrNotTerminal) {
		t.Fatalf("expected errNotTerminal")
	}
}

func TestRunTUIWithNoRestore(t *testing.T) {
	env := Env{
		In:         bytes.NewReader([]byte("q")),
		Out:        io.Discard,
		FD:         1,
		IsTerminal: func(int) bool { return true },
		MakeRaw:    func(int) (*term.State, error) { return &term.State{}, nil },
		GetSize:    func(int) (int, int, error) { return 80, 24, nil },
		SignalCh:   make(chan os.Signal),
	}
	if err := runWith(env, model.Options{}, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunTUIWithSearchError(t *testing.T) {
	env := Env{
		In:         bytes.NewReader([]byte("q")),
		Out:        io.Discard,
		FD:         1,
		IsTerminal: func(int) bool { return true },
		MakeRaw:    func(int) (*term.State, error) { return &term.State{}, nil },
		Restore:    func(int, *term.State) error { return nil },
		GetSize:    func(int) (int, int, error) { return 80, 24, nil },
		SignalCh:   make(chan os.Signal),
	}
	if err := runWith(env, model.Options{Source: "nope"}, "cats"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunTUIWithSizeError(t *testing.T) {
	env := Env{
		In:         bytes.NewReader([]byte("q")),
		Out:        io.Discard,
		FD:         1,
		IsTerminal: func(int) bool { return true },
		MakeRaw:    func(int) (*term.State, error) { return &term.State{}, nil },
		Restore:    func(int, *term.State) error { return nil },
		GetSize:    func(int) (int, int, error) { return 0, 0, errors.New("bad") },
		SignalCh:   make(chan os.Signal),
	}
	if err := runWith(env, model.Options{}, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type emptyTenorTransport struct{}

func (t *emptyTenorTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	body := `{"results":[]}`
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}, nil
}

func TestRunTUIWithEmptyResultsAndSignal(t *testing.T) {
	testutil.WithTransport(t, &emptyTenorTransport{}, func() {
		sigs := make(chan os.Signal, 1)
		sigs <- os.Interrupt
		env := Env{
			In:         bytes.NewReader([]byte("q")),
			Out:        io.Discard,
			FD:         1,
			IsTerminal: func(int) bool { return true },
			MakeRaw:    func(int) (*term.State, error) { return &term.State{}, nil },
			Restore:    func(int, *term.State) error { return nil },
			GetSize:    func(int) (int, int, error) { return 80, 24, nil },
			SignalCh:   sigs,
		}
		if err := runWith(env, model.Options{Source: "tenor"}, "cats"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
