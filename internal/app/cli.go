package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/reveal"
	"github.com/steipete/gifgrep/internal/search"
	"github.com/steipete/gifgrep/internal/tui"
	"golang.org/x/term"
)

type CLI struct {
	Globals Globals `embed:""`

	Search SearchCmd `cmd:"" default:"withargs" help:"Search and print GIF URLs."`
	TUI    TUICmd    `cmd:"" help:"Interactive browser with inline preview."`
	Still  StillCmd  `cmd:"" help:"Extract a single frame as PNG."`
	Sheet  SheetCmd  `cmd:"" help:"Generate a sheet PNG of sampled frames."`
}

type Globals struct {
	Color   string           `help:"Color output." enum:"auto,always,never" default:"auto"`
	NoColor bool             `help:"Disable color output."`
	Reveal  bool             `help:"Reveal output file in file manager."`
	Verbose int              `help:"Verbose stderr logs (-vv for more)." short:"v" type:"counter"`
	Quiet   bool             `help:"Suppress non-essential stderr output." short:"q"`
	Version kong.VersionFlag `help:"Show version."`
}

func (g Globals) toOptions() model.Options {
	color := g.Color
	if g.NoColor {
		color = "never"
	}
	return model.Options{
		Color:   color,
		Reveal:  g.Reveal,
		Verbose: g.Verbose,
		Quiet:   g.Quiet,
	}
}

type SearchCmd struct {
	Source string `help:"Source to search." enum:"auto,tenor,giphy" default:"auto"`
	Max    int    `help:"Max results to fetch." name:"max" short:"m" default:"20"`
	JSON   bool   `help:"Emit JSON array of results."`
	Number bool   `help:"Prefix lines with 1-based index." short:"n"`

	Query []string `arg:"" name:"query" help:"Search query."`
}

func (c *SearchCmd) Run(ctx *kong.Context, cli *CLI) error {
	query := strings.TrimSpace(strings.Join(c.Query, " "))
	if query == "" {
		return errors.New("missing query")
	}

	opts := cli.Globals.toOptions()
	opts.JSON = c.JSON
	opts.Number = c.Number
	opts.Limit = c.Max
	opts.Source = c.Source
	return runSearch(ctx.Stdout, ctx.Stderr, opts, query)
}

type TUICmd struct {
	Source string `help:"Source to search." enum:"auto,tenor,giphy" default:"auto"`
	Max    int    `help:"Max results to fetch." name:"max" short:"m" default:"20"`

	Query []string `arg:"" optional:"" name:"query" help:"Initial query."`
}

func (c *TUICmd) Run(_ *kong.Context, cli *CLI) error {
	opts := cli.Globals.toOptions()
	opts.Limit = c.Max
	opts.Source = c.Source

	query := strings.TrimSpace(strings.Join(c.Query, " "))
	return tui.Run(opts, query)
}

type StillCmd struct {
	GIF    string        `arg:"" name:"gif" help:"GIF path or URL."`
	At     DurationValue `help:"Timestamp (e.g. 1.5s or 1.5)." name:"at" required:""`
	Output string        `help:"Output path or '-' for stdout." name:"output" short:"o" default:"still.png"`
}

func (c *StillCmd) Run(_ *kong.Context, cli *CLI) error {
	opts := cli.Globals.toOptions()
	opts.GifInput = c.GIF
	opts.StillSet = true
	opts.StillAt = time.Duration(c.At)
	opts.OutPath = c.Output
	opts.StillsCount = 0
	if err := runExtract(opts); err != nil {
		return err
	}
	if opts.Reveal {
		outPath := resolveExtractOutPath(opts)
		if outPath != "-" {
			return reveal.Reveal(outPath)
		}
	}
	return nil
}

type SheetCmd struct {
	GIF     string `arg:"" name:"gif" help:"GIF path or URL."`
	Frames  int    `help:"Number of frames to sample." name:"frames" default:"12"`
	Cols    int    `help:"Columns (0 = auto)." name:"cols" default:"0"`
	Padding int    `help:"Padding between frames (px)." name:"padding" default:"2"`
	Output  string `help:"Output path or '-' for stdout." name:"output" short:"o" default:"sheet.png"`
}

func (c *SheetCmd) Run(_ *kong.Context, cli *CLI) error {
	opts := cli.Globals.toOptions()
	opts.GifInput = c.GIF
	opts.StillSet = false
	opts.StillsCount = c.Frames
	opts.StillsCols = c.Cols
	opts.StillsPadding = c.Padding
	opts.OutPath = c.Output
	if err := runExtract(opts); err != nil {
		return err
	}
	if opts.Reveal {
		outPath := resolveExtractOutPath(opts)
		if outPath != "-" {
			return reveal.Reveal(outPath)
		}
	}
	return nil
}

func runSearch(stdout io.Writer, stderr io.Writer, opts model.Options, query string) error {
	if strings.TrimSpace(query) == "" {
		return errors.New("missing query")
	}
	if opts.Verbose > 0 && !opts.Quiet {
		_, _ = fmt.Fprintf(stderr, "source=%s max=%d\n", search.ResolveSource(opts.Source), opts.Limit)
	}

	results, err := search.Search(query, opts)
	if err != nil {
		return err
	}

	if opts.JSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	useColor := shouldUseColor(opts, stdout)
	for i, res := range results {
		prefix := ""
		if opts.Number {
			prefix = fmt.Sprintf("%d\t", i+1)
		}
		label := strings.Join(strings.Fields(res.Title), " ")
		if label == "" {
			label = strings.Join(strings.Fields(res.ID), " ")
		}
		if label == "" {
			label = "untitled"
		}
		url := res.URL
		if useColor {
			label = "\x1b[1m" + label + "\x1b[0m"
			url = "\x1b[36m" + url + "\x1b[0m"
		}
		_, _ = fmt.Fprintf(stdout, "%s%s\t%s\n", prefix, label, url)
	}
	return nil
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
