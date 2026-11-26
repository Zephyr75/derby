package app

import (
	"os"
	"testing"
)

// The fixtures are Claude Code 2.1.311's output in a new folder: the trust
// dialog, then after a Down press, then after Enter (the REPL).
func TestReadTrust(t *testing.T) {
	for _, c := range []struct {
		file  string
		ok    bool
		downs int
	}{
		{"trust-shown.bin", true, 1},
		{"trust-down.bin", true, 0},
		{"trust-accepted.bin", false, 0},
	} {
		b, err := os.ReadFile("testdata/" + c.file)
		if err != nil {
			t.Fatal(err)
		}
		folder, downs, ok := readTrust(b)
		if ok != c.ok || downs != c.downs {
			t.Errorf("%s: ok %v downs %d, want %v %d", c.file, ok, downs, c.ok, c.downs)
		}
		if ok && folder != "/private/tmp/jockey-trust-fixture" {
			t.Errorf("%s: folder %q", c.file, folder)
		}
	}
}

func TestScreenRows(t *testing.T) {
	rows := screenRows([]byte("ab\r\ncd\x1b[1A\x1b[2K\x1b[Gx\x1b[1B\r\x1b[3Ge\x1b]0;title\x07"))
	if len(rows) != 2 || rows[0] != "x" || rows[1] != "cde" {
		t.Errorf("rows %q", rows)
	}
}
