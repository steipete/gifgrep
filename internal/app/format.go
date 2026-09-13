package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/termcaps"
	"golang.org/x/term"
)

type outputFormat string

const (
	formatAuto    outputFormat = "auto"
	formatPlain   outputFormat = "plain"
	formatTSV     outputFormat = "tsv"
	formatMD      outputFormat = "md"
	formatURL     outputFormat = "url"
	formatComment outputFormat = "comment"
	formatJSON    outputFormat = "json"
)

var isTerminalWriter = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

func resolveOutputFormat(opts model.Options, stdout io.Writer) outputFormat {
	if opts.JSON {
		return formatJSON
	}
	f := outputFormat(strings.ToLower(strings.TrimSpace(opts.Format)))
	if f == "" || f == formatAuto {
		if isTerminalWriter(stdout) {
			return formatPlain
		}
		return formatURL
	}
	return f
}

func normalizeTitle(res model.Result) string {
	label := strings.Join(strings.Fields(res.Title), " ")
	if label == "" {
		label = strings.Join(strings.Fields(res.ID), " ")
	}
	if label == "" {
		label = "untitled"
	}
	return label
}

func writeSearchResults(out *bufio.Writer, opts model.Options, useColor bool, thumbs termcaps.InlineProtocol, results []model.Result, termCols int, format outputFormat) {
	switch format {
	case formatPlain:
		renderPlain(out, opts, useColor, thumbs, results, termCols)
		return
	case formatURL:
		for i, res := range results {
			url := res.URL
			if opts.Number {
				_, _ = fmt.Fprintf(out, "%d\t%s\n", i+1, url)
				continue
			}
			_, _ = fmt.Fprintln(out, url)
		}
		return
	case formatMD:
		for i, res := range results {
			title := normalizeTitle(res)
			url := res.URL
			prefix := "- "
			if opts.Number {
				prefix = fmt.Sprintf("%d. ", i+1)
			}
			_, _ = fmt.Fprintf(out, "%s[%s](%s)\n", prefix, title, url)
		}
		return
	case formatComment:
		for i, res := range results {
			title := normalizeTitle(res)
			url := res.URL
			if opts.Number {
				_, _ = fmt.Fprintf(out, "%d\t%s  # %s\n", i+1, url, title)
				continue
			}
			_, _ = fmt.Fprintf(out, "%s  # %s\n", url, title)
		}
		return
	case formatJSON:
		// handled by caller
		return
	case formatTSV, formatAuto:
		fallthrough
	default:
		for i, res := range results {
			title := normalizeTitle(res)
			url := res.URL
			if useColor {
				title = "\x1b[1m" + title + "\x1b[0m"
				url = "\x1b[36m" + url + "\x1b[0m"
			}
			if opts.Number {
				_, _ = fmt.Fprintf(out, "%d\t%s\t%s\n", i+1, title, url)
				continue
			}
			_, _ = fmt.Fprintf(out, "%s\t%s\n", title, url)
		}
		return
	}
}
