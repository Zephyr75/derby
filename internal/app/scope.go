package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"derby/internal/gitops"
	"derby/internal/settings"
	"derby/internal/zed"
)

// Branch is a project's local branch and the checkout it is in, if any.
type Branch struct {
	Name string `json:"name"`
	Path string `json:"path"` // "" when no worktree has it checked out
}

func (a *App) project(id string) (settings.Project, error) {
	p, ok := a.settings.Project(id)
	if !ok {
		return p, fmt.Errorf("unknown project %q", id)
	}
	return p, nil
}

// ProjectWorktrees lists a project's checkouts for the top bar, its own first.
func (a *App) ProjectWorktrees(projectID string) ([]gitops.Worktree, error) {
	p, err := a.project(projectID)
	if err != nil {
		return nil, err
	}
	return a.git.Worktrees(a.ctx, p.Path)
}

// ProjectBranches lists a project's local branches for the top bar, most
// recent first.
func (a *App) ProjectBranches(projectID string) ([]Branch, error) {
	p, err := a.project(projectID)
	if err != nil {
		return nil, err
	}
	names, err := a.git.Branches(a.ctx, p.Path)
	if err != nil {
		return nil, err
	}
	wts, _ := a.git.Worktrees(a.ctx, p.Path)
	at := map[string]string{}
	for _, w := range wts {
		if w.Branch != "" {
			at[w.Branch] = w.Path
		}
	}
	list := make([]Branch, len(names))
	for i, n := range names {
		list[i] = Branch{Name: n, Path: at[n]}
	}
	return list, nil
}

// OpenWorktreeInZed opens one of a project's checkouts.
func (a *App) OpenWorktreeInZed(projectID, path string) error {
	p, err := a.project(projectID)
	if err != nil {
		return err
	}
	wts, err := a.git.Worktrees(a.ctx, p.Path)
	if err != nil {
		return err
	}
	for _, w := range wts {
		if filepath.Clean(w.Path) == filepath.Clean(path) {
			return zed.Open(a.settings.Config().ZedCommand, w.Path)
		}
	}
	return fmt.Errorf("%s is not a worktree of %s", path, p.Name)
}

// OpenBranchInZed opens the checkout that has branch. A branch checked out
// nowhere is switched to in the project's own checkout, when it has no
// uncommitted changes.
func (a *App) OpenBranchInZed(projectID, branch string) error {
	p, err := a.project(projectID)
	if err != nil {
		return err
	}
	if _, err := a.git.Run(a.ctx, p.Path, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err != nil {
		return fmt.Errorf("no branch %s in %s", branch, p.Name)
	}
	wts, err := a.git.Worktrees(a.ctx, p.Path)
	if err != nil {
		return err
	}
	for _, w := range wts {
		if w.Branch == branch {
			return zed.Open(a.settings.Config().ZedCommand, w.Path)
		}
	}
	if dirty, err := a.git.Run(a.ctx, p.Path, "status", "--porcelain", "--untracked-files=no"); err != nil {
		return err
	} else if dirty != "" {
		return errors.New(p.Name + " has uncommitted changes")
	}
	if _, err := a.git.Run(a.ctx, p.Path, "switch", "--quiet", branch); err != nil {
		return err
	}
	return zed.Open(a.settings.Config().ZedCommand, p.Path)
}
