// Package gitops runs git for worktrees, commits, step diff views and push.
// Every command runs with -C <path> so repositories never share state.
package gitops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Log receives each command jockey runs, for toasts and the Push tab.
type Log func(line string)

// Git runs git commands.
type Git struct {
	Bin string // "git"
	Log Log
}

func (g Git) bin() string {
	if g.Bin == "" {
		return "git"
	}
	return g.Bin
}

// Run runs git -C dir args... and returns trimmed stdout.
func (g Git) Run(ctx context.Context, dir string, args ...string) (string, error) {
	out, err := g.output(ctx, dir, args...)
	return strings.TrimSpace(out), err
}

// File returns a file's content at a revision (path from the repository's
// root), untrimmed: its lines keep their numbers.
func (g Git) File(ctx context.Context, dir, rev, path string) (string, error) {
	return g.output(ctx, dir, "cat-file", "-p", rev+":"+path)
}

func (g Git) output(ctx context.Context, dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir}, args...)
	if g.Log != nil {
		g.Log("$ git " + quoteArgs(full))
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, g.bin(), full...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = strings.TrimSpace(out.String())
		}
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return out.String(), nil
}

func quoteArgs(a []string) string {
	out := make([]string, len(a))
	for i, s := range a {
		if strings.ContainsAny(s, " \t\"'$") {
			out[i] = "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
		} else {
			out[i] = s
		}
	}
	return strings.Join(out, " ")
}

