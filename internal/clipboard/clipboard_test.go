package clipboard

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestCopyCommandLinuxSession(t *testing.T) {
	for _, tt := range []struct {
		name        string
		display     string
		sessionType string
		xclip       bool
		wlCopy      bool
		want        string
	}{
		{"wayland with both tools", "wayland-0", "wayland", true, true, "wl-copy"},
		{"wayland display only", "wayland-1", "", true, true, "wl-copy"},
		{"wayland session only", "", "wayland", true, true, "wl-copy"},
		{"X11 with both tools", "", "x11", true, true, "xclip"},
		{"no session hints", "", "", true, true, "xclip"},
		{"wayland without wl-copy", "wayland-0", "wayland", true, false, "xclip"},
		{"only wl-copy", "", "", false, true, "wl-copy"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WAYLAND_DISPLAY", tt.display)
			t.Setenv("XDG_SESSION_TYPE", tt.sessionType)
			prev := lookPath
			t.Cleanup(func() { lookPath = prev })
			lookPath = func(name string) (string, error) {
				if (name == "xclip" && tt.xclip) || (name == "wl-copy" && tt.wlCopy) {
					return "/usr/bin/" + name, nil
				}
				return "", errors.New("not found")
			}
			spec, err := copyCommand("linux", "/tmp/a.gif")
			if err != nil {
				t.Fatal(err)
			}
			if spec.name != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, spec.name)
			}
		})
	}
}

func TestCopyFileDoesNotWaitForClipboardOwner(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires a Unix shell")
	}
	dir := t.TempDir()
	tool := "xclip"
	if runtime.GOOS == "darwin" {
		tool = "osascript"
	}
	pidPath := filepath.Join(dir, "owner.pid")
	t.Setenv("GIFGREP_TEST_OWNER_PID", pidPath)
	t.Setenv("PATH", dir)
	// The launcher exits, but the clipboard owner keeps its output descriptors.
	script := "#!/bin/sh\n/bin/sleep 30 &\necho $! > \"$GIFGREP_TEST_OWNER_PID\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, tool), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	t.Cleanup(func() {
		data, err := os.ReadFile(pidPath)
		if err != nil {
			t.Error(err)
			return
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			t.Error(err)
			return
		}
		proc, err := os.FindProcess(pid)
		if err == nil {
			_ = proc.Kill()
			_ = proc.Release()
		}
	})
	go func() { done <- CopyFile(filepath.Join(dir, "a.gif")) }()
	deadline := time.Now().Add(10 * time.Second)
	for {
		data, _ := os.ReadFile(pidPath)
		if strings.TrimSpace(string(data)) != "" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("clipboard launcher did not start")
		}
		time.Sleep(10 * time.Millisecond)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("copy waited for the background clipboard owner")
	}
}
