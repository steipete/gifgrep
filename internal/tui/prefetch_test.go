package tui

import (
	"bytes"
	"context"
	"math"
	"os"
	"testing"

	"github.com/steipete/gifgrep/internal/testutil"
)

func TestPrefetchGIFToTempRespectsSize(t *testing.T) {
	data := testutil.MakeTestGIF()
	rt := &testutil.FakeTransport{GIFData: data}
	testutil.WithTransport(t, rt, func() {
		dir := t.TempDir()
		path, err := prefetchGIFToTemp(context.Background(), "https://example.test/full.gif", dir, int64(len(data)+1))
		if err != nil {
			t.Fatalf("prefetch failed: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing temp file: %v", err)
		}

		if _, err := prefetchGIFToTemp(context.Background(), "https://example.test/full.gif", dir, int64(len(data)-1)); err == nil {
			t.Fatalf("expected size cap error")
		}
		path, err = prefetchGIFToTemp(context.Background(), "https://example.test/full.gif", dir, math.MaxInt64)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("maximum byte limit lost download data: %v", err)
		}
	})
}
