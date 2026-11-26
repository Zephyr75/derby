package app

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Jockey was called derby: its folder was ~/.derby, its commands /derby:<agent>
// and its variables DERBY_*. MigrateHome moves that folder to ~/.jockey once,
// leaving a link at the old path for the Claude Code sessions started before
// (their run directories and hooks settings), and renames them in your
// agents.toml and prompts.toml, so they stay identical to defaults/.
func MigrateHome() {
	if os.Getenv("JOCKEY_HOME") != "" {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	if err := migrateHome(filepath.Join(home, ".derby"), filepath.Join(home, ".jockey")); err != nil {
		log.Printf("moving ~/.derby to ~/.jockey: %v", err)
	}
}

// oldNames renames derby's names in a settings file, as the repository was.
var oldNames = strings.NewReplacer("DERBY", "JOCKEY", "Derby", "Jockey", "derby", "jockey")

func migrateHome(old, cur string) error {
	if _, err := os.Lstat(cur); err == nil {
		return nil // moved already, or a fresh ~/.jockey
	}
	if fi, err := os.Lstat(old); err != nil || !fi.IsDir() {
		return nil // nothing to move
	}
	if err := os.Rename(old, cur); err != nil {
		return err
	}
	if err := os.Symlink(cur, old); err != nil {
		log.Printf("linking ~/.derby to ~/.jockey: %v", err)
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
