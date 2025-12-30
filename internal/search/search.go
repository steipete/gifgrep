package search

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/steipete/gifgrep/internal/model"
)

type tenorV1Response struct {
	Results []struct {
		ID                 string               `json:"id"`
		Title              string               `json:"title"`
		ContentDescription string               `json:"content_description"`
		Tags               []string             `json:"tags"`
		Media              []map[string]mediaV1 `json:"media"`
	} `json:"results"`
}

type mediaV1 struct {
	URL  string `json:"url"`
	Dims []int  `json:"dims"`
}

func Search(query string, opts model.Options) ([]model.Result, error) {
	switch opts.Source {
	case "tenor":
		return fetchTenorV1(query, opts)
	case "giphy":
		return fetchGiphyV1(query, opts)
	default:
		return nil, fmt.Errorf("unknown source: %s", opts.Source)
	}
}

func fetchTenorV1(query string, opts model.Options) ([]model.Result, error) {
	apiKey := os.Getenv("TENOR_API_KEY")
	if apiKey == "" {
		apiKey = "LIVDSRZULELA"
	}
	if apiKey == "" {
		return nil, errors.New("missing TENOR_API_KEY")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("key", apiKey)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("contentfilter", "low")

	reqURL := "https://api.tenor.com/v1/search?" + params.Encode()
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
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}

	var parsed tenorV1Response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	out := make([]model.Result, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		title := r.Title
		if title == "" {
			title = r.ContentDescription
		}
		if title == "" {
			title = r.ID
		}
		gifURL := ""
		preview := ""
		width := 0
		height := 0
		if len(r.Media) > 0 {
			media := r.Media[0]
			if m, ok := media["gif"]; ok {
				gifURL = m.URL
				if len(m.Dims) == 2 {
					width, height = m.Dims[0], m.Dims[1]
				}
			}
			if m, ok := media["tinygif"]; ok {
				preview = m.URL
				if gifURL == "" {
					gifURL = m.URL
					if len(m.Dims) == 2 {
						width, height = m.Dims[0], m.Dims[1]
					}
				}
			}
			if preview == "" {
				preview = gifURL
			}
		}
		if gifURL == "" {
			continue
		}
		out = append(out, model.Result{
			ID:         r.ID,
			Title:      title,
			URL:        gifURL,
			PreviewURL: preview,
			Tags:       r.Tags,
			Width:      width,
			Height:     height,
		})
	}
	return out, nil
}
