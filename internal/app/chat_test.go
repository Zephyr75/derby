package app

import (
	"strings"
	"testing"
)

func TestParseChat(t *testing.T) {
	lines := strings.Join([]string{
		`{"type":"ai-title","aiTitle":"Fix the unlock retry"}`,
		`{"type":"user","message":{"role":"user","content":"<command-message>herd:implement</command-message>\n<command-name>/herd:implement</command-name>\n<command-args>4356 -- keep it small</command-args>"}}`,
		`{"type":"user","isMeta":true,"message":{"role":"user","content":[{"type":"text","text":"You are the Herd **Implement** agent."}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"","signature":"x"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Reading the plan."}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Edit","input":{"file_path":"/w/a.go","old_string":"a","new_string":"b"}}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"The file /w/a.go has been updated."}]},"toolUseResult":{"filePath":"/w/a.go","structuredPatch":[{"oldStart":3,"oldLines":1,"newStart":3,"newLines":1,"lines":["-a","+b"]}]}}`,
		`{"type":"assistant","isSidechain":true,"message":{"role":"assistant","content":[{"type":"text","text":"a subagent"}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"[Request interrupted by user]"}]}}`,
		`{"type":"user","message":{"role":"user","content":"why b?"}}`,
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
	if c := items[0]; c.Tool != "/herd:implement" || c.Text != "4356 -- keep it small" {
		t.Errorf("command: %+v", c)
	}
	if r := items[3]; r.ID != "t1" || r.Diff != "@@ -3,1 +3,1 @@\n-a\n+b\n" {
		t.Errorf("result: %+v", r)
	}
	if e := items[6]; e.Text != "Can't reach the API server" {
		t.Errorf("error: %+v", e)
	}
}
