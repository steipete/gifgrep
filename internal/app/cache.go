package app

import (
	"errors"
	"time"

	"github.com/steipete/gifgrep/internal/model"
)

type CacheFlags struct {
	Cache         bool          `help:"Reuse explicit downloads from a persistent cache." env:"GIFGREP_CACHE"`
	CacheDir      string        `help:"Cache directory (default: OS user cache/gifgrep). Requires --cache." env:"GIFGREP_CACHE_DIR"`
	CacheMaxAge   time.Duration `help:"Maximum cached download age; 0 disables expiry." default:"168h"`
	CacheMaxBytes int64         `help:"Cache size in bytes; 0 disables the size limit." default:"104857600"`
}

func (c CacheFlags) options() (model.CacheOptions, error) {
	if c.CacheMaxAge < 0 || c.CacheMaxBytes < 0 {
		return model.CacheOptions{}, errors.New("cache age and size limits must not be negative")
	}
	return model.CacheOptions{
		Enabled: c.Cache, Dir: c.CacheDir, MaxAge: c.CacheMaxAge, MaxBytes: c.CacheMaxBytes,
	}, nil
}
