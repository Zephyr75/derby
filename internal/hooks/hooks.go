// Package hooks receives Claude Code hook events (POST /hook) and writes the
// hooks settings file that makes every derby session post them.
package hooks

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Event is one hook call, decoded from Claude Code's JSON payload.
type Event struct {
	Name             string // SessionStart, UserPromptSubmit, Notification, PreToolUse, PostToolUse, Stop
	StepID           string // <task id>/<step key>
	PID              int    // the claude process that ran the hook (0 if unknown)
	SessionID        string `json:"session_id"`
	Source           string `json:"source"` // SessionStart: startup | resume | clear | compact
	Message          string `json:"message"`
	NotificationType string `json:"notification_type"`
	ToolName         string `json:"tool_name"` // PreToolUse, PostToolUse
	Raw              json.RawMessage
}

// IsPermission reports whether a Notification asks for permission.
func (e Event) IsPermission() bool {
	if e.NotificationType != "" {
		return e.NotificationType == "permission_prompt"
	}
	return strings.Contains(strings.ToLower(e.Message), "permission")
}

// Receiver handles POST /hook.
type Receiver struct {
	Token   string
	OnEvent func(Event)
}

func (h *Receiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Derby-Token")), []byte(h.Token)) != 1 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	ev, ok := decode(r.URL.Query().Get("event"), r.Header.Get("X-Derby-Step"), r.Header.Get("X-Derby-Pid"), body)
	if !ok {
		http.Error(w, "missing event or step", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
	if h.OnEvent != nil {
		go h.OnEvent(ev)
	}
}

func decode(name, step, pid string, body []byte) (Event, bool) {
	ev := Event{Name: name, StepID: step, Raw: body}
	ev.PID, _ = strconv.Atoi(pid)
	_ = json.Unmarshal(body, &ev)
	return ev, ev.Name != "" && ev.StepID != "" && ev.StepID != "/"
}

// Replay hands on the events hooks spooled in dir while no derby answered,
// oldest first, and removes them. A spooled event is its name, step and pid
// on a line each, then its JSON payload.
func Replay(dir string, on func(Event)) {
	es, _ := os.ReadDir(dir) // sorted by name: the time it was spooled
	for _, e := range es {
		if strings.HasPrefix(e.Name(), ".") {
			continue // still being written
		}
		p := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(p)
		_ = os.Remove(p)
		f := strings.SplitN(string(b), "\n", 4)
		if err != nil || len(f) < 4 {
			continue
		}
		if ev, ok := decode(f[0], f[1], f[2], []byte(f[3])); ok {
			on(ev)
		}
	}
}

// Events derby listens to.
var Events = []string{"SessionStart", "UserPromptSubmit", "Notification", "PreToolUse", "PostToolUse", "Stop"}

// AskTool is Claude Code's question form: while it is open, the step waits
// for the user. PreToolUse is only sent for it.
const AskTool = "AskUserQuestion"

// Command is the shell command a hook runs: it does nothing outside derby
// sessions and never fails the hook. $DERBY_ENDPOINT holds the current derby's
// address and token, so sessions that outlived a derby restart still report.
// When no derby answers (it is restarting), the event waits in spool/ next to
// it for the next derby to Replay.
func Command(event string) string {
	return `[ -n "$DERBY_TASK" ] && [ -n "$DERBY_ENDPOINT" ] || exit 0; b=$(cat); [ -r "$DERBY_ENDPOINT" ] && . "$DERBY_ENDPOINT"; ` +
		`printf '%s' "$b" | curl -sf -m 3 -X POST "$DERBY_URL/hook?event=` + event + `" -H "X-Derby-Token: $DERBY_TOKEN" -H "X-Derby-Step: $DERBY_TASK/$DERBY_STEP" -H "X-Derby-Pid: $PPID" -H "Content-Type: application/json" --data-binary @- >/dev/null 2>&1 || ` +
		`{ d="${DERBY_ENDPOINT%/*}/spool"; mkdir -p "$d" && printf '%s\n%s\n%s\n%s' ` + event + ` "$DERBY_TASK/$DERBY_STEP" "$PPID" "$b" > "$d/.$$" && mv "$d/.$$" "$d/$(date +%s%N)-$$"; }; exit 0`
}

// StatusCommand is the statusline every derby session runs: it hands Claude
// Code's status (context window, plan usage, model, effort) to derby, in the
// background so the line never waits, then runs the user's own statusline
// (user, from their Claude Code settings), so the terminal looks the same.
func StatusCommand(user string) string {
	post := `b=$(cat); if [ -n "$DERBY_TASK" ] && [ -r "$DERBY_ENDPOINT" ]; then . "$DERBY_ENDPOINT"; ` +
		`( printf '%s' "$b" | curl -sf -m 2 -X POST "$DERBY_URL/hook?event=` + StatusEvent + `" -H "X-Derby-Token: $DERBY_TOKEN" -H "X-Derby-Step: $DERBY_TASK/$DERBY_STEP" -H "Content-Type: application/json" --data-binary @- ) >/dev/null 2>&1 & fi`
	if strings.TrimSpace(user) == "" {
		return post
	}
	return post + `; printf '%s' "$b" | sh -c '` + strings.ReplaceAll(user, `'`, `'\''`) + `'`
}

// StatusEvent is the event name StatusCommand posts.
const StatusEvent = "StatusLine"

// StatusLine is a Claude Code statusLine setting.
type StatusLine struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Padding *int   `json:"padding,omitempty"`
}

// Settings returns the Claude Code settings every derby session gets: derby's
// hooks, its statusline (around the user's, if any), and claudeMdExcludes
// (CLAUDE.md files the sessions must not load).
func Settings(excludes []string, user StatusLine) ([]byte, error) {
	type hook struct {
		Type    string `json:"type"`
		Command string `json:"command"`
	}
	type matcher struct {
		Matcher string `json:"matcher,omitempty"`
		Hooks   []hook `json:"hooks"`
	}
	all := map[string][]matcher{}
	for _, e := range Events {
		m := matcher{Hooks: []hook{{Type: "command", Command: Command(e)}}}
		if e == "PreToolUse" {
			m.Matcher = AskTool
		}
		all[e] = []matcher{m}
	}
	status := StatusLine{Type: "command", Command: StatusCommand(user.Command), Padding: user.Padding}
	return json.MarshalIndent(map[string]any{"hooks": all, "claudeMdExcludes": excludes, "statusLine": status}, "", "  ")
}

// WriteSettings writes the settings file (~/.derby/hooks.json).
func WriteSettings(path string, excludes []string, user StatusLine) error {
	b, err := Settings(excludes, user)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// BackgroundShell reports whether the claude process pid still runs a shell
// from its Bash tool (a background command or Monitor): its turn ended, but
// it is waiting on that shell, not on the user. Bash tool shells are direct
// children of claude that source a Claude Code shell snapshot.
func BackgroundShell(pid int) bool {
	if pid <= 0 {
		return false
	}
	out, err := exec.Command("ps", "-A", "-o", "ppid=,args=").Output()
	if err != nil {
		return false
	}
	return hasShellChild(string(out), pid)
}

func hasShellChild(ps string, pid int) bool {
	for _, line := range strings.Split(ps, "\n") {
		ppid, args, ok := strings.Cut(strings.TrimSpace(line), " ")
		if ok && ppid == strconv.Itoa(pid) && strings.Contains(args, "/shell-snapshots/") {
			return true
		}
	}
	return false
}
