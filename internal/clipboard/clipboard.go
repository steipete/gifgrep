package clipboard

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
)

var lookPath = exec.LookPath

func CopyFile(path string) error {
	if path == "" {
		return errors.New("empty path")
	}
	cmdSpec, err := copyCommand(runtime.GOOS, path)
	if err != nil {
		return err
	}
	cmd := exec.Command(cmdSpec.name, cmdSpec.args...)
	if cmdSpec.stdinPath != "" {
		f, err := os.Open(cmdSpec.stdinPath)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		cmd.Stdin = f
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

type commandSpec struct {
	name      string
	args      []string
	stdinPath string
}

func copyCommand(goos string, path string) (commandSpec, error) {
	switch goos {
	case "darwin":
		return commandSpec{
			name: "osascript",
			args: []string{
				"-e",
				`on run argv
set the clipboard to (POSIX file (item 1 of argv))
end run`,
				path,
			},
		}, nil
	default:
		if _, err := lookPath("xclip"); err == nil {
			return commandSpec{name: "xclip", args: []string{"-selection", "clipboard", "-t", "image/gif", "-i", path}}, nil
		}
		if _, err := lookPath("wl-copy"); err == nil {
			return commandSpec{name: "wl-copy", args: []string{"--type", "image/gif"}, stdinPath: path}, nil
		}
		return commandSpec{}, errors.New("no clipboard tool found (need xclip or wl-copy)")
	}
}
