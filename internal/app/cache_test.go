package app

import (
	"testing"
	"time"

	"github.com/alecthomas/kong"
)

func TestCacheFlags(t *testing.T) {
	t.Setenv("GIFGREP_CACHE", "1")
	t.Setenv("GIFGREP_CACHE_DIR", "/environment/cache")
	for _, command := range []string{"search", "tui"} {
		t.Run(command, func(t *testing.T) {
			var cli CLI
			parser, err := kong.New(&cli)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := parser.Parse([]string{command, "cats"}); err != nil {
				t.Fatal(err)
			}
			flags := cli.Search.CacheFlags
			if command == "tui" {
				flags = cli.TUI.CacheFlags
			}
			if !flags.Cache || flags.CacheDir != "/environment/cache" || flags.CacheMaxAge != 7*24*time.Hour || flags.CacheMaxBytes != 100*1024*1024 {
				t.Fatalf("unexpected defaults/environment: %+v", flags)
			}
			if _, err := parser.Parse([]string{command, "cats", "--cache=false", "--cache-dir=/flag/cache", "--cache-max-age=0", "--cache-max-bytes=0"}); err != nil {
				t.Fatal(err)
			}
			flags = cli.Search.CacheFlags
			if command == "tui" {
				flags = cli.TUI.CacheFlags
			}
			if flags.Cache || flags.CacheDir != "/flag/cache" || flags.CacheMaxAge != 0 || flags.CacheMaxBytes != 0 {
				t.Fatalf("flags did not override environment/defaults: %+v", flags)
			}
		})
	}
}

func TestCacheRejectsNegativeLimits(t *testing.T) {
	for _, flags := range []CacheFlags{{CacheMaxAge: -time.Second}, {CacheMaxBytes: -1}} {
		if _, err := flags.options(); err == nil {
			t.Fatal("negative cache limit accepted")
		}
	}
}
