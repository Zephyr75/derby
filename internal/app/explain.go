package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"derby/internal/artifacts"
	"derby/internal/settings"
	"derby/internal/store"
)

// A Copilot step's files, in the run directory:
//   - COPILOT-n.md: an explanation of the PR's changes, written by a Claude
//     run derby starts with the step (its model and instructions: Settings >
//     Agents > Copilot);
//   - FEEDBACK-n.md: the step's state (front matter: status, pr, comments,
//     ci), Copilot's review and the CI checks, for the steps that fix them
//     (the PR tab shows both too).

const explaining = "_Claude is writing an explanation of the changes…_\n"

func (a *App) reviewPath(t store.Task, s store.Step) string {
	return filepath.Join(a.runDir(a.projectOf(t), t), fmt.Sprintf("FEEDBACK-%d.md", s.N))
}

// writeReviewFile rewrites FEEDBACK-n.md: its front matter with set
// applied, and its body when body is not nil.
func (a *App) writeReviewFile(t store.Task, s store.Step, set map[string]string, body *string) {
	a.copilotMu.Lock()
	defer a.copilotMu.Unlock()
	path := a.reviewPath(t, s)
	art, _ := artifacts.Read(path)
	text := strings.TrimSpace(art.Body)
	if body != nil {
		text = strings.TrimSpace(*body)
	}
	_ = os.WriteFile(path, []byte(frontMatter(art, set)+"\n"+text+"\n"), 0o644)
}

// frontMatter is the review file's front matter: art's, with set applied.
func frontMatter(art artifacts.Artifact, set map[string]string) string {
	var b strings.Builder
	b.WriteString("---\n")
	for _, k := range []string{"status", "pr", "comments", "ci"} {
		v, ok := set[k]
		if !ok {
			v = art.Get(k)
		}
		if v != "" {
			fmt.Fprintf(&b, "%s: %s\n", k, v)
		}
	}
	b.WriteString("---\n")
	return b.String()
}

// explainChanges runs Claude, without a terminal, to explain the PR's changes
// in the Copilot step's COPILOT-n.md.
func (a *App) explainChanges(t store.Task, s store.Step) {
	p := a.projectOf(t)
	runDir, wt, file := a.runDir(p, t), worktreePath(p, t), a.copilotPath(t, s)
	_ = os.WriteFile(file, []byte(explaining), 0o644)
	a.emit(EvArtifact, map[string]string{"taskId": t.ID, "file": filepath.Base(file)})
	stem := strings.TrimSuffix(filepath.Base(file), ".md")
	ag := agentFor(a.settings.Agents(), copilotAgent)
	prompt := settings.Fill(ag.Instructions, map[string]string{"pr": fmt.Sprint(t.PRNumber), "base": p.BaseBranch, "file": file}) +
		"\n\n" + settings.Fill(a.settings.Prompt(settings.PromptDiagrams), map[string]string{"stem": stem, "artifact": filepath.Base(file)})
	bin := strings.Fields(a.settings.Config().ClaudeCommand)
	if len(bin) == 0 {
		bin = []string{"claude"}
	}
	args := append([]string{}, bin[1:]...)
	if ag.Model != "" {
		args = append(args, "--model", ag.Model)
	}
	if ag.Effort != "" {
		args = append(args, "--effort", ag.Effort)
	}
	args = append(args, "-p", "--permission-mode", "auto", "--settings", a.hooksPath(), "--add-dir", runDir)
	for _, dir := range a.knowledgeDirs(p) {
		args = append(args, "--add-dir", dir)
	}
	args = append(args, "--append-system-prompt", a.explainSystemPrompt(t, p), "--", prompt)
	ctx, cancel := context.WithCancel(a.ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin[0], args...)
	cmd.Dir = wt
	// No DERBY_TASK: derby's hooks do nothing for this run.
	cmd.Env = append(slices.DeleteFunc(os.Environ(), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return slices.Contains(inheritedSessionVars, name) || strings.HasPrefix(name, "DERBY_")
	}), "DERBY_RUN_DIR="+runDir, "DERBY_ARTIFACT="+file)
	out, err := cmd.CombinedOutput()
	if a.ctx.Err() != nil {
		return // derby is quitting
	}
	if b, _ := os.ReadFile(file); strings.TrimSpace(string(b)) == "" || string(b) == explaining { // what it wrote counts, whatever its exit
		why := strings.TrimSpace(string(out))
		if err == nil && why != "" { // it answered with the explanation instead of writing the file
			_ = os.WriteFile(file, []byte(why+"\n"), 0o644)
			a.emit(EvArtifact, map[string]string{"taskId": t.ID, "file": filepath.Base(file)})
			return
		}
		if err != nil {
			why = strings.TrimSpace(err.Error() + ": " + lastLines(why, 3))
		}
		_ = os.WriteFile(file, []byte(fmt.Sprintf("_Derby could not write an explanation of the changes (%s)._\n", strings.ReplaceAll(why, "\n", " "))), 0o644)
	}
	a.emit(EvArtifact, map[string]string{"taskId": t.ID, "file": filepath.Base(file)})
}

