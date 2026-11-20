// Package zed opens folders in Zed.
package zed

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Open runs the configured Zed command (e.g. "zed" or "zeditor --new") on
// paths without waiting for it. A folder and a file in it open as one
// workspace showing the file.
func Open(command string, paths ...string) error {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return errors.New("no Zed command configured (Settings > Appearance)")
	}
	bin, err := lookPath(fields[0])
	if err != nil {
		return fmt.Errorf("%s is not on PATH (set the Zed command in Settings > Appearance)", fields[0])
	}
	cmd := exec.Command(bin, append(fields[1:], paths...)...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// Available reports whether the Zed command is on PATH.
func Available(command string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return false
	}
	_, err := lookPath(fields[0])
	return err == nil
}

// bundledCLI is the CLI inside Zed.app, which Zed's "Install CLI" links as
// zed; it works without that link. zeditor, the CLI's name in Linux
// packages, falls back to it too, so one config works on both.
const bundledCLI = "/Applications/Zed.app/Contents/MacOS/cli"

func lookPath(name string) (string, error) {
	bin, err := exec.LookPath(name)
	if err != nil && (name == "zed" || name == "zeditor") && runtime.GOOS == "darwin" {
		if b, e := exec.LookPath(bundledCLI); e == nil {
			return b, nil
		}
	}
	return bin, err
}
