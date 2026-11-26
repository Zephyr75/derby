package settings

import (
	"github.com/BurntSushi/toml"

	"jockey/defaults"
)

// DefaultConfig is written to config.toml on first launch.
func DefaultConfig() Config {
	return Config{
		Appearance: Appearance{
			Theme:        "fleet",
			Mode:         "dark",
			UIFont:       "Geist",
			UIFontSize:   13,
			TermFont:     "Geist Mono",
			TermFontSize: 12.5,
			SessionView:  "terminal",
		},
		ZedCommand:      "zed",
		ClaudeCommand:   "claude",
		MaxLiveSessions: 6,
		Notifications:   true,
		MaxTabs:         5,
		BranchPrefix:    "ar-",
		FoldWords:       []string{"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "Cargo.lock", "go.sum", "poetry.lock", "uv.lock", ".min.", "node_modules", "vendor/", "dist/", "generated", ".snap"},
		Context:         map[string]ContextMeta{},
	}
}

// DefaultGeneralContext is context/general.md on first launch: empty, since
// the shared prompts (Settings > Context > Prompts) already say how to work.
const DefaultGeneralContext = ""

// DefaultAgents is the default pipeline (defaults/agents.toml): Plan,
// Implement or Discuss, Review, Test, Regression (pony-core only), Commit
// and push, then Copilot (built in). Every step ends with status: done
// (succeeded) or status: failed.
func DefaultAgents() []Agent {
	var af agentsFile
	if _, err := toml.Decode(string(defaults.Agents), &af); err != nil {
		panic("defaults/agents.toml: " + err.Error()) // caught by the tests
	}
	return af.Agent
}

// DefaultKeymap binds only what is needed every day: searching tasks, a new
// task, hiding the side panels, the chat's messages, the todo lists, and Escape for popups. config.toml's [keymap]
// can bind any other action the Shortcuts page lists (scope, tab1…tab9, …). "Mod" is
// Cmd on macOS and Ctrl elsewhere; in a terminal, Ctrl+<letter> stays the
// terminal's.
func DefaultKeymap() Keymap {
	return Keymap{
		"palette":     {"Mod+F"},
		"newTask":     {"Mod+N"},
		"toggleLeft":  {"Alt+E"},
		"toggleRight": {"Alt+A"},
		"messages":    {"Alt+S"},
		"todos":       {"Alt+T"},
		"close":       {"Escape"},
	}
}
