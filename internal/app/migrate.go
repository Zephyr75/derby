package app

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Derby was called herd: its folder was ~/.herd, its commands /herd:<agent>
// and its variables HERD_*. MigrateHome moves that folder to ~/.derby once,
// leaving a link at the old path for the Claude Code sessions started before
// (their run directories and hooks settings), and renames them in your
// agents.toml and prompts.toml, so they stay identical to defaults/.
func MigrateHome() {
	if os.Getenv("DERBY_HOME") != "" {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	if err := migrateHome(filepath.Join(home, ".herd"), filepath.Join(home, ".derby")); err != nil {
		log.Printf("moving ~/.herd to ~/.derby: %v", err)
	}
}

// oldNames renames herd's names in a settings file, as the repository was.
var oldNames = strings.NewReplacer("HERD", "DERBY", "Herd", "Derby", "herd", "derby")

func migrateHome(old, cur string) error {
	if _, err := os.Lstat(cur); err == nil {
		return nil // moved already, or a fresh ~/.derby
	}
	if fi, err := os.Lstat(old); err != nil || !fi.IsDir() {
		return nil // nothing to move
	}
	if err := os.Rename(old, cur); err != nil {
		return err
	}
	if err := os.Symlink(cur, old); err != nil {
		log.Printf("linking ~/.herd to ~/.derby: %v", err)
	}
	for _, name := range []string{"agents.toml", "prompts.toml"} {
		path := filepath.Join(cur, name)
		if b, err := os.ReadFile(path); err == nil {
			if err := os.WriteFile(path, []byte(oldNames.Replace(string(b))), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}
