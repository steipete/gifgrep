package clipboard

import (
	"errors"
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
	// Nil outputs use the null device, not pipes kept open by clipboard daemons.
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
		xclip := commandSpec{name: "xclip", args: []string{"-selection", "clipboard", "-t", "image/gif", "-i", path}}
		wlCopy := commandSpec{name: "wl-copy", args: []string{"--type", "image/gif"}, stdinPath: path}
		commands := []commandSpec{xclip, wlCopy}
		if os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland" {
			commands = []commandSpec{wlCopy, xclip}
		}
		for _, command := range commands {
			if _, err := lookPath(command.name); err == nil {
				return command, nil
			}
		}
		return commandSpec{}, errors.New("no clipboard tool found (need xclip or wl-copy)")
	}
}
