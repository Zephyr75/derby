package settings

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"derby/defaults"
)

func TestCheckTriggers(t *testing.T) {
	ok := []Agent{{ID: "plan", Name: "Plan", Trigger: "manual"}, {ID: "impl", Name: "Implement", Trigger: "after:plan"}, {ID: "review", Name: "Review", Trigger: "after:plan"}, {ID: "test", Name: "Test", Trigger: "after:review"}}
	if err := checkTriggers(ok); err != nil {
		t.Fatal(err)
	}
	for name, ags := range map[string][]Agent{
		"self":    {{ID: "a", Name: "A", Trigger: "after:a"}},
		"missing": {{ID: "a", Name: "A", Trigger: "after:b"}},
		"loop":    {{ID: "a", Name: "A", Trigger: "after:b"}, {ID: "b", Name: "B", Trigger: "after:c"}, {ID: "c", Name: "C", Trigger: "after:a"}},
		"unknown": {{ID: "a", Name: "A", Trigger: "sometimes"}},
		"verdict": {{ID: "a", Name: "A"}, {ID: "b", Name: "B", Trigger: "after:a=approve"}},
	} {
		if checkTriggers(ags) == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestAgentListsAreNeverNull(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveAgents(DefaultAgents()); err != nil {
		t.Fatal(err)
	}
	for _, a := range s.Agents() {
		b, _ := json.Marshal(a)
		if strings.Contains(string(b), `"next":null`) || strings.Contains(string(b), `"onFail":null`) {
			t.Fatalf("%s: %s", a.ID, b)
		}
	}
}

// An agents.toml from before the git boxes became instructions: the former
// push box meant commit, push and open the PR, and an agents.toml from before
// Copilot was listed gets it, last.
func TestNormalizeOlderAgents(t *testing.T) {
	old := []Agent{
		{ID: "plan", Name: "Plan", Trigger: "manual"},
		{ID: "push", Name: "Push", Trigger: "manual", Push: true, Instructions: "Write PUSH.md."},
		{ID: "test", Name: "Test", Trigger: "manual", Commit: true},
	}
	got := normalize(old)
	var ids []string
	for _, a := range got {
		ids = append(ids, a.ID)
	}
	if strings.Join(ids, " ") != "plan push test copilot" {
		t.Fatalf("order %v", ids)
	}
	if p := got[1]; p.Push || p.Commit || p.OpenPR || !strings.HasPrefix(p.Instructions, "Write PUSH.md.\n") || !strings.Contains(p.Instructions, "commit all the changes") || !strings.Contains(p.Instructions, "push the branch") || strings.Contains(p.Instructions, "pull request") {
		t.Fatalf("former push box: %+v", p)
	}
	if x := got[2]; x.Commit || strings.Contains(x.Instructions, "push the branch") || !strings.Contains(x.Instructions, "commit") {
		t.Fatalf("former commit box: %+v", x)
	}
	if f, _ := First(got); f.ID != "plan" {
		t.Fatalf("first %s", f.ID)
	}
}

func TestKeymapMovesToConfig(t *testing.T) {
	dir := t.TempDir()
	km := `{"close": ["Escape"], "palette": ["Mod+P"]}`
	if err := os.WriteFile(filepath.Join(dir, "keymap.json"), []byte(km), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "keymap.json")); !os.IsNotExist(err) {
		t.Fatal("keymap.json kept")
	}
	if got := s.Snapshot().Keymap["palette"]; len(got) != 1 || got[0] != "Mod+P" {
		t.Fatalf("palette = %v", got)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "config.toml"))
	if !strings.Contains(string(b), "[keymap]") || strings.Contains(string(b), "Escape") {
		t.Fatalf("config.toml keeps only the changed bindings:\n%s", b)
	}
	if _, err := os.Stat(filepath.Join(dir, "views")); !os.IsNotExist(err) {
		t.Fatal("views/ created")
	}
	if err := s.SaveContext(Context{ID: "proj"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "context", "proj.md")); !os.IsNotExist(err) {
		t.Fatal("empty project context written")
	}
}

// Themes from before dark and light variants become a theme and a mode.
func TestOlderThemes(t *testing.T) {
	for old, want := range map[string][2]string{
		"dark": {"fleet", "dark"}, "light": {"fleet", "light"}, "system": {"fleet", "system"},
		"mocha": {"catppuccin", "dark"}, "latte": {"catppuccin", "light"}, "solarized": {"fleet", "dark"},
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[appearance]\ntheme = \""+old+"\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		s, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		if ap := s.Snapshot().Config.Appearance; ap.Theme != want[0] || ap.Mode != want[1] {
			t.Errorf("theme %q: got %s %s, want %s %s", old, ap.Theme, ap.Mode, want[0], want[1])
		}
	}
}

func TestCopilotCannotBeStartedByHand(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ags := DefaultAgents()
	ags[0].Next = append(ags[0].Next, CopilotID)
	if s.SaveAgents(ags) == nil {
		t.Fatal("Copilot offered as a next step")
	}
	ags = DefaultAgents()
	ags[1].Trigger = "after:" + CopilotID
	if s.SaveAgents(ags) == nil {
		t.Fatal("an agent starting after Copilot")
	}
}

func TestPromptOverride(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetPrompt(PromptSisters, "Split it: {artifact}"); err != nil {
		t.Fatal(err)
	}
	if got := Fill(s.Prompt(PromptSisters), map[string]string{"artifact": "PLAN-{n}.md"}); got != "Split it: PLAN-{n}.md" {
		t.Fatalf("prompt = %q", got)
	}
	if err := s.SetPrompt(PromptSisters, ""); err != nil {
		t.Fatal(err)
	}
	if len(s.Config().Prompts) != 0 || !strings.HasPrefix(s.Prompt(PromptSisters), "# Sister tasks") {
		t.Fatal("empty text did not restore the default")
	}
	if s.SetPrompt("nope", "x") == nil {
		t.Fatal("unknown prompt saved")
	}
}

