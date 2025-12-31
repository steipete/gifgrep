package reveal

import (
	"errors"
	"testing"
)

func TestCommandForRevealDarwin(t *testing.T) {
	cmd, args, err := commandForReveal("darwin", "/tmp/a.gif")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cmd != "open" {
		t.Fatalf("unexpected cmd: %q", cmd)
	}
	if len(args) != 2 || args[0] != "-R" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestCommandForRevealWindows(t *testing.T) {
	cmd, args, err := commandForReveal("windows", `C:\tmp\a.gif`)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cmd != "explorer.exe" {
		t.Fatalf("unexpected cmd: %q", cmd)
	}
	if len(args) != 1 || args[0] == "" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestCommandForRevealLinuxPrefersXDGOpen(t *testing.T) {
	prev := lookPath
	lookPath = func(name string) (string, error) {
		if name == "xdg-open" {
			return "/usr/bin/xdg-open", nil
		}
		return "", errors.New("nope")
	}
	t.Cleanup(func() { lookPath = prev })

	cmd, args, err := commandForReveal("linux", "/tmp/a.gif")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cmd != "xdg-open" {
		t.Fatalf("unexpected cmd: %q", cmd)
	}
	if len(args) != 1 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestCommandForRevealLinuxFallsBackToGio(t *testing.T) {
	prev := lookPath
	lookPath = func(name string) (string, error) {
		if name == "gio" {
			return "/usr/bin/gio", nil
		}
		return "", errors.New("nope")
	}
	t.Cleanup(func() { lookPath = prev })

	cmd, args, err := commandForReveal("linux", "/tmp/a.gif")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cmd != "gio" {
		t.Fatalf("unexpected cmd: %q", cmd)
	}
	if len(args) != 2 || args[0] != "open" {
		t.Fatalf("unexpected args: %#v", args)
	}
}
