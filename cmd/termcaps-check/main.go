package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/steipete/gifgrep/internal/termcaps"
)

type report struct {
	Detected     string `json:"detected"`
	TermProgram  string `json:"term_program,omitempty"`
	Term         string `json:"term,omitempty"`
	ItermSession string `json:"iterm_session_id,omitempty"`
	KittyWindow  string `json:"kitty_window_id,omitempty"`
}

func main() {
	var expect string
	var asJSON bool
	flag.StringVar(&expect, "expect", "", "Expected protocol: none|kitty|iterm (optional)")
	flag.BoolVar(&asJSON, "json", true, "Emit JSON")
	flag.Parse()

	detected := termcaps.DetectInlineRobust(os.Getenv)

	r := report{
		Detected:     detected.String(),
		TermProgram:  os.Getenv("TERM_PROGRAM"),
		Term:         os.Getenv("TERM"),
		ItermSession: os.Getenv("ITERM_SESSION_ID"),
		KittyWindow:  os.Getenv("KITTY_WINDOW_ID"),
	}

	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(r); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "encode: %v\n", err)
			os.Exit(1)
		}
	} else {
		if _, err := fmt.Fprintln(os.Stdout, r.Detected); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}
	}

	if expect != "" && expect != r.Detected {
		fmt.Fprintf(os.Stderr, "expected %q, got %q\n", expect, r.Detected)
		os.Exit(1)
	}
}
