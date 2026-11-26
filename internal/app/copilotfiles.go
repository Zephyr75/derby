package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"jockey/internal/artifacts"
	"jockey/internal/store"
)

// A Copilot step writes one file in the run directory, FEEDBACK-n.md: the
// step's state (front matter: status, pr, comments, ci), Copilot's review and
// the CI checks, for the steps that fix them (the PR tab shows them too).

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

// renameArtifacts gives a run directory's artifacts their current names:
// PR.md became PUSH.md, and PR-REVIEW-n.md FEEDBACK-n.md. (A FEEDBACK.md,
// without a number, is from much older jockey: nothing reads it.)
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

// migrateCopilotFiles moves a run directory's Copilot files to their
// current names. Older jockey wrote COPILOT-n.md (state, explanation and a
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
