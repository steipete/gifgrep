package search

import (
	"fmt"
	"os"
	"strings"

	"github.com/steipete/gifgrep/internal/model"
)

func Search(query string, opts model.Options) ([]model.Result, error) {
	source := ResolveSource(opts.Source)
	if source == "giphy" && isAutoSource(opts.Source) && os.Getenv("KLIPY_API_KEY") != "" {
		results, err := fetchGiphyV1(query, opts)
		if err == nil {
			return results, nil
		}
		return fetchKlipyV2(query, opts)
	}

	switch source {
	case "klipy":
		return fetchKlipyV2(query, opts)
	case "giphy":
		return fetchGiphyV1(query, opts)
	default:
		return nil, fmt.Errorf("unknown source: %s", opts.Source)
	}
}

func isAutoSource(source string) bool {
	source = strings.ToLower(strings.TrimSpace(source))
	return source == "" || source == "auto"
}
