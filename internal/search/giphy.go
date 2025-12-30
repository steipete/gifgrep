package search

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/steipete/gifgrep/internal/model"
)

type giphySearchResponse struct {
	Data []struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Images struct {
			Original struct {
				URL    string `json:"url"`
				Width  string `json:"width"`
				Height string `json:"height"`
			} `json:"original"`
			FixedWidthSmall struct {
				URL    string `json:"url"`
				Width  string `json:"width"`
				Height string `json:"height"`
			} `json:"fixed_width_small"`
			PreviewGIF struct {
				URL    string `json:"url"`
				Width  string `json:"width"`
				Height string `json:"height"`
			} `json:"preview_gif"`
		} `json:"images"`
	} `json:"data"`
}

func fetchGiphyV1(query string, opts model.Options) ([]model.Result, error) {
	apiKey := os.Getenv("GIPHY_API_KEY")
	if apiKey == "" {
		return nil, errors.New("missing GIPHY_API_KEY")
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("api_key", apiKey)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("rating", "g")

	reqURL := "https://api.giphy.com/v1/gifs/search?" + params.Encode()
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gifgrep")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}

	var parsed giphySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	out := make([]model.Result, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		gifURL := item.Images.Original.URL
		preview := item.Images.FixedWidthSmall.URL
		if preview == "" {
			preview = item.Images.PreviewGIF.URL
		}
		if preview == "" {
			preview = gifURL
		}
		if gifURL == "" {
			continue
		}

		width := parseMaybeInt(item.Images.Original.Width)
		height := parseMaybeInt(item.Images.Original.Height)

		title := item.Title
		if title == "" {
			title = item.ID
		}

		out = append(out, model.Result{
			ID:         item.ID,
			Title:      title,
			URL:        gifURL,
			PreviewURL: preview,
			Width:      width,
			Height:     height,
		})
	}

	return out, nil
}

func parseMaybeInt(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	if v < 0 {
		return 0
	}
	return v
}

