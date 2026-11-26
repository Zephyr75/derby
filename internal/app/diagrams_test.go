package app

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"jockey/internal/settings"
	"jockey/internal/store"
)

func TestDiagrams(t *testing.T) {
	a, _ := newTestApp(t)
	tv, _ := a.CreateTask("core", "", "draw it", "", false, nil)
	_ = a.db.SaveStep(store.Step{ID: store.StepID(tv.ID, "impl-2"), TaskID: tv.ID, Key: "impl-2", Agent: "impl", N: 2, Seq: 1, State: store.Done})
	os.MkdirAll(tv.RunDir, 0o755)
	write := func(name, body string) { os.WriteFile(filepath.Join(tv.RunDir, name), []byte(body), 0o644) }
	write("IMPLEMENT-2-flow.html", "<html><head><title>New lock flow</title></head></html>")
	write("other.html", "<html></html>")
	write("PLAN.md", "not a page")

	ds, err := a.Diagrams(tv.ID)
	if err != nil || len(ds) != 2 {
		t.Fatalf("diagrams: %+v %v", ds, err)
	}
	if ds[0].File != "IMPLEMENT-2-flow.html" || ds[0].Title != "New lock flow" || ds[0].Step != "Implement 2" || ds[1].Step != "" {
		t.Fatalf("diagrams: %+v", ds)
	}

	get := func(file string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		a.RunFiles().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/run/"+url.PathEscape(tv.ID)+"/"+file, nil))
		return w
	}
	w := get("IMPLEMENT-2-flow.html")
	if w.Code != 200 || !strings.Contains(w.Header().Get("Content-Security-Policy"), "sandbox allow-scripts") || !strings.Contains(w.Body.String(), "New lock flow") {
		t.Fatalf("served: %d %q", w.Code, w.Header())
	}
	if got := string(withTheme([]byte("<!doctype html><html lang=en><body>"), "dark")); got != `<!doctype html><html data-theme="dark" lang=en><body>` {
		t.Fatal(got)
	}
	if got := string(withTheme([]byte(`<html data-theme="light">`), "dark")); got != `<html data-theme="light">` {
		t.Fatal(got)
	}
	page := `<html lang="en" data-preset="classic"><style>svg[data-preset="classic"] .n{}</style><svg viewBox="0 0 1 1" data-preset="classic" role="img">`
	want := `<html lang="en" data-preset="signal-flow"><style>svg[data-preset="classic"] .n{}</style><svg viewBox="0 0 1 1" data-preset="signal-flow" role="img">`
	if got := string(withSignalFlow([]byte(page))); got != want {
		t.Fatal(got)
	}
	for _, bad := range []string{"PLAN.md", "..%2F..%2Fsecret.html", ".hidden.html", "missing.html"} {
		if w := get(bad); w.Code != 404 {
			t.Errorf("%s: %d", bad, w.Code)
		}
	}
}

// Agents with Sisters on (Plan and Implement by default) can ask for sister
// tasks; the others cannot.
func TestSisterOffersFromAllowedAgents(t *testing.T) {
	a, _ := newTestApp(t)
	tv, _ := a.CreateTask("core", "", "two repos", "", false, nil)
	os.MkdirAll(tv.RunDir, 0o755)
	ask := "---\nstatus: done\n---\n# What changed\nx\n\n---\n\n## Sister tasks\n\n### %s\n\nEcho it.\n"
	for i, s := range []struct{ key, agent, file, repo string }{{"impl-1", "impl", "IMPLEMENT-1.md", "okai-tcp-server"}, {"review-1", "review", "REVIEW-1.md", "FirebaseFunctions"}} {
		os.WriteFile(filepath.Join(tv.RunDir, s.file), []byte(strings.ReplaceAll(ask, "%s", s.repo)), 0o644)
		_ = a.db.SaveStep(store.Step{ID: store.StepID(tv.ID, s.key), TaskID: tv.ID, Key: s.key, Agent: s.agent, N: 1, Seq: i + 1, State: store.Done})
	}
	v, _ := a.GetTask(tv.ID)
	if len(v.Next.Sisters) != 1 || v.Next.Sisters[0].Repo != "okai-tcp-server" || !strings.HasSuffix(v.Next.Sisters[0].From, "IMPLEMENT-1.md") {
		t.Fatalf("offers: %+v", v.Next.Sisters)
	}
	task, _ := a.db.Task(tv.ID)
	for id, want := range map[string]bool{"impl": true, "review": false} {
		ag, _ := a.settings.Agent(id)
		if strings.Contains(a.stepBlock(launch{task: task, proj: a.projectOf(task), agent: ag, step: store.Step{N: 1}}, "/run", "/wt"), "# Sister tasks") != want {
			t.Errorf("%s: sister rules %v", id, !want)
		}
	}
}

