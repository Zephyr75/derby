package settings

import (
	"log"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"

	"jockey/defaults"
)

// Prompt is a text Jockey adds to what agents get, beside their own
// instructions. prompts.toml lists every one; Settings > Context > Prompts
// edits it there.
type Prompt struct {
	ID      string `toml:"id" json:"id"`
	Name    string `toml:"name" json:"name"`
	Where   string `toml:"where" json:"where"`           // where Jockey uses it
	Vars    string `toml:"vars,omitempty" json:"vars"`   // what Jockey replaces in it
	Skill   string `toml:"skill,omitempty" json:"skill"` // set: an internal skill, /jockey:<Skill> in every session; Where is its description
	Default string `toml:"-" json:"default"`
	Text    string `toml:"text" json:"text"` // the text in use: the user's, else Default
}

type promptsFile struct {
	Prompt []Prompt `toml:"prompt"`
}

// Prompt ids.
const (
	PromptStep      = "step"
	PromptWriting   = "writing"
	PromptCode      = "code"
	PromptDiagrams  = "diagrams"
	PromptSisters   = "sisters"
	PromptComments  = "comments"
	PromptRebase    = "rebase"
	PromptFixChecks = "fixChecks"
	PromptFeedback  = "fixFeedback"
	PromptNotes     = "diffNotes"
	PromptTodo      = "todo"
	// Internal skills: Claude Code skills Jockey ships in its plugin.
	PromptRegression = "regressionTests"
	// Jockey's own texts around the agents' prompts.
	PromptBlock    = "stepBlock"
	PromptContext  = "context"
	PromptRules    = "rules"
	PromptCommand  = "command"
	PromptTaskFile = "taskFile"
	PromptParts    = "partsSummary"
	PromptName     = "taskName"
	PromptClaudeMd = "claudeMd"
)

// DefaultPrompts lists every prompt (defaults/prompts.toml), in the order
// Settings shows them. The first six are the shared parts of every agent's
// system prompt, in that order; each agent's own instructions only say what
// is special to it. The last ones are Jockey's own texts around them.
func DefaultPrompts() []Prompt {
	var pf promptsFile
	if _, err := toml.Decode(string(defaults.Prompts), &pf); err != nil {
		panic("defaults/prompts.toml: " + err.Error()) // caught by the tests
	}
	for i := range pf.Prompt {
		pf.Prompt[i].Default = pf.Prompt[i].Text
	}
	return pf.Prompt
}

// Fill replaces each {name} of vars in text.
func Fill(text string, vars map[string]string) string {
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}

// Render fills a prompt that is a Go template (text/template) with data. A
// template that no longer parses or runs (edited in prompts.toml) falls back
// to the default, so a step always gets its prompt.
func (s *Store) Render(id string, data any) string {
	out, err := render(s.Prompt(id), data)
	if err == nil {
		return out
	}
	log.Printf("prompt %s: %v; using the default", id, err)
	out, _ = render(defaultPrompt(id), data)
	return out
}

func render(text string, data any) (string, error) {
	t, err := template.New("").Option("missingkey=error").Parse(text)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func defaultPrompt(id string) string {
	for _, p := range DefaultPrompts() {
		if p.ID == id {
			return p.Default
		}
	}
	return ""
}
