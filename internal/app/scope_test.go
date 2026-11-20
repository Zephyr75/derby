package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"derby/internal/settings"
)

func TestScopeLists(t *testing.T) {
	a, repo := newTestApp(t)
	if err := a.settings.UpdateConfig(func(c *settings.Config) { c.ZedCommand = "true" }); err != nil {
		t.Fatal(err)
	}
	wt := filepath.Join(t.TempDir(), "feature")
	run(t, repo, "git", "worktree", "add", "-q", "-b", "feature", wt)
	run(t, repo, "git", "branch", "side")

	wts, err := a.ProjectWorktrees("core")
	if err != nil || len(wts) != 2 || !wts[0].Main || wts[0].Branch != "main" || wts[1].Branch != "feature" {
		t.Fatalf("worktrees %+v, %v", wts, err)
	}
	brs, err := a.ProjectBranches("core")
	if err != nil {
		t.Fatal(err)
	}
	at := map[string]string{}
	for _, b := range brs {
		at[b.Name] = b.Path
	}
	if len(brs) != 3 || at["main"] == "" || at["feature"] == "" || at["side"] != "" {
		t.Fatalf("branches %+v", brs)
	}

	if err := a.OpenWorktreeInZed("core", "/elsewhere"); err == nil {
		t.Fatal("opened a folder that is not a worktree")
	}
	if err := a.OpenWorktreeInZed("core", wt); err != nil {
		t.Fatal(err)
	}
	// A branch checked out nowhere is switched to in the clean main checkout.
	if err := a.OpenBranchInZed("core", "side"); err != nil {
		t.Fatal(err)
	}
	if b := run(t, repo, "git", "branch", "--show-current"); b != "side" {
		t.Fatalf("main checkout on %q", b)
	}
	// ...but never over uncommitted changes.
	run(t, repo, "git", "branch", "other")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.OpenBranchInZed("core", "other"); err == nil || !strings.Contains(err.Error(), "uncommitted") {
		t.Fatalf("switched a dirty checkout: %v", err)
	}
}
