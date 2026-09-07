package download

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
)

func httpHandlerString(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	})
}

func TestFilenameForResult(t *testing.T) {
	t.Parallel()

	if got := filenameForResult(model.Result{Title: "Cat Christmas GIF"}); got != "Cat_Christmas_GIF.gif" {
		t.Fatalf("unexpected filename: %q", got)
	}
	if got := filenameForResult(model.Result{ID: "abc"}); got != "abc.gif" {
		t.Fatalf("unexpected filename: %q", got)
	}
	if got := filenameForResult(model.Result{URL: "https://example.com/foo/bar.gif?x=1"}); got != "bar.gif" {
		t.Fatalf("unexpected filename: %q", got)
	}

	long := strings.Repeat("a", 200)
	got := filenameForResult(model.Result{Title: long})
	if len(got) > 80 {
		t.Fatalf("expected <=80 chars, got %d (%q)", len(got), got)
	}
	if !strings.HasSuffix(strings.ToLower(got), ".gif") {
		t.Fatalf("expected .gif suffix, got %q", got)
	}
}

func TestReserveFilePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	p1, err := reserveFilePath(dir, "x.gif")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p1) != "x.gif" {
		t.Fatalf("unexpected path: %q", p1)
	}
	if err := os.WriteFile(p1, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	p2, err := reserveFilePath(dir, "x.gif")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p2) != "x-1.gif" {
		t.Fatalf("unexpected path: %q", p2)
	}
}

func TestDownloadGIFToFile(t *testing.T) {
	t.Parallel()

	const payload = "GIF89a"
	srv := httptest.NewServer(httpHandlerString(payload))
	t.Cleanup(srv.Close)

	dest := filepath.Join(t.TempDir(), "out.gif")
	if err := downloadGIFToFile(srv.Client(), srv.URL, dest); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != payload {
		t.Fatalf("unexpected payload: %q", string(b))
	}
}

func TestToDownloadsUsesHome(t *testing.T) {
	const payload = "GIF89a"
	srv := httptest.NewServer(httpHandlerString(payload))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("HOME", home)

	res := model.Result{
		Title: "a",
		URL:   srv.URL,
	}
	got, err := ToDownloads(res, model.CacheOptions{})
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(home, "Downloads")
	if filepath.Dir(got) != wantDir {
		t.Fatalf("expected %q, got %q", wantDir, filepath.Dir(got))
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatal(err)
	}
}
