//go:build darwin

package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

// An app opened from Finder or the Dock gets launchd's PATH
// (/usr/bin:/bin:/usr/sbin:/sbin), which misses Homebrew and the user's own
// bin folders, so claude, gh and zed are not found. useLoginShellPath takes
// PATH from the user's login shell instead, as a terminal would see it.
func useLoginShellPath() {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Interactive, so .zshrc/.bashrc run too; the markers cut away anything
	// the rc files print.
	const mark = "__DERBY_PATH__"
	out, err := exec.CommandContext(ctx, shell, "-ilc", "printf '"+mark+"%s"+mark+"' \"$PATH\"").Output()
	if err != nil {
		log.Printf("derby: reading PATH from %s: %v", shell, err)
		return
	}
	parts := strings.Split(string(out), mark)
	if len(parts) < 3 || parts[1] == "" {
		return
	}
	_ = os.Setenv("PATH", parts[1])
}
