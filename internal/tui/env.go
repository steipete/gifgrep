package tui

import (
	"io"
	"os"

	"golang.org/x/term"
)

type Env struct {
	In         io.Reader
	Out        io.Writer
	FD         int
	IsTerminal func(int) bool
	MakeRaw    func(int) (*term.State, error)
	Restore    func(int, *term.State) error
	GetSize    func(int) (int, int, error)
	SignalCh   <-chan os.Signal
}

var defaultEnvFn = defaultEnv

func defaultEnv() Env {
	return Env{
		In:         os.Stdin,
		Out:        os.Stdout,
		FD:         int(os.Stdin.Fd()),
		IsTerminal: term.IsTerminal,
		MakeRaw:    term.MakeRaw,
		Restore:    term.Restore,
		GetSize:    term.GetSize,
	}
}

func SetDefaultEnvForTest(fn func() Env) {
	if fn == nil {
		defaultEnvFn = defaultEnv
		return
	}
	defaultEnvFn = fn
}
