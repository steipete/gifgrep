package model

import "time"

const AppName = "gifgrep"

const Tagline = "Grep the GIF. Stick the landing."

var Version = "0.2.0"

type Result struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	URL        string   `json:"url"`
	PreviewURL string   `json:"preview_url"`
	Tags       []string `json:"tags,omitempty"`
	Width      int      `json:"width,omitempty"`
	Height     int      `json:"height,omitempty"`
}

type Options struct {
	Color    string
	Verbose  int
	Quiet    bool
	Reveal   bool
	Download bool
	Format   string
	Thumbs   string

	JSON   bool
	Number bool
	Limit  int
	Source string

	GifInput      string
	StillAt       time.Duration
	StillSet      bool
	StillsCount   int
	StillsCols    int
	StillsPadding int
	OutPath       string
}
