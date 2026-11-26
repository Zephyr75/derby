package app

import (
	"os"
	"path/filepath"
	"testing"
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
