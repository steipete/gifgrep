package download

import (
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
)

func ToDownloads(item model.Result, cache model.CacheOptions) (string, error) {
	dir, err := DefaultDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	filename := filenameForResult(item)
	finalPath, err := reserveFilePath(dir, filename)
	if err != nil {
		return "", err
	}
	saved := false
	defer func() {
		if !saved {
			_ = os.Remove(finalPath)
		}
	}()
	cachePath := cachedDownloadPath(item.URL, cache)
	if cachePath != "" {
		pruneDownloadCache(filepath.Dir(cachePath), cache)
	}
	if cachePath != "" && copyCachedDownload(cachePath, finalPath, cache) == nil {
		saved = true
		return finalPath, nil
	}
	client := &http.Client{Timeout: 20 * time.Second}
	if err := downloadGIFToFile(client, item.URL, finalPath); err != nil {
		return "", err
	}
	saved = true
	if cachePath != "" {
		storeCachedDownload(finalPath, cachePath, cache)
	}
	return finalPath, nil
}

func DefaultDir() (string, error) {
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

func reserveFilePath(dir, filename string) (string, error) {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := filepath.Ext(filename)
	for i := 0; i < 1000; i++ {
		name := filename
		if i > 0 {
			name = fmt.Sprintf("%s-%d%s", base, i, ext)
		}
		candidate := filepath.Join(dir, name)
		// Reserve before fetching/copying so simultaneous saves cannot overwrite
		// each other. The caller removes the reservation on failure.
		file, err := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if err := file.Close(); err != nil {
			_ = os.Remove(candidate)
			return "", err
		}
		return candidate, nil
	}
	return "", errors.New("could not pick filename")
}

func downloadGIFToFile(client *http.Client, gifURL, dest string) error {
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequest(http.MethodGet, gifURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "gifgrep")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
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
