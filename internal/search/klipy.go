package search

import (
	"errors"
	"net/url"
	"os"
	"strconv"

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
	params.Set("limit", strconv.Itoa(limit))
	params.Set("contentfilter", "low")
	params.Set("media_filter", "gif,tinygif,mediumgif,nanogif,preview")

	var parsed klipyV2Response
	if err := fetchSearchJSON("https://api.klipy.com/v2/search", params, &parsed); err != nil {
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