// An agent can ask the user in a form (the step waits for them) and commit
// the work itself: jockey records the commit as the step's.
func TestFormThenCommit(t *testing.T) {
	a, _ := newTestApp(t)
	ags := append(a.settings.Agents(), settings.Agent{ID: "form", Name: "Form", Command: "/jockey:form {task}", Trigger: "manual", Writes: "FORM.md"})
	if err := a.SaveAgents(ags); err != nil {
		t.Fatal(err)
	}
	tv, _ := a.CreateTask("core", "", "ask then commit", "", false, nil)
	os.WriteFile(filepath.Join(tv.Worktree, "change.txt"), []byte("x\n"), 0o644)
	sv, err := a.StartStep(tv.ID, "form", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "form open", func() bool { return stepState(a, sv.ID) == store.Idle })
	sess, _ := a.run.Get(sv.ID)
	sess.Write([]byte("feat\r"))
	waitFor(t, "done", func() bool { return stepState(a, sv.ID) == store.Done })
	waitFor(t, "commit recorded", func() bool { s, _ := a.db.Step(sv.ID); return s.SHA != "" })
	s, _ := a.db.Step(sv.ID)
	if msg := run(t, tv.Worktree, "git", "log", "-1", "--format=%B"); msg != "feat(x): the title #1" || run(t, tv.Worktree, "git", "rev-parse", "--short", "HEAD") != s.SHA {
		t.Fatalf("commit %q, step sha %q", msg, s.SHA)
	}
}

// fakeGH stands in for gh: the branch has PR #7, titled and described.
func fakeGH(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gh")
	script := `#!/bin/sh
case "$*" in
  "pr list"*) printf '%s\n' '[{"number":7,"url":"https://github.com/o/pony-core/pull/7","state":"OPEN"}]' ;;
  *title,body*) printf '%s\n' '{"title":"feat(x): the title","body":"Because it was slow.\n\n## Flow\nmore"}' ;;
  *) exit 1 ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// The Push agent opens the PR itself: after it succeeds jockey finds the PR of
// the branch, Copilot's step starts with the PR's title and description in
// COPILOT-n.md; Explain is manual and stays unstarted.
func TestAfterPushFindsThePR(t *testing.T) {
	a, _ := newTestApp(t)
	a.gh.Bin = fakeGH(t)
	tv, _ := a.CreateTask("core", "", "push it", "", false, nil)
	id := store.StepID(tv.ID, "push-1")
	_ = a.db.SaveStep(store.Step{ID: id, TaskID: tv.ID, Key: "push-1", Agent: "push", N: 1, Seq: 1, State: store.Done, StartedAt: time.Now()})
	a.complete(id)
	waitFor(t, "copilot started", func() bool { _, err := a.db.Step(store.StepID(tv.ID, "copilot-1")); return err == nil })
	if _, err := a.db.Step(store.StepID(tv.ID, "explain-1")); err == nil {
		t.Fatal("Explain started by itself")
	}
	task, _ := a.db.Task(tv.ID)
	if task.PRNumber != 7 || task.PRState != "open" || !strings.HasSuffix(task.PRURL, "/pull/7") {
		t.Fatalf("task PR: %+v", task)
	}
	waitFor(t, "PR description", func() bool {
		b, _ := os.ReadFile(filepath.Join(tv.RunDir, "COPILOT-1.md"))
		return string(b) == "# feat(x): the title\n\nBecause it was slow.\n\n## Flow\nmore\n"
	})
}

// Without a pull request there is nothing for Copilot to review.
func TestAfterPushWithoutPR(t *testing.T) {
	a, _ := newTestApp(t)
	a.gh.Bin = "false"
	tv, _ := a.CreateTask("core", "", "no pr", "", false, nil)
	id := store.StepID(tv.ID, "push-1")
	_ = a.db.SaveStep(store.Step{ID: id, TaskID: tv.ID, Key: "push-1", Agent: "push", N: 1, Seq: 1, State: store.Done, StartedAt: time.Now()})
	a.complete(id)
	time.Sleep(300 * time.Millisecond)
	for _, key := range []string{"copilot-1", "explain-1"} {
		if _, err := a.db.Step(store.StepID(tv.ID, key)); err == nil {
			t.Fatalf("%s started without a PR", key)
		}
	}
}
