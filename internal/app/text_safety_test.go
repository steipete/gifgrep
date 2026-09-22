package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/termcaps"
	"github.com/steipete/gifgrep/internal/testutil"
)

func TestSearchTextDoesNotEmitProviderControls(t *testing.T) {
	payload := "\x1b]52;c;c3ludGhldGlj\a\x1b[2J\u009b31m"
	result := model.Result{Title: "Café " + payload, URL: "https://example.test/gif" + payload + "\r\nforged"}
	for _, format := range []outputFormat{formatPlain, formatTSV, formatURL, formatMD, formatComment} {
		t.Run(string(format), func(t *testing.T) {
			var buf bytes.Buffer
			out := bufio.NewWriter(&buf)
			writeSearchResults(out, model.Options{}, false, termcaps.InlineNone, []model.Result{result}, 80, format)
			if err := out.Flush(); err != nil {
				t.Fatal(err)
			}
			got := buf.String()
			if strings.ContainsAny(got, "\x1b\a\r\u009b") || strings.Contains(got, "\nforged") {
				t.Fatalf("unsafe provider controls in %q", got)
			}
			if format != formatURL && !strings.Contains(got, "Café") {
				t.Fatalf("lost Unicode title: %q", got)
			}
		})
	}
}

type textProviderTransport struct{}

func (textProviderTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"fixture","title":"Café \u001b[2J","images":{"original":{"url":"https://example.test/gif\nline"}}}]}`))}, nil
}

func TestSearchJSONPreservesProviderText(t *testing.T) {
	t.Setenv("GIPHY_API_KEY", "synthetic-test-key")
	testutil.WithTransport(t, textProviderTransport{}, func() {
		var out bytes.Buffer
		if err := runSearch(&out, io.Discard, model.Options{Source: "giphy", JSON: true}, "fixture"); err != nil {
			t.Fatal(err)
		}
		var results []model.Result
		if err := json.Unmarshal(out.Bytes(), &results); err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 || results[0].Title != "Café \x1b[2J" || results[0].URL != "https://example.test/gif\nline" {
			t.Fatalf("JSON changed provider fields: %+v", results)
		}
	})
}

func TestRunErrorsDoNotEmitControls(t *testing.T) {
	payload := "Café\x1b]52;c;c3ludGhldGlj\a\u009b31m\nforged"
	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{name: "missing input", args: []string{"still", filepath.Join(t.TempDir(), payload), "--at", "0"}, code: 1},
		{name: "invalid flag", args: []string{"search", "--" + payload}, code: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stderr, err := os.CreateTemp(t.TempDir(), "stderr")
			if err != nil {
				t.Fatal(err)
			}
			original := os.Stderr
			os.Stderr = stderr
			t.Cleanup(func() {
				os.Stderr = original
				_ = stderr.Close()
			})
			if code := Run(tc.args); code != tc.code {
				t.Fatalf("exit code = %d, want %d", code, tc.code)
			}
			output, err := os.ReadFile(stderr.Name())
			if err != nil {
				t.Fatal(err)
			}
			got := string(output)
			if strings.ContainsAny(got, "\x1b\a\u009b") || strings.Contains(got, "\nforged") {
				t.Fatalf("unsafe diagnostic controls in %q", got)
			}
			if !strings.Contains(got, "Café") || !strings.Contains(got, `\x1b`) {
				t.Fatalf("diagnostic lost escaped input: %q", got)
			}
		})
	}
}