// IsRepo reports whether path is a git work tree.
func (g Git) IsRepo(ctx context.Context, path string) bool {
	out, err := g.Run(ctx, path, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// GitDir returns the repository's common .git directory.
func (g Git) GitDir(ctx context.Context, repo string) (string, error) {
	d, err := g.Run(ctx, repo, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(d) {
		d = filepath.Join(repo, d)
	}
	return d, nil
}

// EnsureExcluded adds pattern to .git/info/exclude when missing.
func (g Git) EnsureExcluded(ctx context.Context, repo, pattern string) (added bool, err error) {
	gd, err := g.GitDir(ctx, repo)
	if err != nil {
		return false, err
	}
	p := filepath.Join(gd, "info", "exclude")
	b, _ := os.ReadFile(p)
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == pattern {
			return false, nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return false, err
	}
	f, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	prefix := ""
	if len(b) > 0 && !bytes.HasSuffix(b, []byte("\n")) {
		prefix = "\n"
	}
	_, err = f.WriteString(prefix + pattern + "\n")
	return err == nil, err
}

// IsExcluded reports whether pattern is in .git/info/exclude.
func (g Git) IsExcluded(ctx context.Context, repo, pattern string) bool {
	gd, err := g.GitDir(ctx, repo)
	if err != nil {
		return false
	}
	b, _ := os.ReadFile(filepath.Join(gd, "info", "exclude"))
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == pattern {
			return true
		}
	}
	return false
}

var repoRe = regexp.MustCompile(`github\.com[:/]([^/]+/[^/]+?)(?:\.git)?/?$`)

// GitHubRepo returns owner/name from the origin remote, or "".
func (g Git) GitHubRepo(ctx context.Context, repo string) string {
	u, err := g.Run(ctx, repo, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	m := repoRe.FindStringSubmatch(u)
	if m == nil {
		return ""
	}
	return m[1]
}

// DefaultBranch returns origin's HEAD branch, or the current branch.
func (g Git) DefaultBranch(ctx context.Context, repo string) string {
	if ref, err := g.Run(ctx, repo, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimPrefix(ref, "origin/")
	}
	if b, err := g.Run(ctx, repo, "branch", "--show-current"); err == nil && b != "" {
		return b
	}
	return "main"
}

// AddWorktree fetches origin and creates branch at path from base. It falls
// back to the local base branch when there is no origin.
func (g Git) AddWorktree(ctx context.Context, repo, path, branch, base string) error {
	start := base
	if _, err := g.Run(ctx, repo, "fetch", "origin", base); err == nil {
		start = "origin/" + base
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := g.Run(ctx, repo, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		_, err := g.Run(ctx, repo, "worktree", "add", path, branch)
		return err
	}
	_, err := g.Run(ctx, repo, "worktree", "add", "-b", branch, path, start)
	return err
}

// RemoveWorktree removes a task worktree and deletes its branch.
func (g Git) RemoveWorktree(ctx context.Context, repo, path, branch string) error {
	var errs []error
	if _, err := os.Stat(path); err == nil {
		if _, err := g.Run(ctx, repo, "worktree", "remove", "--force", path); err != nil {
			errs = append(errs, err)
		}
	}
	_, _ = g.Run(ctx, repo, "worktree", "prune")
	if branch != "" {
		if _, err := g.Run(ctx, repo, "branch", "-D", branch); err != nil && !strings.Contains(err.Error(), "not found") {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// PruneViews removes every view worktree under dir that belongs to repo.
func (g Git) PruneViews(ctx context.Context, repo, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		_, _ = g.Run(ctx, repo, "worktree", "remove", "--force", p)
		_ = os.RemoveAll(p)
	}
	_, _ = g.Run(ctx, repo, "worktree", "prune")
}

// privateIndex runs git in worktree with its own index file, so the
// worktree's real index is never touched.
func (g Git) privateIndex(ctx context.Context, worktree, index string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, g.bin(), append([]string{"-C", worktree}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_INDEX_FILE="+index, "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// SnapshotIndex is where a worktree's snapshot index lives: in its own git
// directory, so it goes with the worktree and never shows in it.
func (g Git) SnapshotIndex(ctx context.Context, worktree string) (string, error) {
	p, err := g.Run(ctx, worktree, "rev-parse", "--git-path", "jockey-snapshot-index")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(worktree, p)
	}
	return p, nil
}

// Snapshot stores the worktree's content, untracked files included (not
// ignored ones), as a tree and returns its hash. The index starts from HEAD:
// tracked files that .gitignore matches stay in, and files committed with
// CRLF keep them.
func (g Git) Snapshot(ctx context.Context, worktree, index string) (string, error) {
	if _, err := g.privateIndex(ctx, worktree, index, "read-tree", "HEAD"); err != nil {
		return "", err
	}
	if _, err := g.privateIndex(ctx, worktree, index, "add", "-A"); err != nil {
		return "", err
	}
	return g.privateIndex(ctx, worktree, index, "write-tree")
}

// Restore puts the worktree back to a Snapshot tree, as uncommitted changes
// on top of HEAD: anything changed since is discarded.
func (g Git) Restore(ctx context.Context, worktree, tree, index string) error {
	if _, err := g.Run(ctx, worktree, "reset", "--hard", "HEAD"); err != nil {
		return err
	}
	if _, err := g.Run(ctx, worktree, "clean", "-fd"); err != nil {
		return err
	}
	gone, err := g.Run(ctx, worktree, "-c", "core.quotePath=false", "diff", "--name-only", "--no-renames", "--diff-filter=D", "HEAD", tree)
	if err != nil {
		return err
	}
	if _, err := g.privateIndex(ctx, worktree, index, "read-tree", tree); err != nil {
		return err
	}
	if _, err := g.privateIndex(ctx, worktree, index, "checkout-index", "-a", "-f"); err != nil {
		return err
	}
	for _, rel := range strings.Split(gone, "\n") {
		if rel != "" {
			_ = os.Remove(filepath.Join(worktree, rel))
		}
	}
	return nil
}

// Worktree is one of a repository's checkouts.
type Worktree struct {
	Path   string `json:"path"`
	Branch string `json:"branch"` // "" when detached
	Main   bool   `json:"main"`   // the repository's own checkout
}

// Worktrees lists repo's checkouts, its own first, skipping bare and missing
// ones.
func (g Git) Worktrees(ctx context.Context, repo string) ([]Worktree, error) {
	out, err := g.Run(ctx, repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var list []Worktree
	for i, block := range strings.Split(out, "\n\n") {
		var w Worktree
		skip := false
		for _, line := range strings.Split(block, "\n") {
			key, val, _ := strings.Cut(line, " ")
			switch key {
			case "worktree":
				w.Path = val
			case "branch":
				w.Branch = strings.TrimPrefix(val, "refs/heads/")
			case "bare", "prunable":
				skip = true
			}
		}
		if w.Path == "" || skip {
			continue
		}
		w.Main = i == 0
		list = append(list, w)
	}
	return list, nil
}

// Branches lists repo's local branches, most recently committed first.
func (g Git) Branches(ctx context.Context, repo string) ([]string, error) {
	out, err := g.Run(ctx, repo, "for-each-ref", "--sort=-committerdate", "--format=%(refname:short)", "refs/heads")
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}
