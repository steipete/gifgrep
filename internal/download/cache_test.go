package download

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/testutil"
)

func TestDownloadCacheReuseAndExpiry(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	var requests atomic.Int32
	payload := testutil.MakeTestGIF()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()
	item := model.Result{ID: "1", Title: "cat", URL: srv.URL}
	opts := model.CacheOptions{Enabled: true, Dir: t.TempDir(), MaxAge: time.Hour, MaxBytes: 1 << 20}
	first, err := ToDownloads(item, opts)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ToDownloads(item, opts)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 || first == second || filepath.Dir(second) != filepath.Join(home, "Downloads") {
		t.Fatalf("reuse/destination: requests=%d first=%s second=%s", requests.Load(), first, second)
	}
	assertFileBytes(t, first, payload)
	assertFileBytes(t, second, payload)
	cachePath := cachedDownloadPath(item.URL, opts)
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(cachePath, old, old); err != nil {
		t.Fatal(err)
	}
	if _, err := ToDownloads(item, opts); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Fatal("expired cache was reused")
	}
	// A changed URL with the same provider ID must fetch independently.
	item.URL += "/variant"
	if _, err := ToDownloads(item, opts); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 3 {
		t.Fatal("different media URLs shared a cache entry")
	}
	srv.Close()
	if _, err := ToDownloads(item, opts); err != nil {
		t.Fatalf("warm cache still needs media server: %v", err)
	}
}

func TestDownloadCacheBypasses(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	payload := testutil.MakeTestGIF()
	for _, name := range []string{"disabled", "unavailable", "oversized", "not-gif", "corrupt", "symlink"} {
		t.Run(name, func(t *testing.T) {
			var requests atomic.Int32
			body := payload
			if name == "not-gif" {
				body = []byte("upstream returned an HTML error with status 200")
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				_, _ = w.Write(body)
			}))
			defer srv.Close()
			opts := model.CacheOptions{Enabled: true, Dir: t.TempDir(), MaxAge: time.Hour, MaxBytes: 1 << 20}
			switch name {
			case "disabled":
				opts.Enabled = false
			case "unavailable":
				opts.Dir = filepath.Join(opts.Dir, "file")
				if err := os.WriteFile(opts.Dir, []byte("untouched"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "oversized":
				opts.MaxBytes = 1
			}
			item := model.Result{Title: name, URL: srv.URL}
			if _, err := ToDownloads(item, opts); err != nil {
				t.Fatal(err)
			}
			cachePath := cachedDownloadPath(item.URL, opts)
			if name == "corrupt" {
				if err := os.WriteFile(cachePath, []byte("bad"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if name == "symlink" {
				if err := os.Remove(cachePath); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(t.TempDir(), "target.gif")
				if err := os.WriteFile(target, payload, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, cachePath); err != nil {
					t.Fatal(err)
				}
			}
			path, err := ToDownloads(item, opts)
			if err != nil {
				t.Fatal(err)
			}
			assertFileBytes(t, path, body)
			if requests.Load() != 2 {
				t.Fatalf("expected cache bypass, got %d requests", requests.Load())
			}
			if name == "disabled" {
				entries, err := os.ReadDir(opts.Dir)
				if err != nil || len(entries) != 0 {
					t.Fatalf("disabled cache touched disk: %v %v", entries, err)
				}
			}
		})
	}
}

func TestDownloadCachePruning(t *testing.T) {
	dir := t.TempDir()
	opts := model.CacheOptions{Enabled: true, Dir: dir, MaxAge: time.Hour, MaxBytes: 12}
	now := time.Now()
	paths := make([]string, 3)
	for i, url := range []string{"https://giphy.test/a", "https://klipy.test/a", "https://giphy.test/b"} {
		paths[i] = cachedDownloadPath(url, opts)
		if err := os.MkdirAll(filepath.Dir(paths[i]), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(paths[i], []byte("GIF89a"), 0o600); err != nil {
			t.Fatal(err)
		}
		written := now.Add(time.Duration(i-3) * time.Minute)
		if err := os.Chtimes(paths[i], written, written); err != nil {
			t.Fatal(err)
		}
	}
	other := filepath.Join(filepath.Dir(paths[0]), "keep.txt")
	if err := os.WriteFile(other, []byte("untouched"), 0o600); err != nil {
		t.Fatal(err)
	}
	pruneDownloadCache(filepath.Dir(paths[0]), opts)
	if _, err := os.Stat(paths[0]); !os.IsNotExist(err) {
		t.Fatal("oldest entry was not evicted")
	}
	assertFileBytes(t, paths[1], []byte("GIF89a"))
	assertFileBytes(t, paths[2], []byte("GIF89a"))
	assertFileBytes(t, other, []byte("untouched"))
	opts.MaxBytes = 0
	opts.MaxAge = time.Second
	pruneDownloadCache(filepath.Dir(paths[0]), opts)
	if _, err := os.Stat(paths[2]); !os.IsNotExist(err) {
		t.Fatal("expired entry was not evicted with unlimited size")
	}
}

func TestDownloadsConcurrentAndFailed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	payload := testutil.MakeTestGIF()
	srv := httptest.NewServer(httpHandlerString(string(payload)))
	defer srv.Close()
	item := model.Result{Title: "same", URL: srv.URL}
	opts := model.CacheOptions{Enabled: true, Dir: t.TempDir()}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := ToDownloads(item, opts); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	entries, err := os.ReadDir(filepath.Join(home, "Downloads"))
	if err != nil || len(entries) != 8 {
		t.Fatalf("concurrent saves collided: %v %v", entries, err)
	}
	for _, entry := range entries {
		assertFileBytes(t, filepath.Join(home, "Downloads", entry.Name()), payload)
	}
	srv.Close()
	item.Title = "failed"
	if _, err := ToDownloads(item, model.CacheOptions{}); err == nil {
		t.Fatal("expected failed download")
	}
	if _, err := os.Stat(filepath.Join(home, "Downloads", "failed.gif")); !os.IsNotExist(err) {
		t.Fatal("failed download left a reservation")
	}
}

func assertFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(want) {
		t.Fatalf("unexpected file contents at %s: %v", path, err)
	}
}
