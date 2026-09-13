package app

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/steipete/gifgrep/internal/download"
	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/reveal"
	"github.com/steipete/gifgrep/internal/search"
	"github.com/steipete/gifgrep/internal/termcaps"
	"golang.org/x/term"
)

func runSearch(stdout io.Writer, stderr io.Writer, opts model.Options, query string) error {
	if strings.TrimSpace(query) == "" {
		return errors.New("missing query")
	}
	logSearchConfig(stderr, opts)

	results, err := search.Search(query, opts)
	if err != nil {
		return err
	}

	if err := downloadSearchResults(results, opts, stderr); err != nil {
		return err
	}

	format := resolveOutputFormat(opts, stdout)
	out := bufio.NewWriter(stdout)
	if format == formatJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			return err
		}
	} else {
		useColor := shouldUseColor(opts, stdout)
		thumbs := thumbsProtocol(opts, stdout, format)
		termCols := termColumns(stdout, thumbs)
		writeSearchResults(out, opts, useColor, thumbs, results, termCols, format)
	}
	return out.Flush()
}

func logSearchConfig(stderr io.Writer, opts model.Options) {
	if opts.Verbose > 0 && !opts.Quiet {
		_, _ = fmt.Fprintf(stderr, "source=%s max=%d\n", search.ResolveSource(opts.Source), opts.Limit)
	}
}

func downloadSearchResults(results []model.Result, opts model.Options, stderr io.Writer) error {
	if !opts.Download {
		return nil
	}

	var lastSaved string
	for _, res := range results {
		if res.URL == "" {
			continue
		}
		savedPath, err := download.ToDownloads(res, opts.Cache)
		if err != nil {
			return err
		}
		lastSaved = savedPath
		if opts.Verbose > 0 && !opts.Quiet {
			_, _ = fmt.Fprintf(stderr, "saved %s\n", savedPath)
		}
	}
	if opts.Reveal && lastSaved != "" {
		return reveal.Reveal(lastSaved)
	}
	return nil
}

func termColumns(w io.Writer, thumbs termcaps.InlineProtocol) int {
	if thumbs == termcaps.InlineNone {
		return 0
	}
	f, ok := w.(*os.File)
	if !ok {
		return 0
	}
	cols, _, err := term.GetSize(int(f.Fd()))
	if err != nil || cols <= 0 {
		return 0
	}
	return cols
}

func shouldUseColor(opts model.Options, w io.Writer) bool {
	if opts.Color == "never" {
		return false
	}
	if opts.Color == "always" {
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	termEnv := strings.ToLower(strings.TrimSpace(os.Getenv("TERM")))
	if termEnv == "dumb" || termEnv == "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
