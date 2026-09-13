package tui

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/steipete/gifgrep/internal/model"
)

type prefetchResult struct {
	key  string
	gen  int
	path string
	err  error
}

func prefetchMaxBytes() int64 {
	const fallback = int64(10 * 1024 * 1024)
	raw := strings.TrimSpace(os.Getenv("GIFGREP_TUI_PREFETCH_MAX_BYTES"))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		return fallback
	}
	return v
}

func ensureTempDir(state *appState) (string, error) {
	if state.tempDir != "" {
		return state.tempDir, nil
	}
	dir, err := os.MkdirTemp("", "gifgrep-tui-")
	if err != nil {
		return "", err
	}
	state.tempDir = dir
	return dir, nil
}

func startPrefetch(state *appState, results []model.Result, notify chan<- prefetchResult) {
	if notify == nil {
		return
	}
	if state.prefetching == nil {
		state.prefetching = map[string]bool{}
	}
	if state.tempPaths == nil {
		state.tempPaths = map[string]string{}
	}
	maxBytes := prefetchMaxBytes()
	if maxBytes == 0 {
		return
	}
	dir, err := ensureTempDir(state)
	if err != nil {
		return
	}
	gen := state.prefetchGen
	if state.prefetchCancel == nil {
		state.prefetchCtx, state.prefetchCancel = context.WithCancel(context.Background())
	}
	ctx := state.prefetchCtx
	for _, item := range results {
		if item.URL == "" {
			continue
		}
		key := resultKey(item)
		if key == "" || key == "unknown" || state.prefetching[key] {
			continue
		}
		if _, ok := savedPathForResult(state, item); ok {
			continue
		}
		if _, ok := tempPathForResult(state, item); ok {
			continue
		}
		state.prefetching[key] = true
		url := item.URL
		state.prefetchWG.Go(func() {
			path, err := prefetchGIFToTemp(ctx, url, dir, maxBytes)
			select {
			case notify <- prefetchResult{key: key, gen: gen, path: path, err: err}:
			case <-ctx.Done():
				if path != "" {
					_ = os.Remove(path)
				}
			}
		})
	}
}

func resetPrefetch(state *appState) {
	if state == nil {
		return
	}
	state.prefetchGen++
	cleanupTempDir(state)
	state.prefetching = map[string]bool{}
	state.tempPaths = map[string]string{}
}

func prefetchGIFToTemp(ctx context.Context, gifURL, dir string, maxBytes int64) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("missing temp dir")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gifURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "gifgrep")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}
	if maxBytes > 0 && resp.ContentLength > 0 && resp.ContentLength > maxBytes {
		return "", fmt.Errorf("too large")
	}

	tmp, err := os.CreateTemp(dir, "gifgrep-*.gif")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(path)
	}

	limit := maxBytes
	if limit <= 0 {
		limit = 1<<63 - 1
	}
	var body io.Reader = resp.Body
	if limit < math.MaxInt64 {
		body = io.LimitReader(body, limit+1)
	}
	n, err := io.Copy(tmp, body)
	if err != nil {
		cleanup()
		return "", err
	}
	if n > limit {
		cleanup()
		return "", fmt.Errorf("too large")
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", err
	}
	return path, nil
}

func cleanupTempDir(state *appState) {
	if state == nil {
		return
	}
	if state.prefetchCancel != nil {
		state.prefetchCancel()
		state.prefetchWG.Wait()
		state.prefetchCancel = nil
		state.prefetchCtx = nil
	}
	if state.tempDir != "" {
		_ = os.RemoveAll(state.tempDir)
	}
	state.tempDir = ""
	state.tempPaths = nil
	state.prefetching = nil
}

func tempPathForResult(state *appState, item model.Result) (string, bool) {
	if state == nil || state.tempPaths == nil {
		return "", false
	}
	key := resultKey(item)
	if key == "" {
		return "", false
	}
	if p, ok := state.tempPaths[key]; ok && p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

func acceptPrefetchResult(state *appState, res prefetchResult) bool {
	if state == nil {
		return false
	}
	if res.gen != state.prefetchGen {
		if res.path != "" {
			_ = os.Remove(res.path)
		}
		return false
	}
	delete(state.prefetching, res.key)
	if res.err != nil {
		return false
	}
	if state.tempPaths == nil {
		state.tempPaths = map[string]string{}
	}
	if res.path == "" {
		return false
	}
	state.tempPaths[res.key] = res.path
	return true
}
