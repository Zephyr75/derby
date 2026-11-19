package app

import (
	"os"
	"path/filepath"
	"testing"

	"derby/internal/store"
)

func TestMigrateHome(t *testing.T) {
	root := t.TempDir()
	old, cur := filepath.Join(root, ".herd"), filepath.Join(root, ".derby")
	run := filepath.Join(old, "runs", "core", "42")
	os.MkdirAll(run, 0o755)
	os.WriteFile(filepath.Join(old, "agents.toml"), []byte("command = \"/herd:plan {task}\"\ninstructions = 'Herd copied it to $HERD_RUN_DIR'\n"), 0o644)
	os.WriteFile(filepath.Join(run, ".herd-task.json"), []byte(`{"task":{"id":"core:42","project":"core","slug":"42"},"steps":[]}`), 0o600)
	if err := migrateHome(old, cur); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(cur, "agents.toml")); string(b) != "command = \"/derby:plan {task}\"\ninstructions = 'Derby copied it to $DERBY_RUN_DIR'\n" {
		t.Fatalf("agents.toml:\n%s", b)
	}
	if target, err := os.Readlink(old); err != nil || target != cur {
		t.Fatalf("no link from the old folder: %q %v", target, err)
	}
	// The task store renames the old task file.
	db, err := store.Open(cur, func(store.Task) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Task("core:42"); err != nil {
		t.Fatal("task not found after the rename:", err)
	}
	// Once moved, it never moves again.
	if err := migrateHome(old, cur); err != nil {
		t.Fatal(err)
	}
}
