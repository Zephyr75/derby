package app

import (
	"strings"
	"testing"
)

func TestParseChat(t *testing.T) {
	lines := strings.Join([]string{
		`{"type":"ai-title","aiTitle":"Fix the unlock retry"}`,
		`{"type":"user","message":{"role":"user","content":"<command-message>jockey:implement</command-message>\n<command-name>/jockey:implement</command-name>\n<command-args>4356 -- keep it small</command-args>"}}`,
		`{"type":"user","isMeta":true,"message":{"role":"user","content":[{"type":"text","text":"You are the Jockey **Implement** agent."}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"","signature":"x"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Reading the plan."}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Edit","input":{"file_path":"/w/a.go","old_string":"a","new_string":"b"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"The file /w/a.go has been updated."}]},"timestamp":"2026-11-20T14:55:02.000Z","toolUseResult":{"filePath":"/w/a.go","structuredPatch":[{"oldStart":3,"oldLines":1,"newStart":3,"newLines":1,"lines":["-a","+b"]}]}}`,
		`{"type":"assistant","isSidechain":true,"message":{"role":"assistant","content":[{"type":"text","text":"a subagent"}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"[Request interrupted by user]"}]}}`,
		`{"type":"user","message":{"role":"user","content":"why b?"},"timestamp":"2026-11-20T14:56:40.000Z"}`,
		`{"type":"system","subtype":"api_error","error":{"message":"Connection error.","formatted":"Can't reach the API server"}}`,
		`{"type":"system","subtype":"stop_hook_summary"}`,
		`{"type":"attachment","attachment":{"type":"hook_success"}}`,
	}, "\n") + "\n"
	items, title, _ := parseChat([]byte(lines))
	if title != "Fix the unlock retry" {
		t.Errorf("title %q", title)
	}
	var kinds []string
	for _, it := range items {
		kinds = append(kinds, it.Kind)
	}
	if got := strings.Join(kinds, " "); got != "command text tool result note user error" {
		t.Fatalf("kinds: %s", got)
	}
	if c := items[0]; c.Tool != "/jockey:implement" || c.Text != "4356 -- keep it small" {
		t.Errorf("command: %+v", c)
	}
	if r := items[3]; r.ID != "t1" || r.Diff != "@@ -3,1 +3,1 @@\n-a\n+b\n" || r.At != "" {
		t.Errorf("result: %+v", r)
	}
	if u := items[5]; u.Text != "why b?" || u.At != "2026-11-20T14:56:40.000Z" {
		t.Errorf("user: %+v", u)
	}
	if e := items[6]; e.Text != "Can't reach the API server" {
		t.Errorf("error: %+v", e)
	}
}

func TestChatBashAndSkills(t *testing.T) {
	lines := strings.Join([]string{
		`{"type":"attachment","attachment":{"type":"skill_listing","names":["jockey","jockey:plan","archify"],"content":"- jockey:plan: Plan a task\n- archify: Draw diagrams"}}`,
		`{"type":"user","message":{"role":"user","content":"<bash-input>echo hi</bash-input>"}}`,
		`{"type":"user","message":{"role":"user","content":"<bash-stdout>hi</bash-stdout><bash-stderr></bash-stderr>"}}`,
	}, "\n") + "\n"
	items, _, meta := parseChat([]byte(lines))
	if len(items) != 2 || items[0].Kind != "bash" || items[0].Text != "echo hi" || items[1].Kind != "bashout" || items[1].Text != "hi" {
		t.Fatalf("bash items: %+v", items)
	}
	if len(meta.skills) != 3 || meta.skills[1].Description != "Plan a task" || meta.skills[0].Description != "" {
		t.Fatalf("skills: %+v", meta.skills)
	}
}

func TestMatchFiles(t *testing.T) {
	files := []string{"internal/unlock/ble.go", "internal/unlock/budget.go", "README.md", "cmd/ble/main.go"}
	got := matchFiles(files, "ble", 10)
	if len(got) == 0 || got[0] != "cmd/ble/" {
		t.Fatalf("ble: %v", got)
	}
	if got := matchFiles(files, "iub", 10); len(got) == 0 || got[0] != "internal/unlock/budget.go" && got[0] != "internal/unlock/ble.go" {
		t.Fatalf("subsequence: %v", got)
	}
}
