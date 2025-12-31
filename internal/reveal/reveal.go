package reveal

import (
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
)

var (
	execCommand = exec.Command
	lookPath    = exec.LookPath
)

func Reveal(path string) error {
	if path == "" {
		return errors.New("empty path")
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	cmd, args, err := commandForReveal(runtime.GOOS, path)
	if err != nil {
		return err
	}
	c := execCommand(cmd, args...)
	c.Stdout = io.Discard
	c.Stderr = io.Discard
	return c.Run()
}

func commandForReveal(goos string, path string) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{"-R", path}, nil
	case "windows":
		return "explorer.exe", []string{"/select," + filepath.Clean(path)}, nil
	default:
		dir := filepath.Dir(path)
		if _, err := lookPath("xdg-open"); err == nil {
			return "xdg-open", []string{dir}, nil
		}
		if _, err := lookPath("gio"); err == nil {
			return "gio", []string{"open", dir}, nil
		}
		return "", nil, errors.New("no file manager opener found (need xdg-open or gio)")
	}
}
