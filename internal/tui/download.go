package tui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/reveal"
)

func downloadSelected(state *appState, out *bufio.Writer) {
	if state.selected < 0 || state.selected >= len(state.results) {
		state.status = "No selection"
		state.renderDirty = true
		return
	}
	item := state.results[state.selected]
	if item.URL == "" {
		state.status = "No URL"
		state.renderDirty = true
		return
	}
	state.status = "Downloading..."
	state.renderDirty = true
	render(state, out, state.lastRows, state.lastCols)
	_ = out.Flush()

	filePath, err := downloadGIFToDownloads(item)
	if err != nil {
		state.status = "Download error: " + err.Error()
		state.renderDirty = true
		return
	}
	state.lastSavedPath = filePath
	if state.opts.Reveal {
		if err := reveal.Reveal(filePath); err != nil {
			state.status = "Saved " + filePath + " (reveal failed)"
			state.renderDirty = true
			return
		}
		state.status = "Saved " + filePath + " (revealed)"
		state.renderDirty = true
		return
	}
	state.status = "Saved " + filePath
	state.renderDirty = true
}

func downloadGIFToDownloads(item model.Result) (string, error) {
	dir, err := defaultDownloadDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	filename := filenameForResult(item)
	finalPath, err := uniqueFilePath(dir, filename)
	if err != nil {
		return "", err
	}
	if err := downloadGIFToFile(item.URL, finalPath); err != nil {
		return "", err
	}
	return finalPath, nil
}

func defaultDownloadDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Downloads"), nil
}

func filenameForResult(item model.Result) string {
	name := strings.TrimSpace(item.Title)
	if name == "" {
		name = strings.TrimSpace(item.ID)
	}
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		name = filenameFromURL(item.URL)
	}
	if name == "" {
		name = "gif"
	}
	name = sanitizeFilename(name)
	if !strings.HasSuffix(strings.ToLower(name), ".gif") {
		name += ".gif"
	}
	const maxLen = 80
	if len(name) > maxLen {
		base := strings.TrimSuffix(name, filepath.Ext(name))
		ext := filepath.Ext(name)
		if len(ext) > 10 {
			ext = ".gif"
		}
		trim := maxLen - len(ext)
		if trim < 1 {
			trim = maxLen
		}
		if len(base) > trim {
			base = base[:trim]
		}
		name = base + ext
	}
	return name
}

func filenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	base := path.Base(parsed.Path)
	if base == "." || base == "/" {
		return ""
	}
	return base
}

func sanitizeFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r > unicode.MaxASCII {
			b.WriteRune('_')
			continue
		}
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('_')
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "._-")
	if out == "" {
		return "gif"
	}
	return out
}

func uniqueFilePath(dir, filename string) (string, error) {
	path := filepath.Join(dir, filename)
	_, err := os.Stat(path)
	if err == nil {
		base := strings.TrimSuffix(filename, filepath.Ext(filename))
		ext := filepath.Ext(filename)
		if ext == "" {
			ext = ".gif"
		}
		for i := 1; i < 1000; i++ {
			candidate := filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, i, ext))
			if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
				return candidate, nil
			}
		}
		return "", errors.New("could not pick filename")
	}
	if errors.Is(err, os.ErrNotExist) {
		return path, nil
	}
	return "", err
}

func downloadGIFToFile(gifURL, dest string) error {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest(http.MethodGet, gifURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "gifgrep")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}

	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, "gifgrep-*.gif")
	if err != nil {
		return err
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dest)
}
