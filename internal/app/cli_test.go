package app

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/steipete/gifgrep/internal/model"
	"github.com/steipete/gifgrep/internal/testutil"
)

func TestRunSearchOutput(t *testing.T) {
	t.Setenv("KLIPY_API_KEY", "test-key")
	gifData := testutil.MakeTestGIF()
	testutil.WithTransport(t, &testutil.FakeTransport{GIFData: gifData}, func() {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		err := runSearch(&stdout, &stderr, model.Options{Number: true, Limit: 1, Source: "tenor"}, "cats")
		if err != nil {
			t.Fatalf("runSearch failed: %v", err)
		}
		if !strings.Contains(stdout.String(), "1\t") {
			t.Fatalf("expected numbered output")
		}
	})
}

func TestRunSearchJSON(t *testing.T) {
	t.Setenv("KLIPY_API_KEY", "test-key")
	gifData := testutil.MakeTestGIF()
	testutil.WithTransport(t, &testutil.FakeTransport{GIFData: gifData}, func() {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		err := runSearch(&stdout, &stderr, model.Options{JSON: true, Limit: 1, Source: "tenor"}, "cats")
		if err != nil {
			t.Fatalf("runSearch json failed: %v", err)
		}
		if !bytes.Contains(stdout.Bytes(), []byte(`"preview_url"`)) {
			t.Fatalf("expected json output")
		}
	})
}

func TestHelpOutput(t *testing.T) {
	oldStdout := os.Stdout
	// A pipe can fill before Run returns and the test starts reading.
	w, err := os.CreateTemp(t.TempDir(), "help")
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
		_ = w.Close()
	})

	code := Run([]string{"--help"})
	_ = w.Close()
	if code != 0 {
		t.Fatalf("expected exit 0")
	}
	out, err := os.ReadFile(w.Name())
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	if !strings.Contains(text, "Examples:") {
		t.Fatalf("expected Examples section")
	}
	if !strings.Contains(text, "--no-color") {
		t.Fatalf("expected --no-color in help")
	}
}
