package app

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/steipete/gifgrep/gifdecode"
	"github.com/steipete/gifgrep/internal/iterm"
	"github.com/steipete/gifgrep/internal/kitty"
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

type thumbsMode string

const (
	thumbsAuto   thumbsMode = "auto"
	thumbsAlways thumbsMode = "always"
	thumbsNever  thumbsMode = "never"
)

var isTerminalWriter = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

var (
	fetchThumb  = fetchURL
	decodeThumb = func(data []byte) (*gifdecode.Frames, error) {
		decodeOpts := gifdecode.DefaultOptions()
		decodeOpts.MaxFrames = 1
		return gifdecode.Decode(data, decodeOpts)
	}
	sendThumbKitty = func(out *bufio.Writer, id uint32, frame gifdecode.Frame, cols, rows int) {
		kitty.SendFrame(out, id, frame, cols, rows)
	}
	sendThumbIterm = func(out *bufio.Writer, data []byte, cols, rows int) {
		iterm.SendInlineFile(out, iterm.File{
			Name:        "thumb.png",
			Data:        data,
			WidthCells:  cols,
			HeightCells: rows,
		})
	}
)

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

func resolveThumbsMode(opts model.Options) thumbsMode {
	m := thumbsMode(strings.ToLower(strings.TrimSpace(opts.Thumbs)))
	if m == "" {
		return thumbsAuto
	}
	return m
}

func thumbsProtocol(opts model.Options, stdout io.Writer, format outputFormat) termcaps.InlineProtocol {
	if format != formatPlain {
		return termcaps.InlineNone
	}
	if !isTerminalWriter(stdout) {
		return termcaps.InlineNone
	}
	switch resolveThumbsMode(opts) {
	case thumbsNever:
		return termcaps.InlineNone
	case thumbsAlways:
		return termcaps.DetectInlineRobust(os.Getenv)
	case thumbsAuto:
		return termcaps.DetectInlineRobust(os.Getenv)
	}
	return termcaps.DetectInlineRobust(os.Getenv)
}

func renderPlain(
	out *bufio.Writer,
	opts model.Options,
	useColor bool,
	thumbs termcaps.InlineProtocol,
	results []model.Result,
) {
	nextID := uint32(1)
	withThumbs := thumbs != termcaps.InlineNone
	for i, res := range results {
		title := normalizeTitle(res)
		url := res.URL

		nPrefix := ""
		if opts.Number {
			nPrefix = fmt.Sprintf("%d. ", i+1)
		}

		if withThumbs {
			if err := renderThumbBlock(out, thumbs, nextID, res, nPrefix, title, url, useColor); err != nil {
				withThumbs = false
			} else {
				nextID++
				continue
			}
		}

		if useColor {
			title = "\x1b[1m" + nPrefix + title + "\x1b[0m"
			url = "\x1b[36m" + url + "\x1b[0m"
		} else {
			title = nPrefix + title
		}
		_, _ = fmt.Fprintln(out, title)
		_, _ = fmt.Fprintln(out, "  "+url)
		_, _ = fmt.Fprintln(out)
	}
}

func renderThumbBlock(out *bufio.Writer, thumbs termcaps.InlineProtocol, id uint32, res model.Result, nPrefix, title, url string, useColor bool) error {
	src := res.PreviewURL
	if src == "" {
		src = res.URL
	}
	data, err := fetchThumb(src)
	if err != nil {
		return err
	}
	decoded, err := decodeThumb(data)
	if err != nil {
		return err
	}
	if decoded == nil || len(decoded.Frames) == 0 {
		return fmt.Errorf("no frames")
	}

	cols := 16
	rows := 8
	if res.Width > 0 && res.Height > 0 {
		rows = clampInt(3, 10, int(float64(cols)*0.5*float64(res.Height)/float64(res.Width)))
	}

	switch thumbs {
	case termcaps.InlineNone:
		return fmt.Errorf("inline thumbnails not supported")
	case termcaps.InlineIterm:
		sendThumbIterm(out, decoded.Frames[0].PNG, cols, rows)
	case termcaps.InlineKitty:
		sendThumbKitty(out, id, decoded.Frames[0], cols, rows)
	}
	for r := 0; r < rows; r++ {
		line := ""
		switch r {
		case 0:
			line = nPrefix + title
			if useColor {
				line = "\x1b[1m" + line + "\x1b[0m"
			}
		case 1:
			line = url
			if useColor {
				line = "\x1b[36m" + line + "\x1b[0m"
			}
		default:
		}
		_, _ = fmt.Fprint(out, strings.Repeat(" ", cols+2))
		_, _ = fmt.Fprintln(out, line)
	}
	return nil
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

func clampInt(minVal, maxVal, v int) int {
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}