// Copilot is the one built-in agent: whatever agents.toml says of it, only its
// trigger and next steps are kept, and it follows the agent it triggers after.
func TestCopilotIsBuiltin(t *testing.T) {
	got := normalize([]Agent{
		{ID: "push", Name: "Push"},
		{ID: CopilotID, Name: "Other", Model: "sonnet", Instructions: "Explain", Trigger: "after:push", Next: []string{"impl"}},
		{ID: "impl", Name: "Implement"},
	})
	c := got[1]
	if c.ID != CopilotID || c.Name != "Copilot" || !c.Builtin || c.Model != "" || c.Instructions != "" || c.Trigger != "after:push" || strings.Join(c.Next, ",") != "impl" {
		t.Fatalf("%+v", c)
	}
}

// With use_defaults on, agents and prompts are the repository's, whatever the
// files say, and they cannot be edited.
func TestUseDefaults(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("use_defaults = true\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "agents.toml"), []byte("[[agent]]\nid = \"mine\"\nname = \"Mine\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "prompts.toml"), []byte("[[prompt]]\nid = \"todo\"\nname = \"Todo\"\nwhere = \"x\"\ntext = \"mine\"\n"), 0o644)
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if f, _ := First(s.Agents()); f.ID != DefaultAgents()[0].ID {
		t.Fatalf("first agent %q", f.ID)
	}
	if s.Prompt(PromptTodo) == "mine" || s.Prompt(PromptTodo) != defaultPrompt(PromptTodo) {
		t.Fatalf("todo = %q", s.Prompt(PromptTodo))
	}
	if s.SaveAgents(DefaultAgents()) == nil || s.SetPrompt(PromptTodo, "x") == nil {
		t.Fatal("edited while the defaults are in use")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "prompts.toml")); !strings.Contains(string(b), "mine") {
		t.Fatal("prompts.toml was rewritten")
	}
	// Off again: the files rule.
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("use_defaults = false\n"), 0o644)
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
	if s.Prompt(PromptTodo) != "mine" {
		t.Fatalf("todo = %q", s.Prompt(PromptTodo))
	}
}

// Internal skills are prompts with a skill name and a description.
func TestInternalSkills(t *testing.T) {
	var found bool
	for _, p := range DefaultPrompts() {
		if p.Skill == "" {
			continue
		}
		found = found || p.Skill == "regression-tests"
		if strings.TrimSpace(p.Where) == "" || strings.Contains(p.Where, "\n") {
			t.Errorf("skill %s needs a one-line description (where)", p.Skill)
		}
	}
	if !found {
		t.Fatal("no regression-tests skill")
	}
}

// A new ~/.derby gets agents.toml and prompts.toml byte for byte as in the
// repository's defaults/, so the two folders compare with a plain diff.
func TestFilesMatchTheDefaults(t *testing.T) {
	dir := t.TempDir()
	if _, err := Open(dir); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string][]byte{"agents.toml": defaults.Agents, "prompts.toml": defaults.Prompts} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from defaults/%s: write the defaults with encodeTables", name, name)
		}
	}
}

// Every prompt that is a template parses, and the texts read back as written.
func TestDefaultPrompts(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range DefaultPrompts() {
		if p.ID == "" || p.Name == "" || strings.TrimSpace(p.Default) == "" || seen[p.ID] {
			t.Fatalf("prompt %+v", p)
		}
		seen[p.ID] = true
		if strings.Contains(p.Default, "{{") {
			if _, err := template.New(p.ID).Parse(p.Default); err != nil {
				t.Errorf("%s: %v", p.ID, err)
			}
		}
	}
	for _, id := range []string{PromptStep, PromptBlock, PromptContext, PromptRules, PromptCommand, PromptTaskFile, PromptRegression, PromptParts, PromptName} {
		if !seen[id] {
			t.Errorf("no default prompt %q", id)
		}
	}
}

// The prompts config.toml held before prompts.toml move there; an edited
// prompt is saved there and read back.
func TestPromptsFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[prompts]\ntodo = \"Old todo: {todo}\"\n"), 0o644)
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Prompt(PromptTodo) != "Old todo: {todo}" {
		t.Fatalf("todo = %q", s.Prompt(PromptTodo))
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "config.toml")); strings.Contains(string(b), "prompts") {
		t.Fatalf("config.toml still has the prompts:\n%s", b)
	}
	if err := s.SetPrompt(PromptRules, "# My rules\n{{.Rules}}"); err != nil {
		t.Fatal(err)
	}
	s, _ = Open(dir)
	if s.Prompt(PromptTodo) != "Old todo: {todo}" || s.Prompt(PromptRules) != "# My rules\n{{.Rules}}" {
		t.Fatalf("not read back: %q %q", s.Prompt(PromptTodo), s.Prompt(PromptRules))
	}
	if got := s.Render(PromptRules, struct{ Rules string }{"a"}); got != "# My rules\na" {
		t.Fatalf("render = %q", got)
	}
	// A broken template falls back to the default.
	_ = s.SetPrompt(PromptRules, "{{.Nope")
	if got := s.Render(PromptRules, struct{ Rules string }{"a"}); !strings.HasPrefix(got, "# Rules") {
		t.Fatalf("broken template = %q", got)
	}
}
