package hooks

import (
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasShellChild(t *testing.T) {
	ps := `      1 /sbin/init
   4115 /usr/bin/zsh -c source /home/u/.claude/shell-snapshots/snapshot-zsh-1.sh 2>/dev/null || true && eval 'sleep 20'
   4115 sh -c [ -r "$DERBY_ENDPOINT" ] && curl …
   4116 /usr/bin/node mcp-server.js`
	if !hasShellChild(ps, 4115) {
		t.Error("the background zsh of claude 4115 was not found")
	}
	if hasShellChild(ps, 4116) || hasShellChild(ps, 1) {
		t.Error("a process without a Bash tool shell was reported")
	}
}

// A hook that finds no derby (it is restarting) spools its event, and the
// next derby replays it.
func TestSpoolWhenNoDerby(t *testing.T) {
	home := t.TempDir()
	endpoint := filepath.Join(home, "endpoint")
	if err := os.WriteFile(endpoint, []byte("DERBY_URL=http://127.0.0.1:1\nDERBY_TOKEN=x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"UserPromptSubmit", "Stop"} {
		cmd := exec.Command("sh", "-c", Command(e))
		cmd.Env = append(os.Environ(), "DERBY_ENDPOINT="+endpoint, "DERBY_TASK=p:1", "DERBY_STEP=test-1")
		cmd.Stdin = strings.NewReader(`{"session_id":"s1"}`)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v %s", e, err, out)
		}
	}
	var got []string
	Replay(filepath.Join(home, "spool"), func(ev Event) {
		got = append(got, ev.Name+" "+ev.StepID+" "+ev.SessionID)
	})
	if strings.Join(got, ", ") != "UserPromptSubmit p:1/test-1 s1, Stop p:1/test-1 s1" {
		t.Errorf("replayed %q", got)
	}
	if es, _ := os.ReadDir(filepath.Join(home, "spool")); len(es) != 0 {
		t.Errorf("%d events left in the spool", len(es))
	}
}

// A hook that reaches derby spools nothing.
func TestHookPostsToDerby(t *testing.T) {
	got := make(chan Event, 1)
	srv := httptest.NewServer(&Receiver{Token: "tok", OnEvent: func(ev Event) { got <- ev }})
	defer srv.Close()
	home := t.TempDir()
	endpoint := filepath.Join(home, "endpoint")
	if err := os.WriteFile(endpoint, []byte("DERBY_URL="+srv.URL+"\nDERBY_TOKEN=tok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", Command("Stop"))
	cmd.Env = append(os.Environ(), "DERBY_ENDPOINT="+endpoint, "DERBY_TASK=p:1", "DERBY_STEP=test-1")
	cmd.Stdin = strings.NewReader(`{"session_id":"s1"}`)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if ev := <-got; ev.Name != "Stop" || ev.StepID != "p:1/test-1" || ev.SessionID != "s1" {
		t.Errorf("derby got %+v", ev)
	}
	if _, err := os.Stat(filepath.Join(home, "spool")); err == nil {
		t.Error("an event derby received was spooled too")
	}
}
