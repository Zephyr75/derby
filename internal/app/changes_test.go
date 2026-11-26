package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"jockey/internal/store"
)

// The changed files follow the newest Explain report's order, the files it
// does not list come after, and without the file the order is untouched.
func TestSortByExplain(t *testing.T) {
	dir := t.TempDir()
	paths := func(fs []FileChange) string {
		s := ""
		for _, f := range fs {
			s += f.Path + " "
		}
		return s
	}
	files := func() []FileChange { return []FileChange{{Path: "a"}, {Path: "b"}, {Path: "c"}, {Path: "d"}} }

	got := files()
	sortByExplain(got, dir)
	if paths(got) != "a b c d " {
		t.Fatalf("no file: %s", paths(got))
	}
	if err := os.WriteFile(filepath.Join(dir, "EXPLAIN-1.files"), []byte("c\nb\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = files()
	sortByExplain(got, dir)
	if paths(got) != "c b a d " {
		t.Fatalf("listed first: %s", paths(got))
	}
}

// Bootstrap sends each task's view without what needs git; WarmTasks then
// sends the full views, one task:changed per task.
func TestBootstrapIsLightThenWarmed(t *testing.T) {
	a, _ := newTestApp(t)
	plat := a.plat.(*fakePlatform)
	tv, _ := a.CreateTask("core", "", "light", "", false, nil)
	os.WriteFile(filepath.Join(tv.Worktree, "change.txt"), []byte("x\n"), 0o644)
	id := store.StepID(tv.ID, "impl-1")
	_ = a.db.SaveStep(store.Step{ID: id, TaskID: tv.ID, Key: "impl-1", Agent: "impl", N: 1, Seq: 1, State: store.Done, StartedAt: time.Now()})
	count := func() (n int) {
		plat.mu.Lock()
		defer plat.mu.Unlock()
		for _, e := range plat.events {
			if e == EvTask {
				n++
			}
		}
		return
	}
	before := count()
	b, err := a.Bootstrap()
	if err != nil || len(b.Tasks) != 1 || len(b.Tasks[0].Steps) != 1 || b.Tasks[0].Steps[0].Diff {
		t.Fatalf("light view: %+v %v", b.Tasks, err)
	}
	if count() != before {
		t.Fatal("Bootstrap sent task events itself")
	}
	a.WarmTasks()
	waitFor(t, "warmed", func() bool { return count() > before })
	full, _ := a.GetTask(tv.ID)
	if !full.Steps[0].Diff {
		t.Fatal("the full view should show the step's uncommitted change")
	}
}
