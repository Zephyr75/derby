package app

// Task names: a few words an agent picks for each task, to tell tasks apart
// in the task list, tabs and title bar. An issue's title is often long or
// vague; the name says what the task is about.

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"derby/internal/settings"
	"derby/internal/store"
)

// nameTasks names tasks in the background, one after the other: the given
// ones, else every task not archived that has no name yet.
func (a *App) nameTasks(ids ...string) {
	if len(ids) == 0 {
		ts, _ := a.db.Tasks()
		for _, t := range ts {
			if t.Name == "" && !t.Archived {
				ids = append(ids, t.ID)
			}
		}
	}
	if len(ids) == 0 {
		return
	}
	a.explaining.Add(1)
	go func() {
		defer a.explaining.Done()
		for _, id := range ids {
			if a.ctx.Err() != nil {
				return
			}
			a.nameTask(id)
		}
	}()
}

// nameTask asks Claude for the task's name, from its issue (ISSUE.md) and
// description. The task keeps showing its title when that fails.
func (a *App) nameTask(id string) {
	t, err := a.db.Task(id)
	if err != nil || t.Name != "" {
		return
	}
	var d struct{ Issue, Description string }
	if t.Issue != "" {
		b, _ := os.ReadFile(filepath.Join(a.runDir(a.projectOf(t), t), issueFile))
		d.Issue = clip(string(b), 3000)
		if d.Issue == "" {
			d.Issue = t.Title
		}
	}
	d.Description = clip(t.Description, 2000)
	name := cleanName(string(a.haiku(a.settings.Render(settings.PromptName, d))))
	if name == "" {
		return
	}
	if err := a.updateTask(id, func(t *store.Task) error { t.Name = name; return nil }); err != nil {
		log.Printf("name %s: %v", id, err)
	}
}

// cleanName is the name in Claude's answer: its first line, without quotes,
// emphasis or a final period. "" when it is no short name.
func cleanName(answer string) string {
	for _, line := range strings.Split(answer, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		for _, p := range []string{"Name:", "Task name:"} {
			if len(line) > len(p) && strings.EqualFold(line[:len(p)], p) {
				line = strings.TrimSpace(line[len(p):])
			}
		}
		line = strings.Trim(line, " \"'`*_.“”‘’")
		if n := len(strings.Fields(line)); n == 0 || n > 7 || utf8.RuneCountInString(line) > 50 {
			return ""
		}
		return line
	}
	return ""
}

// haiku runs a prompt through Claude Code's Haiku, without a session
// (claude -p), and returns its answer.
func (a *App) haiku(prompt string) []byte {
	bin := strings.Fields(a.settings.Config().ClaudeCommand)
	if len(bin) == 0 {
		bin = []string{"claude"}
	}
	ctx, cancel := context.WithTimeout(a.ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin[0], append(bin[1:], "-p", "--model", "haiku", "--", prompt)...)
	cmd.Dir = a.Home
	cmd.Env = slices.DeleteFunc(os.Environ(), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return slices.Contains(inheritedSessionVars, name) || strings.HasPrefix(name, "DERBY_")
	})
	out, _ := cmd.Output()
	return out
}
