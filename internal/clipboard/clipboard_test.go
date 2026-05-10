package clipboard

import (
	"errors"
	"testing"
)

func TestCopyFileEmptyPath(t *testing.T) {
	if err := CopyFile(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestCopyCommandDarwin(t *testing.T) {
	spec, err := copyCommand("darwin", `/tmp/a "quote".gif`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if spec.name != "osascript" {
		t.Fatalf("expected osascript, got %q", spec.name)
	}
	if len(spec.args) != 3 || spec.args[0] != "-e" || spec.args[2] != `/tmp/a "quote".gif` {
		t.Fatalf("unexpected args: %#v", spec.args)
	}
	if spec.stdinPath != "" {
		t.Fatalf("unexpected stdin path: %q", spec.stdinPath)
	}
}

func TestCopyCommandLinuxXclip(t *testing.T) {
	prev := lookPath
	lookPath = func(name string) (string, error) {
		if name == "xclip" {
			return "/usr/bin/xclip", nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() { lookPath = prev })

	spec, err := copyCommand("linux", "/tmp/a.gif")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if spec.name != "xclip" {
		t.Fatalf("expected xclip, got %q", spec.name)
	}
	if len(spec.args) != 6 || spec.args[4] != "-i" {
		t.Fatalf("unexpected args: %#v", spec.args)
	}
	if spec.stdinPath != "" {
		t.Fatalf("unexpected stdin path: %q", spec.stdinPath)
	}
}

func TestCopyCommandLinuxWlCopy(t *testing.T) {
	prev := lookPath
	lookPath = func(name string) (string, error) {
		if name == "wl-copy" {
			return "/usr/bin/wl-copy", nil
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() { lookPath = prev })

	spec, err := copyCommand("linux", "/tmp/a.gif")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if spec.name != "wl-copy" {
		t.Fatalf("expected wl-copy, got %q", spec.name)
	}
	if len(spec.args) != 2 || spec.args[0] != "--type" || spec.args[1] != "image/gif" {
		t.Fatalf("unexpected args: %#v", spec.args)
	}
	if spec.stdinPath != "/tmp/a.gif" {
		t.Fatalf("expected stdin path, got %q", spec.stdinPath)
	}
}

func TestCopyCommandLinuxNoTool(t *testing.T) {
	prev := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("not found") }
	t.Cleanup(func() { lookPath = prev })

	_, err := copyCommand("linux", "/tmp/a.gif")
	if err == nil {
		t.Fatal("expected error when no tool available")
	}
}
