package app

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/steipete/gifgrep/gifdecode"
	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/stills"
)

func runExtract(opts model.Options) error {
	if opts.GifInput == "" {
		return errors.New("missing GIF input")
	}
	if opts.StillSet {
		if opts.StillsCount > 0 {
			return errors.New("use still or sheet, not both")
		}
	} else {
		if opts.StillsCount < 1 {
			return errors.New("bad args: --frames must be >= 1")
		}
		if opts.StillsCols < 0 {
			return errors.New("bad args: --cols must be >= 0")
		}
		if opts.StillsPadding < 0 {
			return errors.New("bad args: --padding must be >= 0")
		}
	}

	data, err := readInput(opts.GifInput)
	if err != nil {
		return err
	}
	decodeOpts := gifdecode.DefaultOptions()
	decodeOpts.MaxFrames = 0
	decoded, err := gifdecode.Decode(data, decodeOpts)
	if err != nil {
		return err
	}

	var output []byte
	if opts.StillSet {
		pngData, _, err := stills.FrameAtPNG(decoded, opts.StillAt)
		if err != nil {
			return err
		}
		output = pngData
	} else {
		pngData, err := stills.ContactSheet(decoded, stills.SheetOptions{
			Count:   opts.StillsCount,
			Columns: opts.StillsCols,
			Padding: opts.StillsPadding,
		})
		if err != nil {
			return err
		}
		output = pngData
	}

	outPath := resolveExtractOutPath(opts)
	return writeOutput(outPath, output)
}

func resolveExtractOutPath(opts model.Options) string {
	outPath := opts.OutPath
	if outPath == "" {
		if opts.StillSet {
			outPath = "still.png"
		} else {
			outPath = "sheet.png"
		}
	}
	return outPath
}

func readInput(input string) ([]byte, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return fetchURL(input)
	}
	if strings.HasPrefix(input, "file://") {
		parsed, err := url.Parse(input)
		if err != nil {
			return nil, err
		}
		return os.ReadFile(parsed.Path)
	}
	return os.ReadFile(input)
}

func fetchURL(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gifgrep")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func writeOutput(path string, data []byte) error {
	if path == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
