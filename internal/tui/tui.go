package tui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/search"
	"github.com/steipete/gifgrep/internal/termcaps"
	"golang.org/x/term"
)

var ErrNotTerminal = errors.New("stdin is not a tty")

var nowFn = time.Now

func Run(opts model.Options, query string) error {
	env := defaultEnvFn()
	return runWith(env, opts, query)
}

func initEnvDefaults(env Env) (Env, error) {
	if env.In == nil {
		env.In = os.Stdin
	}
	if env.Out == nil {
		env.Out = os.Stdout
	}
	if env.IsTerminal == nil {
		env.IsTerminal = term.IsTerminal
	}
	if env.MakeRaw == nil {
		env.MakeRaw = term.MakeRaw
	}
	if env.Restore == nil {
		env.Restore = term.Restore
	}
	if env.GetSize == nil {
		env.GetSize = term.GetSize
	}
	if env.FD == 0 {
		env.FD = int(os.Stdin.Fd())
	}
	if !env.IsTerminal(env.FD) {
		return env, ErrNotTerminal
	}
	return env, nil
}

func detectInlineProtocol() (termcaps.InlineProtocol, error) {
	inline := termcaps.DetectInlineRobust(os.Getenv)
	if inline == termcaps.InlineNone {
		return inline, errUnsupportedInline(os.Getenv)
	}
	return inline, nil
}

func setupOutput(out *bufio.Writer, inline termcaps.InlineProtocol) func() {
	hideCursor(out)
	return func() {
		showCursor(out)
		if inline == termcaps.InlineKitty {
			clearImages(out)
		}
		_ = out.Flush()
	}
}

func setupSignals(env Env) (<-chan os.Signal, func()) {
	if env.SignalCh != nil {
		return env.SignalCh, func() {}
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	return sigs, func() { signal.Stop(sigs) }
}

func setupInputReader(in io.Reader) (chan inputEvent, chan struct{}) {
	inputCh := make(chan inputEvent, 16)
	stopCh := make(chan struct{})
	go func() {
		defer close(inputCh)
		readInput(in, inputCh, stopCh)
	}()
	return inputCh, stopCh
}

func newAppState(inline termcaps.InlineProtocol, opts model.Options) *appState {
	return &appState{
		mode:            modeQuery,
		status:          "Type a search and press Enter",
		tagline:         pickTagline(time.Now(), os.Getenv, nil),
		cache:           map[string]*gifCacheEntry{},
		savedPaths:      map[string]string{},
		tempPaths:       map[string]string{},
		prefetching:     map[string]bool{},
		renderDirty:     true,
		nextImageID:     1,
		inline:          inline,
		useSoftwareAnim: inline == termcaps.InlineSixel || inline == termcaps.InlineANSI || (inline == termcaps.InlineKitty && useSoftwareAnimation()),
		useColor:        opts.Color != "never",
		opts:            opts,
	}
}

func runInitialSearch(state *appState, query string, out *bufio.Writer, prefetchCh chan<- prefetchResult) {
	if strings.TrimSpace(query) == "" {
		return
	}
	state.query = query
	state.mode = modeBrowse
	searchResults(state, out, prefetchCh)
}

func searchResults(state *appState, out *bufio.Writer, prefetchCh chan<- prefetchResult) {
	state.status = "Searching..."
	render(state, out, state.lastRows, state.lastCols)
	_ = out.Flush()

	results, source, err := search.Search(state.query, state.opts)
	if err != nil {
		state.status = "Search error: " + err.Error()
		state.renderDirty = true
		return
	}

	state.results = results
	state.source = source
	state.selected = 0
	state.scroll = 0
	resetPrefetch(state)
	state.cache = map[string]*gifCacheEntry{}
	if len(results) == 0 {
		state.status = "No results"
		state.currentAnim = nil
		state.previewDirty = true
		state.renderDirty = true
		return
	}

	state.status = fmt.Sprintf("%d results", len(results))
	loadSelectedImage(state)
	startPrefetch(state, results, prefetchCh)
	state.renderDirty = true
}

func handlePrefetchResult(state *appState, res prefetchResult) {
	if !acceptPrefetchResult(state, res) {
		return
	}
	if state.selected < 0 || state.selected >= len(state.results) {
		return
	}
	if resultKey(state.results[state.selected]) != res.key {
		return
	}
	loadSelectedImage(state)
	state.renderDirty = true
}

func updateSizeIfNeeded(state *appState, env Env) {
	cols, rows := terminalSize(env)
	if rows <= 0 || cols <= 0 {
		return
	}
	if rows != state.lastRows || cols != state.lastCols {
		state.lastRows = rows
		state.lastCols = cols
		state.renderDirty = true
		state.previewDirty = true
	}
}

func terminalSize(env Env) (int, int) {
	if cols, rows, err := env.GetSize(env.FD); err == nil && cols > 0 && rows > 0 {
		return cols, rows
	}
	cols, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("COLUMNS")))
	rows, _ := strconv.Atoi(strings.TrimSpace(os.Getenv("LINES")))
	if cols > 0 && rows > 0 {
		return cols, rows
	}
	return 0, 0
}

