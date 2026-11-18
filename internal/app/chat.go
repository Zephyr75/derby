package app

// The chat view of a step (Settings > Appearance > Session view): its Claude
// Code conversation read from the transcript Claude Code writes, and the
// messages the user sends typed into the same session. The terminal and the
// chat show one session, so the user can switch between them at any time.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ChatItem is one entry of a step's conversation.
type ChatItem struct {
	Kind    string         `json:"kind"`    // user | command | text | thinking | tool | result | note | error
	ID      string         `json:"id"`      // the tool call a tool or result item belongs to
	Text    string         `json:"text"`    // the message; a result's output; a command's arguments
	Tool    string         `json:"tool"`    // a tool call's tool, a command's name
	Input   map[string]any `json:"input"`   // a tool call's input (long values cut)
	Diff    string         `json:"diff"`    // a result's change to a file, as a unified diff
	IsError bool           `json:"isError"` // a result that failed
}

// Chat is the part of a step's conversation after Offset.
type Chat struct {
	Items  []ChatItem `json:"items"`
	Offset int64      `json:"offset"` // where the next StepChat call starts
	Reset  bool       `json:"reset"`  // the transcript was replaced: drop the items shown so far
	Title  string     `json:"title"`  // Claude Code's title for the conversation, when it named it
	Found  bool       `json:"found"`  // Claude Code saved a transcript for this session
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
	path := transcriptPath(s.SessionID)
	if path == "" {
		return Chat{Items: []ChatItem{}, Offset: offset}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return Chat{Items: []ChatItem{}, Offset: offset}, nil
	}
	defer f.Close()
	c := Chat{Items: []ChatItem{}, Found: true}
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
	c.Items, c.Title = parseChat(b)
	c.Offset = offset + int64(len(b))
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

// transcriptLine is the part of a transcript line the chat shows.
type transcriptLine struct {
	Type        string          `json:"type"`
	Subtype     string          `json:"subtype"`
	IsMeta      bool            `json:"isMeta"`
	IsSidechain bool            `json:"isSidechain"`
	Content     string          `json:"content"` // system lines
	AITitle     string          `json:"aiTitle"`
	Error       json.RawMessage `json:"error"`
	Message     struct {
		Content json.RawMessage `json:"content"`
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
)

// parseChat turns transcript lines into chat items, and the conversation's
// latest title.
func parseChat(b []byte) ([]ChatItem, string) {
	items := []ChatItem{}
	title := ""
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
		case "user":
			if !l.IsMeta {
				items = append(items, userItems(l)...)
			}
		case "assistant":
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
					items = append(items, it)
				}
			}
		}
	}
	return items, title
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

// commandItem reads a /command the user ran, e.g. herd's agent commands.
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

// SendToStep types a message into a step's session, as if in its terminal.
// A session that is not running is resumed with the message.
func (a *App) SendToStep(stepID, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if sess, ok := a.run.Get(stepID); ok {
		// A bracketed paste keeps newlines in the message; Enter sends it.
		if _, err := sess.Write([]byte("\x1b[200~" + text + "\x1b[201~")); err != nil {
			return err
		}
		time.Sleep(80 * time.Millisecond)
		_, err := sess.Write([]byte("\r"))
		return err
	}
	if err := a.elsewhereErr(stepID); err != nil {
		return err
	}
	a.setPaused(stepID, false)
	return a.resumeWith(stepID, text)
}

// InterruptStep stops what Claude is doing, as Esc does in its terminal: an
// answer, or a permission prompt (declined).
func (a *App) InterruptStep(stepID string) error {
	sess, ok := a.run.Get(stepID)
	if !ok {
		return errors.New("the session is not running")
	}
	_, err := sess.Write([]byte("\x1b"))
	return err
}
