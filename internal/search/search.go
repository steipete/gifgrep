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

type klipyV2Response struct {
	Results []klipyV2Result `json:"results"`
}

type klipyV2Result struct {
	ID                 string             `json:"id"`
	Title              string             `json:"title"`
	ContentDescription string             `json:"content_description"`
	Tags               []string           `json:"tags"`
	MediaFormats       map[string]mediaV2 `json:"media_formats"`
}

type mediaV2 struct {
	URL  string `json:"url"`
	Dims []int  `json:"dims"`
}

func Search(query string, opts model.Options) ([]model.Result, error) {
	switch ResolveSource(opts.Source) {
	case "klipy":
		return fetchKlipyV2(query, opts)
	case "giphy":
		return fetchGiphyV1(query, opts)
	default:
		return nil, fmt.Errorf("unknown source: %s", opts.Source)
	}
}

func fetchKlipyV2(query string, opts model.Options) ([]model.Result, error) {
	apiKey := os.Getenv("KLIPY_API_KEY")
	if apiKey == "" {
		return nil, errors.New("missing KLIPY_API_KEY")
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
	params.Set("media_filter", "gif,tinygif,mediumgif,nanogif,preview")

	reqURL := "https://api.klipy.com/v2/search?" + params.Encode()
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
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

	var parsed klipyV2Response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	out := make([]model.Result, 0, len(parsed.Results))
	for i := range parsed.Results {
		r := &parsed.Results[i]
		title := r.Title
		if title == "" {
			title = r.ContentDescription
		}
		if title == "" {
			title = r.ID
		}

		gifMedia, ok := mediaFormat(r.MediaFormats, "gif", "mediumgif", "tinygif", "nanogif", "preview")
		if !ok || gifMedia.URL == "" {
			continue
		}
		previewMedia, ok := mediaFormat(r.MediaFormats, "tinygif", "nanogif", "preview", "mediumgif", "gif")
		if !ok || previewMedia.URL == "" {
			previewMedia = gifMedia
		}
		width, height := mediaDims(gifMedia)

		out = append(out, model.Result{
			ID:         r.ID,
			Title:      title,
			URL:        gifMedia.URL,
			PreviewURL: previewMedia.URL,
			Tags:       r.Tags,
			Width:      width,
			Height:     height,
		})
	}
	return out, nil
}

func mediaFormat(formats map[string]mediaV2, names ...string) (mediaV2, bool) {
	for _, name := range names {
		media, ok := formats[name]
		if ok && media.URL != "" {
			return media, true
		}
	}
	return mediaV2{}, false
}

func mediaDims(media mediaV2) (int, int) {
	if len(media.Dims) != 2 {
		return 0, 0
	}
	return media.Dims[0], media.Dims[1]
}
