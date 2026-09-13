package search

import (
	"fmt"
	"os"
	"strings"

	"github.com/steipete/gifgrep/internal/model"
)

func Search(query string, opts model.Options) ([]model.Result, string, error) {
	source := ResolveSource(opts.Source)
	var results []model.Result
	var err error
	switch source {
	case "klipy":
		results, err = fetchKlipyV2(query, opts)
	case "giphy":
		results, err = fetchGiphyV1(query, opts)
	default:
		return nil, source, fmt.Errorf("unknown source: %s", opts.Source)
	}
	if err != nil && source == "giphy" && isAutoSource(opts.Source) && os.Getenv("KLIPY_API_KEY") != "" {
		source = "klipy"
		results, err = fetchKlipyV2(query, opts)
	}
	return results, source, err
}

func isAutoSource(source string) bool {
	source = strings.ToLower(strings.TrimSpace(source))
	return source == "" || source == "auto"
}
