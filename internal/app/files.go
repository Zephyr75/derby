package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"derby/internal/settings"
	"derby/internal/store"
)

// attachDir holds the files attached at task creation, in the run dir.
const attachDir = "attachments"

// copyAttachments copies files to dir/attachments, renaming duplicates, and
// returns where they went.
func copyAttachments(dir string, files []string) ([]string, error) {
	if len(files) == 0 {
		return nil, nil
	}
	dst := filepath.Join(dir, attachDir)
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return nil, err
	}
	var out []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		ext := filepath.Ext(f)
		base := strings.TrimSuffix(filepath.Base(f), ext)
		name := base + ext
		for i := 2; ; i++ {
			if _, err := os.Stat(filepath.Join(dst, name)); os.IsNotExist(err) {
				break
			}
			name = fmt.Sprintf("%s-%d%s", base, i, ext)
		}
		if err := os.WriteFile(filepath.Join(dst, name), b, 0o644); err != nil {
			return nil, err
		}
		out = append(out, filepath.Join(dst, name))
	}
	return out, nil
}

// attachments lists the attached files' names, sorted.
func attachments(dir string) []string {
	es, _ := os.ReadDir(filepath.Join(dir, attachDir))
	var names []string
	for _, e := range es {
		if e.Type().IsRegular() {
			names = append(names, e.Name())
		}
	}
	return names
}

// writeTaskFile writes TASK.md to the run dir: the task source every agent
// reads first.
func (a *App) writeTaskFile(p settings.Project, t store.Task) error {
	dir := a.runDir(p, t)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	d := struct {
		Source, Issue, Repo, Project, Branch, Worktree, Base, Title, IssueFile, Description string
		Copied                                                                              bool
		Attachments                                                                         []string
	}{Source: "local", Issue: t.Issue, Project: p.Name, Branch: branchName(t), Worktree: worktreePath(p, t), Base: p.BaseBranch, Title: t.Title, IssueFile: issueFile, Description: strings.TrimSpace(t.Description)}
	switch {
	case t.Issue != "" && t.Description != "":
		d.Source = "issue+description"
	case t.Issue != "":
		d.Source = "issue"
	}
	if t.Issue != "" {
		d.Repo = a.issueRepo(t)
		_, err := os.Stat(filepath.Join(dir, issueFile))
		d.Copied = err == nil
	}
	for _, n := range attachments(dir) {
		d.Attachments = append(d.Attachments, filepath.Join(dir, attachDir, n))
	}
	return os.WriteFile(filepath.Join(dir, "TASK.md"), []byte(a.settings.Render(settings.PromptTaskFile, d)), 0o644)
}

// contextText is the Context part of a step's system prompt: the general
// context, then the project's, then the knowledge base folders.
func (a *App) contextText(p settings.Project) string {
	var d struct {
		General, Project string
		ClaudeMd         bool
		Knowledge        []string
	}
	if c, ok := a.settings.Context("general"); ok {
		d.General = strings.TrimSpace(c.Text)
	}
	if c, ok := a.settings.Context(p.ID); ok {
		d.Project, d.ClaudeMd = strings.TrimSpace(c.Text), c.ClaudeMd
	}
	d.Knowledge = a.knowledgeDirs(p)
	if d.General == "" && d.Project == "" && !d.ClaudeMd && len(d.Knowledge) == 0 {
		return ""
	}
	return a.settings.Render(settings.PromptContext, d)
}

var cmdName = regexp.MustCompile(`^/derby:([A-Za-z0-9_-]+)`)

// commandName returns "implement" for "/derby:implement {task} -- {guidance}".
func commandName(command string) string {
	m := cmdName.FindStringSubmatch(strings.TrimSpace(command))
	if m == nil {
		return ""
	}
	return m[1]
}

// writePlugin rebuilds the derby Claude Code plugin in ~/.derby/plugin: its
// manifest, one command per agent, built from the agent's instructions, and
// the internal skills (prompts with a skill name, Settings > Context).
func (a *App) writePlugin() error {
	dir := a.pluginDir()
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, ".claude-plugin"), 0o755); err != nil {
		return err
	}
	manifest := `{"name": "derby", "version": "1.0.0", "description": "Agents run by derby: one command per agent in Settings > Agents"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".claude-plugin", "plugin.json"), []byte(manifest), 0o644); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "commands"), 0o755); err != nil {
		return err
	}
	for _, ag := range a.settings.Agents() {
		name := commandName(ag.Command)
		if name == "" {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, "commands", name+".md"), []byte(a.CommandText(ag)), 0o644); err != nil {
			return err
		}
	}
	for _, p := range a.settings.Snapshot().Prompts {
		if p.Skill == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Join(dir, "skills", p.Skill), 0o755); err != nil {
			return err
		}
		text := fmt.Sprintf("---\nname: %s\ndescription: %q\n---\n\n%s\n", p.Skill, p.Where, strings.TrimSpace(p.Text))
		if err := os.WriteFile(filepath.Join(dir, "skills", p.Skill, "SKILL.md"), []byte(text), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// CommandText is the command derby writes for an agent (/derby:<name>): just
// its own instructions; what every agent shares is in the system prompt.
func (a *App) CommandText(ag settings.Agent) string {
	return a.settings.Render(settings.PromptCommand, struct{ Name, Instructions string }{ag.Name, strings.TrimSpace(ag.Instructions)})
}
