package download

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/steipete/gifgrep/internal/model"
)

func cachedDownloadPath(rawURL string, opts model.CacheOptions) string {
	if !opts.Enabled || opts.MaxAge < 0 || opts.MaxBytes < 0 {
		return ""
	}
	dir := opts.Dir
	if dir == "" {
		root, err := os.UserCacheDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(root, "gifgrep")
	}
	// Hash the complete URL, including provider host and media variant. Never
	// persist queries, credentials, or result titles in cache filenames.
	key := sha256.Sum256([]byte(rawURL))
	return filepath.Join(dir, "downloads-v1", fmt.Sprintf("%x.gif", key))
}

func copyCachedDownload(source, dest string, opts model.CacheOptions) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || cacheExpired(info, opts, time.Now()) ||
		(opts.MaxBytes > 0 && info.Size() > opts.MaxBytes) {
		return errors.New("cache miss")
	}
	if !hasGIFHeader(source) {
		return errors.New("cache miss")
	}
	return copyFileAtomic(source, dest)
}

func storeCachedDownload(source, dest string, opts model.CacheOptions) {
	info, err := os.Stat(source)
	if err != nil || (opts.MaxBytes > 0 && info.Size() > opts.MaxBytes) || !hasGIFHeader(source) {
		return
	}
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	if err := copyFileAtomic(source, dest); err != nil {
		return
	}
	pruneDownloadCache(dir, opts)
}

func hasGIFHeader(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()
	var header [6]byte
	if _, err := io.ReadFull(file, header[:]); err != nil {
		return false
	}
	return string(header[:]) == "GIF87a" || string(header[:]) == "GIF89a"
}

func cacheExpired(info os.FileInfo, opts model.CacheOptions, now time.Time) bool {
	return opts.MaxAge > 0 && now.Sub(info.ModTime()) >= opts.MaxAge
}

func pruneDownloadCache(dir string, opts model.CacheOptions) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var files []os.FileInfo
	var total int64
	now := time.Now()
	for _, entry := range entries {
		name := entry.Name()
		if len(name) != 64+len(".gif") || !strings.HasSuffix(name, ".gif") {
			continue
		}
		if _, err := hex.DecodeString(strings.TrimSuffix(name, ".gif")); err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if cacheExpired(info, opts, now) && os.Remove(filepath.Join(dir, name)) == nil {
			continue
		}
		files = append(files, info)
		total += info.Size()
	}
	if opts.MaxBytes == 0 {
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].ModTime().Before(files[j].ModTime()) })
	for _, info := range files {
		if total <= opts.MaxBytes {
			break
		}
		if os.Remove(filepath.Join(dir, info.Name())) == nil {
			total -= info.Size()
		}
	}
}

func copyFileAtomic(source, dest string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.CreateTemp(filepath.Dir(dest), "gifgrep-copy-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
		_ = os.Remove(out.Name())
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(out.Name(), dest)
}
