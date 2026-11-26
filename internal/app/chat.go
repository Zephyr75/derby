package app

// The chat view of a step (Settings > Appearance > Session view): its Claude
// Code conversation read from the transcript Claude Code writes, and the
// messages the user sends typed into the same session. The terminal and the
// chat show one session, so the user can switch between them at any time.

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"jockey/internal/hooks"
	"jockey/internal/runner"
	"jockey/internal/settings"
	"jockey/internal/store"
)

// ChatItem is one entry of a step's conversation.
type ChatItem struct {
	Kind    string         `json:"kind"`         // user | command | bash | bashout | text | thinking | tool | result | note | error
	ID      string         `json:"id"`           // the tool call a tool or result item belongs to
	Text    string         `json:"text"`         // the message; a result's output; a command's arguments
	Tool    string         `json:"tool"`         // a tool call's tool, a command's name
	Input   map[string]any `json:"input"`        // a tool call's input (long values cut)
	Diff    string         `json:"diff"`         // a result's change to a file, as a unified diff
	IsError bool           `json:"isError"`      // a result that failed
	At      string         `json:"at,omitempty"` // when the user sent it (user, command and bash items), RFC 3339
}

// Chat is the part of a step's conversation after Offset.
type Chat struct {
	Items  []ChatItem `json:"items"`
	Offset int64      `json:"offset"` // where the next StepChat call starts
	Reset  bool       `json:"reset"`  // the transcript was replaced: drop the items shown so far
	Title  string     `json:"title"`  // Claude Code's title for the conversation, when it named it
	Found  bool       `json:"found"`  // Claude Code saved a transcript for this session
	// Status: the session's model, effort and context, from its statusline,
	// else from the transcript read so far (Live: the statusline's).
	Status SessionStatus `json:"status"`
	Live   bool          `json:"live"`
	// Skills: the session's skills and plugin commands (/jockey:implement…),
	// when the transcript read listed them.
	Skills []Skill `json:"skills"`
	// Trust: the folder Claude Code asks to trust before it starts (a new
	// worktree), while it asks.
	Trust string `json:"trust"`
}

// Skill is a skill or command the session offers after a "/".
type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

const (
	chatTextMax   = 8000 // a message, in bytes
	chatResultMax = 3000 // a tool's output
	chatInputMax  = 2000 // each string in a tool's input
	chatDiffMax   = 400  // lines of a file change
)

// StepChat reads a step's conversation from byte offset on; the chat view
// calls it again with the returned Offset to follow the session.
func (a *App) StepChat(stepID string, offset int64) (Chat, error) {
	s, err := a.db.Step(stepID)
	if err != nil {
		return Chat{}, err
	}
	trust := ""
	if folder, _, ok := a.stepTrust(stepID); ok {
		trust = cmp.Or(folder, "this folder")
	}
	path := transcriptPath(s.SessionID)
	if path == "" {
		return Chat{Items: []ChatItem{}, Offset: offset, Trust: trust}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return Chat{Items: []ChatItem{}, Offset: offset, Trust: trust}, nil
	}
	defer f.Close()
	c := Chat{Items: []ChatItem{}, Found: true, Trust: trust}
	if fi, err := f.Stat(); err == nil && fi.Size() < offset {
		offset, c.Reset = 0, true // rewritten, e.g. after a compaction
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return Chat{}, err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return Chat{}, err
	}
	// Only whole lines: Claude Code may be writing the last one.
	if i := bytes.LastIndexByte(b, '\n'); i >= 0 {
		b = b[:i+1]
	} else {
		b = nil
	}
	var meta chatMeta
	c.Items, c.Title, meta = parseChat(b)
	c.Skills = meta.skills
	c.Offset = offset + int64(len(b))
	st, ok := a.stepStatus(stepID)
	if !ok {
		st.Model, st.Effort, st.ContextUsed = meta.model, meta.effort, meta.context
	}
	c.Status, c.Live = st, ok
	return c, nil
}

// transcriptPath finds a session's transcript in Claude Code's projects
// directory (one folder per working directory).
func transcriptPath(session string) string {
	if session == "" {
		return ""
	}
	m, _ := filepath.Glob(filepath.Join(claudeDir(), "projects", "*", session+".jsonl"))
	if len(m) == 0 {
		return ""
	}
	return m[0]
}