// RetryExplanation runs the Claude run again that explains a Copilot step's
// PR, when the first one failed (the explanation says so).
func (a *App) RetryExplanation(stepID string) error {
	s, err := a.db.Step(stepID)
	if err != nil {
		return err
	}
	if s.Agent != copilotAgent {
		return errors.New("only a Copilot step has an explanation")
	}
	t, err := a.db.Task(s.TaskID)
	if err != nil {
		return err
	}
	if t.PRNumber == 0 {
		return errors.New("the task has no pull request")
	}
	a.copilotMu.Lock()
	b, _ := os.ReadFile(a.copilotPath(t, s))
	busy := string(b) == explaining
	if !busy {
		_ = os.WriteFile(a.copilotPath(t, s), []byte(explaining), 0o644)
	}
	a.copilotMu.Unlock()
	if busy {
		return errors.New("Claude is already writing it")
	}
	a.explaining.Add(1)
	go func() {
		defer a.explaining.Done()
		a.explainChanges(t, s)
	}()
	return nil
}

// explainSystemPrompt is the system prompt of the Claude run that explains a
// PR: what it works on, then the Writing prompt and the Rules.
func (a *App) explainSystemPrompt(t store.Task, p settings.Project) string {
	sys := a.settings.Render(settings.PromptExplain, struct{ Task, Project, Worktree, RunDir string }{t.ID, p.Name, worktreePath(p, t), a.runDir(p, t)})
	if w := strings.TrimSpace(a.settings.Prompt(settings.PromptWriting)); w != "" {
		sys += "\n\n" + w
	}
	if r := a.rulesText(); r != "" {
		sys += "\n\n" + r
	}
	return sys
}

// renameArtifacts gives a run directory's artifacts their current names:
// PR.md became PUSH.md, and PR-REVIEW-n.md FEEDBACK-n.md. (A FEEDBACK.md,
// without a number, is from much older derby: nothing reads it.)
func renameArtifacts(runDir string) {
	rename := func(from, to string) {
		if _, err := os.Stat(filepath.Join(runDir, to)); err != nil {
			_ = os.Rename(filepath.Join(runDir, from), filepath.Join(runDir, to))
		}
	}
	rename("PR.md", "PUSH.md")
	ents, _ := os.ReadDir(runDir)
	for _, e := range ents {
		if n, ok := strings.CutPrefix(e.Name(), "PR-REVIEW-"); ok {
			rename(e.Name(), "FEEDBACK-"+n)
		}
	}
}

// abandonedExplanations marks the explanations a quit of derby cut short as
// failed, so the Retry button shows: no run writes them anymore.
func abandonedExplanations(runDir string) {
	files, _ := filepath.Glob(filepath.Join(runDir, "COPILOT-*.md"))
	for _, f := range files {
		if b, _ := os.ReadFile(f); string(b) == explaining {
			_ = os.WriteFile(f, []byte("_Derby could not write an explanation of the changes (derby quit before it finished)._\n"), 0o644)
		}
	}
}

// migrateCopilotFiles moves a run directory's Copilot files to their
// current names. Older derby wrote COPILOT-n.md (state, explanation and a
// footer), COPILOT-n-review.md (the review) and .COPILOT-n-explain.md (the
// explanation as Claude wrote it); before that, COPILOT-n.md held the state
// and the review.
func migrateCopilotFiles(runDir string) {
	ents, err := os.ReadDir(runDir)
	if err != nil {
		return
	}
	old := regexp.MustCompile(`^\.?COPILOT-(\d+)(-review|-explain)?\.md$`)
	var ns []string
	for _, e := range ents {
		if m := old.FindStringSubmatch(e.Name()); m != nil && !slices.Contains(ns, m[1]) {
			ns = append(ns, m[1])
		}
	}
	for _, n := range ns {
		path := filepath.Join(runDir, "COPILOT-"+n+".md")
		review := filepath.Join(runDir, "FEEDBACK-"+n+".md")
		if _, err := os.Stat(review); err == nil {
			continue // done already
		}
		oldReview := filepath.Join(runDir, "COPILOT-"+n+"-review.md")
		oldExplain := filepath.Join(runDir, ".COPILOT-"+n+"-explain.md")
		art, _ := artifacts.Read(path)
		rb, reviewErr := os.ReadFile(oldReview)
		eb, explainErr := os.ReadFile(oldExplain)
		if reviewErr != nil && explainErr != nil {
			if art.Get("comments") != "" || art.Get("status") != "" {
				// The review itself in COPILOT-n.md: it stays there to read,
				// and FEEDBACK-n.md gets a copy for the steps that read it.
				b, _ := os.ReadFile(path)
				_ = os.WriteFile(review, b, 0o644)
			}
			continue
		}
		// The last layout: the state goes to FEEDBACK-n.md with the review,
		// and COPILOT-n.md keeps only the explanation.
		if reviewErr == nil {
			_ = os.WriteFile(review, []byte(frontMatter(art, nil)+"\n"+strings.TrimSpace(string(rb))+"\n"), 0o644)
		}
		explanation := strings.TrimSpace(art.Body)
		if i := strings.LastIndex(explanation, "\n---\n"); i >= 0 && strings.HasPrefix(strings.TrimSpace(explanation[i+5:]), "Copilot's review and the CI checks are in the PR tab") {
			explanation = strings.TrimSpace(explanation[:i]) // the footer
		}
		if strings.TrimSpace(string(eb)) != "" {
			explanation = strings.TrimSpace(string(eb))
		}
		if explanation != "" {
			_ = os.WriteFile(path, []byte(explanation+"\n"), 0o644)
		}
		_ = os.Remove(oldReview)
		_ = os.Remove(oldExplain)
	}
}
