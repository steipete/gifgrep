package tui

import (
	"fmt"
	"net/http"
	"time"

	"github.com/steipete/gifgrep/internal/imageinput"
)

func fetchGIF(gifURL string) ([]byte, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, gifURL, nil)
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
	return imageinput.Read(resp.Body)
}
