package app

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/steipete/gifgrep/internal/model"
)

type exitPanic struct {
	code int
}

func Run(args []string) int {
	if args == nil {
		args = []string{}
	}

	cli := &CLI{}
	parser, err := kong.New(
		cli,
		kong.Name(model.AppName),
		kong.Vars{"version": model.AppName + " " + model.Version},
		kong.Help(helpPrinter),
		kong.ConfigureHelp(kong.HelpOptions{
			WrapUpperBound: 100,
		}),
		kong.Exit(func(code int) {
			panic(exitPanic{code: code})
		}),
	)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	parser.Stdout = os.Stdout
	parser.Stderr = os.Stderr

	if len(args) == 0 {
		_, _ = parseWithExit(parser, []string{"--help"})
		return 2
	}

	args = rewriteCommandAliases(args)

	ctx, err := parseWithExit(parser, args)
	if err != nil {
		var parseErr *kong.ParseError
		if errors.As(err, &parseErr) {
			_ = parseErr.Context.PrintUsage(true)
		}
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	if ctx == nil {
		return 0
	}

	if err := ctx.Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

func parseWithExit(parser *kong.Kong, args []string) (ctx *kong.Context, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(exitPanic); ok {
				ctx = nil
				err = nil
				return
			}
			panic(r)
		}
	}()

	ctx, err = parser.Parse(args)
	return ctx, err
}

func rewriteCommandAliases(args []string) []string {
	if len(args) == 0 {
		return args
	}
	out := append([]string(nil), args...)
	for i := 0; i < len(out); i++ {
		arg := out[i]
		if arg == "--" {
			return out
		}
		if arg == "--color" {
			i++
			continue
		}
		if strings.HasPrefix(arg, "--color=") {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(arg)) {
		case "contact-sheet", "contactsheet", "stills":
			out[i] = "sheet"
		}
		return out
	}
	return out
}