// chatMeta is what the transcript read says of the session, for when no
// statusline reported it: the model and effort of its last request, and the
// tokens in its context after the last answer.
type chatMeta struct {
	model, effort string
	context       int
	skills        []Skill
}

// transcriptLine is the part of a transcript line the chat shows.
type transcriptLine struct {
	Type        string `json:"type"`
	Timestamp   string `json:"timestamp"`
	QuerySource string `json:"querySource"` // api-request: repl_main_thread for the conversation's own
	Params      struct {
		Model  string `json:"model"`
		Output struct {
			Effort string `json:"effort"`
		} `json:"output_config"`
	} `json:"params"`
	Subtype     string `json:"subtype"`
	IsMeta      bool   `json:"isMeta"`
	IsSidechain bool   `json:"isSidechain"`
	Content     string `json:"content"` // system lines
	AITitle     string `json:"aiTitle"`
	Attachment  struct {
		Type    string   `json:"type"`
		Content string   `json:"content"`
		Names   []string `json:"names"`
	} `json:"attachment"`
	Error   json.RawMessage `json:"error"`
	Message struct {
		Content json.RawMessage `json:"content"`
		Usage   *struct {
			Input       int `json:"input_tokens"`
			CacheCreate int `json:"cache_creation_input_tokens"`
			CacheRead   int `json:"cache_read_input_tokens"`
			Output      int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
	ToolUseResult json.RawMessage `json:"toolUseResult"`
}

type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     map[string]any  `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

var (
	cmdNameTag = regexp.MustCompile(`<command-name>([^<]*)</command-name>`)
	cmdArgsTag = regexp.MustCompile(`(?s)<command-args>(.*?)</command-args>`)
	cmdOutTag  = regexp.MustCompile(`(?s)<local-command-std(?:out|err)>(.*?)</local-command-std(?:out|err)>`)
	bashIn     = regexp.MustCompile(`(?s)^<bash-input>(.*)</bash-input>$`)
	bashOut    = regexp.MustCompile(`(?s)<bash-std(?:out|err)>(.*?)</bash-std(?:out|err)>`)
)

// parseChat turns transcript lines into chat items, and the conversation's
// latest title and meta.
func parseChat(b []byte) ([]ChatItem, string, chatMeta) {
	items := []ChatItem{}
	title := ""
	var meta chatMeta
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for sc.Scan() {
		var l transcriptLine
		if json.Unmarshal(sc.Bytes(), &l) != nil || l.IsSidechain {
			continue
		}
		switch l.Type {
		case "ai-title":
			title = l.AITitle
		case "attachment":
			if l.Attachment.Type == "skill_listing" && len(l.Attachment.Names) > 0 {
				meta.skills = skillsOf(l.Attachment.Names, l.Attachment.Content)
			}
		case "api-request":
			if l.QuerySource == "repl_main_thread" {
				meta.model, meta.effort = orStr(l.Params.Model, meta.model), orStr(l.Params.Output.Effort, meta.effort)
			}
		case "user":
			if !l.IsMeta {
				items = append(items, sent(userItems(l), l.Timestamp)...)
			}
		case "assistant":
			if u := l.Message.Usage; u != nil {
				meta.context = u.Input + u.CacheCreate + u.CacheRead + u.Output
			}
			var blocks []contentBlock
			if json.Unmarshal(l.Message.Content, &blocks) != nil {
				continue
			}
			for _, bl := range blocks {
				switch bl.Type {
				case "text":
					if t := strings.TrimSpace(bl.Text); t != "" {
						items = append(items, ChatItem{Kind: "text", Text: cut(t, chatTextMax)})
					}
				case "thinking":
					if t := strings.TrimSpace(bl.Thinking); t != "" {
						items = append(items, ChatItem{Kind: "thinking", Text: cut(t, chatTextMax)})
					}
				case "tool_use":
					items = append(items, ChatItem{Kind: "tool", ID: bl.ID, Tool: bl.Name, Input: cutInput(bl.Input)})
				}
			}
		case "system":
			switch l.Subtype {
			case "api_error":
				var e struct{ Message, Formatted string }
				_ = json.Unmarshal(l.Error, &e)
				items = append(items, ChatItem{Kind: "error", Text: orStr(e.Formatted, orStr(e.Message, "API error"))})
			case "informational":
				items = append(items, ChatItem{Kind: "note", Text: cut(strings.TrimSpace(l.Content), chatTextMax)})
			case "compact_boundary":
				items = append(items, ChatItem{Kind: "note", Text: "Conversation compacted"})
			case "local_command":
				if it, ok := commandItem(l.Content); ok {
					it.At = l.Timestamp
					items = append(items, it)
				}
			}
		}
	}
	return items, title, meta
}

// sent stamps what the user sent with its transcript line's time.
func sent(items []ChatItem, at string) []ChatItem {
	for i, it := range items {
		if it.Kind == "user" || it.Kind == "command" || it.Kind == "bash" {
			items[i].At = at
		}
	}
	return items
}

// userItems: what the user typed (a message or a /command), a local
// command's output, or tool results.
func userItems(l transcriptLine) []ChatItem {
	var s string
	if json.Unmarshal(l.Message.Content, &s) == nil {
		return userText(s)
	}
	var blocks []contentBlock
	if json.Unmarshal(l.Message.Content, &blocks) != nil {
		return nil
	}
	var out []ChatItem
	for _, bl := range blocks {
		switch bl.Type {
		case "text":
			out = append(out, userText(bl.Text)...)
		case "image":
			out = append(out, ChatItem{Kind: "user", Text: "[image]"})
		case "tool_result":
			it := ChatItem{Kind: "result", ID: bl.ToolUseID, IsError: bl.IsError, Text: cut(resultText(bl.Content), chatResultMax)}
			it.Diff = resultDiff(l.ToolUseResult)
			out = append(out, it)
		}
	}
	return out
}

func userText(s string) []ChatItem {
	s = strings.TrimSpace(s)
	switch {
	case s == "", strings.HasPrefix(s, "<local-command-caveat>"), strings.HasPrefix(s, "<system-reminder>"):
		return nil
	case strings.HasPrefix(s, "[Request interrupted by user"):
		return []ChatItem{{Kind: "note", Text: "Interrupted"}}
	}
	// Bash mode (a message starting with "!"): the command, then its output.
	if m := bashIn.FindStringSubmatch(s); m != nil {
		return []ChatItem{{Kind: "bash", Text: strings.TrimSpace(m[1])}}
	}
	if strings.HasPrefix(s, "<bash-std") {
		var out []string
		for _, m := range bashOut.FindAllStringSubmatch(s, -1) {
			if t := strings.TrimRight(html.UnescapeString(m[1]), "\n"); t != "" { // Claude Code escapes the output
				out = append(out, t)
			}
		}
		return []ChatItem{{Kind: "bashout", Text: cut(strings.Join(out, "\n"), chatResultMax)}}
	}
	if m := cmdOutTag.FindStringSubmatch(s); m != nil {
		if t := strings.TrimSpace(m[1]); t != "" {
			return []ChatItem{{Kind: "note", Text: cut(t, chatTextMax)}}
		}
		return nil
	}
	if it, ok := commandItem(s); ok {
		return []ChatItem{it}
	}
	return []ChatItem{{Kind: "user", Text: cut(s, chatTextMax)}}
}

// commandItem reads a /command the user ran, e.g. jockey's agent commands.
func commandItem(s string) (ChatItem, bool) {
	m := cmdNameTag.FindStringSubmatch(s)
	if m == nil {
		return ChatItem{}, false
	}
	it := ChatItem{Kind: "command", Tool: strings.TrimSpace(m[1])}
	if a := cmdArgsTag.FindStringSubmatch(s); a != nil {
		it.Text = cut(strings.TrimSpace(a[1]), chatTextMax)
	}
	return it, true
}

// resultText is a tool result's text: a string, or text blocks.
func resultText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimRight(s, "\n")
	}
	var blocks []contentBlock
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	var parts []string
	for _, bl := range blocks {
		switch bl.Type {
		case "text":
			parts = append(parts, bl.Text)
		case "image":
			parts = append(parts, "[image]")
		}
	}
	return strings.TrimRight(strings.Join(parts, "\n"), "\n")
}

// resultDiff is the change an Edit or Write made, from Claude Code's record
// of it: its patch hunks, or a new file's content.
func resultDiff(raw json.RawMessage) string {
	var r struct {
		Type     string `json:"type"`
		Content  string `json:"content"`
		FilePath string `json:"filePath"`
		Patch    []struct {
			OldStart, OldLines, NewStart, NewLines int
			Lines                                  []string
		} `json:"structuredPatch"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &r) != nil || r.FilePath == "" {
		return ""
	}
	var b strings.Builder
	n := 0
	line := func(s string) bool {
		if n == chatDiffMax {
			return false
		}
		b.WriteString(s + "\n")
		n++
		return true
	}
	if len(r.Patch) == 0 && r.Type == "create" {
		lines := strings.Split(strings.TrimSuffix(r.Content, "\n"), "\n")
		fmt.Fprintf(&b, "@@ -0,0 +1,%d @@\n", len(lines))
		for _, l := range lines {
			if !line("+" + l) {
				break
			}
		}
		return b.String()
	}
	for _, h := range r.Patch {
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", h.OldStart, h.OldLines, h.NewStart, h.NewLines)
		for _, l := range h.Lines {
			if !line(l) {
				return b.String()
			}
		}
	}
	return b.String()
}