func renderIfNeeded(state *appState, out *bufio.Writer) {
	if !state.renderDirty {
		return
	}
	render(state, out, state.lastRows, state.lastCols)
	state.renderDirty = false
	_ = out.Flush()
}

func handleEvents(state *appState, out *bufio.Writer, prefetchCh chan prefetchResult, inputCh <-chan inputEvent, stopCh chan struct{}, sigs <-chan os.Signal, ticker *time.Ticker) bool {
	select {
	case <-sigs:
		close(stopCh)
		return true
	case ev, ok := <-inputCh:
		if !ok {
			close(stopCh)
			return true
		}
		if handleInput(state, ev, out, prefetchCh) {
			close(stopCh)
			return true
		}
	case res := <-prefetchCh:
		handlePrefetchResult(state, res)
	case <-ticker.C:
	}
	return false
}

func errUnsupportedInline(getenv func(string) string) error {
	if getenv == nil {
		getenv = os.Getenv
	}
	termProgram := strings.TrimSpace(getenv("TERM_PROGRAM"))
	term := strings.TrimSpace(getenv("TERM"))
	return fmt.Errorf(
		"gifgrep tui needs inline image support.\n\nSupported terminals:\n  - Kitty (Kitty graphics protocol)\n  - Ghostty (Kitty graphics protocol)\n  - iTerm2 (OSC 1337 inline images)\n  - Windows Terminal / WezTerm with Sixel\n  - Truecolor ANSI fallback\n\nDetected:\n  TERM_PROGRAM=%q\n  TERM=%q\n\nSee: docs/kitty.md, docs/iterm.md, and docs/sixel.md\n\nTip: You can force detection with GIFGREP_INLINE=kitty|iterm|sixel|ansi|none",
		termProgram,
		term,
	)
}

func runWith(env Env, opts model.Options, query string) error {
	var err error
	env, err = initEnvDefaults(env)
	if err != nil {
		return err
	}

	inline, err := detectInlineProtocol()
	if err != nil {
		return err
	}

	oldState, err := env.MakeRaw(env.FD)
	if err != nil {
		return err
	}
	if oldState != nil {
		defer func() {
			_ = env.Restore(env.FD, oldState)
		}()
	}

	out := bufio.NewWriter(env.Out)
	defer setupOutput(out, inline)()

	sigs, stopSignals := setupSignals(env)
	defer stopSignals()
	inputCh, stopCh := setupInputReader(env.In)
	prefetchCh := make(chan prefetchResult, 64)

	state := newAppState(inline, opts)
	defer cleanupTempDir(state)
	if cols, rows := terminalSize(env); cols > 0 && rows > 0 {
		state.lastRows = rows
		state.lastCols = cols
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	runInitialSearch(state, query, out, prefetchCh)

	for {
		if handleEvents(state, out, prefetchCh, inputCh, stopCh, sigs, ticker) {
			return nil
		}
		updateSizeIfNeeded(state, env)
		renderIfNeeded(state, out)

		advanceManualAnimation(state, out)
	}
}
