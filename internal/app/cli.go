package app

import (
	"errors"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/reveal"
	"github.com/steipete/gifgrep/internal/tui"
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
	CacheFlags `embed:""`

	Source   string `help:"Source to search." enum:"auto,klipy,tenor,giphy" default:"auto"`
	Max      int    `help:"Max results to fetch." name:"max" short:"m" default:"20"`
	JSON     bool   `help:"Emit JSON array of results."`
	Number   bool   `help:"Prefix lines with 1-based index." short:"n"`
	Download bool   `help:"Download results to ~/Downloads."`
	Format   string `help:"Output format." enum:"auto,plain,tsv,md,url,comment,json" default:"auto"`
	Thumbs   string `help:"Inline thumbnails (Kitty protocol / iTerm2 images; TTY only)." enum:"auto,always,never" default:"auto"`

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
	opts.Format = c.Format
	opts.Thumbs = c.Thumbs
	opts.Download = c.Download
	cache, err := c.options()
	if err != nil {
		return err
	}
	opts.Cache = cache
	return runSearch(ctx.Stdout, ctx.Stderr, opts, query)
}

type TUICmd struct {
	CacheFlags `embed:""`

	Source string `help:"Source to search." enum:"auto,klipy,tenor,giphy" default:"auto"`
	Max    int    `help:"Max results to fetch." name:"max" short:"m" default:"20"`

	Query []string `arg:"" optional:"" name:"query" help:"Initial query."`
}

func (c *TUICmd) Run(_ *kong.Context, cli *CLI) error {
	opts := cli.Globals.toOptions()
	opts.Limit = c.Max
	opts.Source = c.Source
	cache, err := c.options()
	if err != nil {
		return err
	}
	opts.Cache = cache

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