func cut(s string, max int) string {
	if len(s) <= max {
		return s
	}
	i := max
	for i > 0 && !utf8Start(s[i]) {
		i--
	}
	return s[:i] + "\n…"
}

func utf8Start(c byte) bool { return c&0xC0 != 0x80 }

// cutInput shortens a tool input's long strings (a Write's content…).
func cutInput(in map[string]any) map[string]any {
	for k, v := range in {
		if s, ok := v.(string); ok {
			in[k] = cut(s, chatInputMax)
		}
	}
	return in
}

// SideQuestion answers a /btw question: a fork of the step's session, so it
// knows the conversation so far, with no tools and nothing saved. The step's
// own session is left as it is.
func (a *App) SideQuestion(stepID, question string) (string, error) {
	s, err := a.db.Step(stepID)
	if err != nil {
		return "", err
	}
	t, err := a.db.Task(s.TaskID)
	if err != nil {
		return "", err
	}
	if s.SessionID == "" {
		return "", errors.New("the session has not started yet")
	}
	args := []string{"--resume", s.SessionID, "--fork-session", "--no-session-persistence", "--tools", ""}
	if s.Model != "" {
		args = append(args, "--model", s.Model)
	}
	prompt := "A side question, asked while you work: answer it briefly from the conversation so far, with no tools.\n\n" + strings.TrimSpace(question)
	out, err := a.claudePrint(worktreePath(a.projectOf(t), t), 5*time.Minute, append(args, "--", prompt)...)
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return "", errors.New(strings.TrimSpace(string(ee.Stderr)))
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// SendToStep types a message into a step's session, as if in its terminal.
// A session that is not running is resumed with the message.
func (a *App) SendToStep(stepID, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if sess, ok := a.run.Get(stepID); ok {
		return typeMessage(sess, text)
	}
	if err := a.elsewhereErr(stepID); err != nil {
		return err
	}
	a.setPaused(stepID, false)
	if !strings.HasPrefix(text, "!") {
		return a.resumeWith(stepID, text)
	}
	// "!cmd" runs in Claude Code's bash mode only when typed: as the resume
	// prompt it would go to the model. So resume, then type it once started.
	started := a.awaitStart(stepID)
	if err := a.resumeWith(stepID, ""); err != nil {
		a.dropStart(stepID, started)
		return err
	}
	go func() {
		fail := func(why string) { a.toast("Could not run "+cut(text, 60), why, true) }
		select {
		case <-started:
		case <-time.After(time.Minute):
			a.dropStart(stepID, started)
			fail("Claude Code did not start within a minute. Is it waiting in the terminal?")
			return
		case <-a.ctx.Done():
			return
		}
		time.Sleep(time.Second) // the prompt draws after the hook
		sess, ok := a.run.Get(stepID)
		if !ok {
			fail("The session ended before the command could be typed.")
			return
		}
		if err := typeMessage(sess, text); err != nil {
			fail(err.Error())
		}
	}()
	return nil
}

// typeMessage types a message into a session's prompt and sends it. A
// bracketed paste keeps newlines in the message; Enter sends it.
func typeMessage(sess *runner.Session, text string) error {
	if _, err := sess.Write([]byte("\x1b[200~" + text + "\x1b[201~")); err != nil {
		return err
	}
	time.Sleep(80 * time.Millisecond)
	_, err := sess.Write([]byte("\r"))
	return err
}

// awaitStart returns a channel closed when the step's session next starts
// (its SessionStart hook).
func (a *App) awaitStart(stepID string) chan struct{} {
	a.startMu.Lock()
	defer a.startMu.Unlock()
	if a.starts == nil {
		a.starts = map[string]chan struct{}{}
	}
	if ch, ok := a.starts[stepID]; ok {
		return ch
	}
	ch := make(chan struct{})
	a.starts[stepID] = ch
	return ch
}

func (a *App) dropStart(stepID string, ch chan struct{}) {
	a.startMu.Lock()
	defer a.startMu.Unlock()
	if a.starts[stepID] == ch {
		delete(a.starts, stepID)
	}
}

// sessionStarted wakes what waits on the step's session start.
func (a *App) sessionStarted(stepID string) {
	a.startMu.Lock()
	defer a.startMu.Unlock()
	if ch, ok := a.starts[stepID]; ok {
		close(ch)
		delete(a.starts, stepID)
	}
}

// InterruptStep stops what Claude is doing, as Esc does in its terminal: an
// answer, or a permission prompt (declined).
func (a *App) InterruptStep(stepID string) error {
	sess, ok := a.run.Get(stepID)
	if !ok {
		return errors.New("the session is not running")
	}
	if _, err := sess.Write([]byte("\x1b")); err != nil {
		return err
	}
	// Claude Code sends no Stop hook for an interrupt: it now waits for the user.
	s, changed := a.updateStep(stepID, func(s *store.Step) bool {
		if s.State != store.Running && s.State != store.Approval {
			return false
		}
		s.State = store.Idle
		return true
	})
	if changed {
		a.emitTask(s.TaskID)
		a.emit(EvStep, map[string]string{"taskId": s.TaskID, "stepId": s.ID, "state": s.State})
	}
	return nil
}

// SessionStatus is what a session's statusline tells: its model, effort and
// context window (Claude Code hands it to statuslines only; see
// hooks.StatusCommand), else what its transcript says.
type SessionStatus struct {
	Model       string    `json:"model"`       // e.g. claude-opus-5-5
	ModelName   string    `json:"modelName"`   // e.g. Opus 5.5
	Effort      string    `json:"effort"`      // low … max
	ContextUsed int       `json:"contextUsed"` // tokens in the context window
	ContextSize int       `json:"contextSize"` // the window's size; 0 when unknown
	Plan        PlanUsage `json:"plan"`        // the account's plan usage
}

// PlanUsage is the share of the plan's 5-hour and weekly limits used.
type PlanUsage struct {
	FiveHour      float64 `json:"fiveHour"` // percent; -1 when unknown
	FiveHourReset int64   `json:"fiveHourReset"`
	Week          float64 `json:"week"`
	WeekReset     int64   `json:"weekReset"`
	At            int64   `json:"at"` // when a statusline last reported it (unix seconds); 0: never
}

// setStatus keeps a statusline's report for the chat view.
func (a *App) setStatus(ev hooks.Event) {
	var in struct {
		Model struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"model"`
		ContextWindow struct {
			Size  int `json:"context_window_size"`
			Usage *struct {
				Input       int `json:"input_tokens"`
				CacheCreate int `json:"cache_creation_input_tokens"`
				CacheRead   int `json:"cache_read_input_tokens"`
			} `json:"current_usage"`
		} `json:"context_window"`
		RateLimits *struct {
			FiveHour *struct {
				Used   *float64 `json:"used_percentage"`
				Resets int64    `json:"resets_at"`
			} `json:"five_hour"`
			SevenDay *struct {
				Used   *float64 `json:"used_percentage"`
				Resets int64    `json:"resets_at"`
			} `json:"seven_day"`
		} `json:"rate_limits"`
		Effort json.RawMessage `json:"effort"`
	}
	if json.Unmarshal(ev.Raw, &in) != nil {
		return
	}
	st := SessionStatus{Model: in.Model.ID, ModelName: in.Model.DisplayName, ContextSize: in.ContextWindow.Size, Effort: effortOf(in.Effort)}
	if u := in.ContextWindow.Usage; u != nil {
		st.ContextUsed = u.Input + u.CacheCreate + u.CacheRead
	}
	a.statusMu.Lock()
	defer a.statusMu.Unlock()
	if r := in.RateLimits; r != nil {
		p := PlanUsage{FiveHour: -1, Week: -1, At: time.Now().Unix()}
		if r.FiveHour != nil && r.FiveHour.Used != nil {
			p.FiveHour, p.FiveHourReset = *r.FiveHour.Used, r.FiveHour.Resets
		}
		if r.SevenDay != nil && r.SevenDay.Used != nil {
			p.Week, p.WeekReset = *r.SevenDay.Used, r.SevenDay.Resets
		}
		a.planUse = p
	}
	a.status[ev.StepID] = st
}

// effortOf reads the statusline's effort: "high", or {"level": "high"}.
func effortOf(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var o struct{ Level string }
	_ = json.Unmarshal(raw, &o)
	return o.Level
}

// stepStatus is a step's latest statusline report, with the account's plan
// usage; ok is false when none came since jockey started.
func (a *App) stepStatus(stepID string) (SessionStatus, bool) {
	a.statusMu.Lock()
	defer a.statusMu.Unlock()
	st, ok := a.status[stepID]
	st.Plan = a.planUse
	if st.Plan.At == 0 {
		st.Plan.FiveHour, st.Plan.Week = -1, -1
	}
	return st, ok
}

// SetStepModel changes the model and effort of a step's session (empty:
// unchanged). Claude Code's /model and /effort would also change the user's
// default for every new session, so jockey restarts the session with --model
// and --effort instead, while Claude waits for the user.
func (a *App) SetStepModel(stepID, model, effort string) error {
	if effort != "" && !slices.Contains(settings.Efforts, effort) {
		return fmt.Errorf("unknown effort %q", effort)
	}
	s, err := a.db.Step(stepID)
	if err != nil {
		return err
	}
	sess, live := a.run.Get(stepID)
	if live && (s.State == store.Running || s.State == store.Approval) {
		return errors.New("Claude is working: change the model or effort once it waits for you")
	}
	s, _ = a.updateStep(stepID, func(x *store.Step) bool {
		if model != "" {
			x.ModelSet, x.Model = model, model
		}
		if effort != "" {
			x.EffortSet = effort
		}
		return true
	})
	if !live {
		a.emitTask(s.TaskID)
		return nil
	}
	a.run.Stop(stepID)
	select {
	case <-sess.Done():
	case <-time.After(5 * time.Second):
		return errors.New("Claude Code did not stop: try again")
	}
	return a.resumeWith(stepID, "")
}

// AttachToStep copies files to the step's run directory (attachments/) and
// returns their new paths, for the user's message to mention.
func (a *App) AttachToStep(stepID string, files []string) ([]string, error) {
	s, err := a.db.Step(stepID)
	if err != nil {
		return nil, err
	}
	t, err := a.db.Task(s.TaskID)
	if err != nil {
		return nil, err
	}
	return copyAttachments(a.runDir(a.projectOf(t), t), files)
}

// skillsOf reads Claude Code's skill listing ("- name: description" lines)
// for the names it lists.
func skillsOf(names []string, content string) []Skill {
	desc := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimPrefix(line, "- ")
		best := "" // the longest name the line starts with: "jockey:plan", not "jockey"
		for _, n := range names {
			if strings.HasPrefix(line, n+": ") && len(n) > len(best) {
				best = n
			}
		}
		if best != "" {
			desc[best] = strings.TrimPrefix(line, best+": ")
		}
	}
	out := make([]Skill, 0, len(names))
	for _, n := range names {
		out = append(out, Skill{Name: n, Description: cut(desc[n], 200)})
	}
	return out
}

// StepFiles lists the files and folders of a step's worktree that match
// query, best first, for the chat's "@" mentions (paths relative to the
// worktree, folders ending in "/").
func (a *App) StepFiles(stepID, query string) ([]string, error) {
	s, err := a.db.Step(stepID)
	if err != nil {
		return nil, err
	}
	t, err := a.db.Task(s.TaskID)
	if err != nil {
		return nil, err
	}
	wt := worktreePath(a.projectOf(t), t)
	out, err := a.git.Run(a.ctx, wt, "ls-files", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return matchFiles(strings.Split(strings.TrimSpace(out), "\n"), query, 30), nil
}

// matchFiles ranks paths (and their folders) against query: a match in the
// file name first, then in the path, then the letters in order.
func matchFiles(files []string, query string, max int) []string {
	seen := map[string]bool{}
	var all []string
	for _, f := range files {
		if f == "" {
			continue
		}
		for d := filepath.Dir(f); d != "." && !seen[d+"/"]; d = filepath.Dir(d) {
			seen[d+"/"] = true
			all = append(all, d+"/")
		}
		all = append(all, f)
	}
	q := strings.ToLower(query)
	type hit struct {
		path  string
		score int
	}
	var hits []hit
	for _, p := range all {
		lp := strings.ToLower(p)
		name := strings.ToLower(filepath.Base(strings.TrimSuffix(p, "/")))
		score := -1
		switch {
		case q == "":
			score = 0
		case strings.HasPrefix(name, q):
			score = 4
		case strings.Contains(name, q):
			score = 3
		case strings.Contains(lp, q):
			score = 2
		case subsequence(lp, q):
			score = 1
		}
		if score >= 0 {
			hits = append(hits, hit{p, score})
		}
	}
	slices.SortStableFunc(hits, func(x, y hit) int {
		if x.score != y.score {
			return y.score - x.score
		}
		return len(x.path) - len(y.path)
	})
	out := []string{}
	for _, h := range hits[:min(len(hits), max)] {
		out = append(out, h.path)
	}
	return out
}

func subsequence(s, q string) bool {
	i := 0
	for j := 0; j < len(s) && i < len(q); j++ {
		if s[j] == q[i] {
			i++
		}
	}
	return i == len(q)
}

// QuestionAnswer is the user's answer to one question of Claude Code's
// question form (AskUserQuestion), from the chat view.
type QuestionAnswer struct {
	Multi bool   `json:"multi"` // the question takes several options
	Count int    `json:"count"` // its options
	Picks []int  `json:"picks"` // the options picked, from 1
	Other string `json:"other"` // a typed answer instead (one-option questions)
}

// AnswerQuestions answers the question form Claude Code shows, by typing
// what its terminal form expects: an option's number picks it (and moves on
// to the next question; on a several-option question it ticks it, and Tab
// moves on), the number after the options opens a text answer, and Enter on
// the last screen submits the answers.
func (a *App) AnswerQuestions(stepID string, answers []QuestionAnswer) error {
	sess, ok := a.run.Get(stepID)
	if !ok {
		return errors.New("the session is not running")
	}
	var keys []string
	for i, q := range answers {
		digit := func(n int) error {
			if n < 1 || n > q.Count+1 || n > 9 {
				return fmt.Errorf("question %d: no option %d", i+1, n)
			}
			keys = append(keys, fmt.Sprint(n))
			return nil
		}
		switch {
		case !q.Multi && strings.TrimSpace(q.Other) != "":
			if err := digit(q.Count + 1); err != nil {
				return err
			}
			keys = append(keys, "\x1b[200~"+strings.TrimSpace(q.Other)+"\x1b[201~", "\r")
		case !q.Multi && len(q.Picks) == 1:
			if err := digit(q.Picks[0]); err != nil {
				return err
			}
		case q.Multi && len(q.Picks) > 0:
			for _, p := range q.Picks {
				if p > q.Count {
					return fmt.Errorf("question %d: no option %d", i+1, p)
				}
				if err := digit(p); err != nil {
					return err
				}
			}
			keys = append(keys, "\t")
		default:
			return fmt.Errorf("question %d has no answer", i+1)
		}
	}
	keys = append(keys, "\r") // Submit answers
	for _, k := range keys {
		if _, err := sess.Write([]byte(k)); err != nil {
			return err
		}
		time.Sleep(250 * time.Millisecond) // the form redraws between keys
	}
	return nil
}
