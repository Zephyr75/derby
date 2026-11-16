package settings

// DefaultConfig is written to config.toml on first launch.
func DefaultConfig() Config {
	return Config{
		Appearance: Appearance{
			Theme:        "dark",
			UIFont:       "Geist",
			UIFontSize:   13,
			TermFont:     "Geist Mono",
			TermFontSize: 12.5,
		},
		ZedCommand:      "zed",
		ClaudeCommand:   "claude",
		MaxLiveSessions: 6,
		Notifications:   true,
		MaxTabs:         5,
		BranchPrefix:    "ar-",
		Context:         map[string]ContextMeta{},
	}
}

// DefaultGeneralContext is context/general.md on first launch: empty, since
// the shared prompts (Settings > Context > Prompts) already say how to work.
const DefaultGeneralContext = ""

// DefaultAgents is the default pipeline: Plan, Implement or Discuss, Review,
// Test, Commit and push, then Copilot (built in). Every step ends with
// status: done (succeeded) or status: failed.
func DefaultAgents() []Agent {
	return []Agent{
		{
			ID: "plan", Name: "Plan", Command: "/herd:plan {task} -- {guidance}", Model: "sonnet", Trigger: "manual",
			Writes: "PLAN-{n}.md", Sisters: true, Choose: true, Next: []string{"impl", "discuss"}, OnFail: []string{"plan"},
			Instructions: `Read ISSUE.md too when the task has a GitHub issue (herd copied it there, comments included). Explore the repo and decide an approach. Do not edit code.
PLAN-{n}.md sections, after the TLDR:
1. Goal: what is wrong or missing today and what will be true once it is done.
2. Current flow: an archify diagram of how the code works today (PLAN-{n}-current.html), from the code with source evidence, the part that goes wrong marked.
3. New flow: the same flow with the fix applied and what changes marked (PLAN-{n}-new.html), laid out like the current one so the two compare at a glance.
4. Options, only when there is a real choice (designs, libraries, where the code goes): a table of each one's pros, cons and size (about +N / -M lines), then the one you recommend.
5. Files: a table, one row per file to change, with what changes there and why. Under it: "Estimated change: about +N / -M lines".
6. Tests: the behaviours to verify (main path, edge cases, what must not change).
Fail when the task cannot be done as described.`,
		},
		{
			ID: "impl", Name: "Implement", Command: "/herd:implement {task} -- {guidance}", Model: "opus", Trigger: "manual",
			Writes: "IMPLEMENT-{n}.md", ShowDiff: true, Sisters: true, Revert: true, OnFail: []string{"impl", "discuss", "plan"},
			Instructions: `Read the newest PLAN, REVIEW, TEST, COPILOT, IMPLEMENT and DISCUSS files.
Implement the plan's New flow and Files. When the task has a pull request, also fix its failing CI checks (gh pr checks) and the comments in the newest FEEDBACK-n.md.
IMPLEMENT-{n}.md sections, after the TLDR:
1. What changed: the option taken and why (when the plan has Options), then how each part of the code changed, following the flow.
2. Files: a table, one row per changed file, with what changed in it.
3. Review comments, when you handled some: one row each (id, title), fixed or rejected, and why.
4. Notes for the reviewer, when there are any: deviations from the plan, what to look at closely, checks that failed or could not run.
Fail when you could not implement it (say what blocks).`,
		},
		{
			ID: "discuss", Name: "Discuss", Command: "/herd:discuss {task} -- {guidance}", Model: "opus", Trigger: "manual",
			Writes: "DISCUSS-{n}.md", ShowDiff: true, Sisters: true, Revert: true, Next: []string{"review", "impl"}, OnFail: []string{"discuss", "impl"},
			Instructions: `Read the newest PLAN, REVIEW, TEST, COPILOT and FEEDBACK files, and the branch's changes (git diff origin/{base}...HEAD, git status).
This step is a conversation led by the user: they may want to understand something, explore an idea or change code. Do not start working on your own: say in two or three lines where the task stands (from the guidance when there is some), then wait for their questions.
Change code only when asked, one change at a time, explaining each briefly.
Write DISCUSS-{n}.md only when the user says the discussion is over: what you discussed and concluded, or what they asked for. If you changed code, add What changed (with the plan's option taken, if any) and Files, for Review.
Fail when the user stops the work.`,
		},
		{
			ID: "review", Name: "Review", Command: "/herd:review {task}", Model: "opus", Trigger: "after:impl",
			Writes: "REVIEW-{n}.md", Rerun: true, Next: []string{"test", "impl", "discuss"}, OnFail: []string{"impl", "discuss"},
			Instructions: `Review the uncommitted changes (git status, git diff HEAD) against the newest PLAN (its New flow and Files) and the newest IMPLEMENT or DISCUSS file. Never edit code, and do not ask for tests: Test adds them.
REVIEW-{n}.md: the findings, most important first.
Done when the changes are ready, failed when they need changes.`,
		},
		{
			ID: "test", Name: "Test", Command: "/herd:test {task} -- {guidance}", Model: "sonnet", Trigger: "manual",
			Writes: "TEST-{n}.md", ShowDiff: true, Next: []string{"push", "impl", "discuss"}, OnFail: []string{"impl", "discuss"},
			Instructions: `Read the newest PLAN (its Tests section lists what to verify), IMPLEMENT, DISCUSS and REVIEW files.
Write tests for the uncommitted changes (git status, git diff HEAD), following the repo's conventions: the plan's Tests, the edge cases and anything the reviews flagged. When the repo has no tests, say so and write none.
First check the tests this branch already added or changed (git diff origin/{base} on the test files): update or remove the ones that no longer match the code. Never change tests that exist on {base}.
Run them. Change only tests: if one shows a bug, keep it failing and explain it.
Report only checks that ran, with the exact command and pass/fail counts. If a run cannot start (no credentials or network), say so; never point tests at real resources to make them run.
TEST-{n}.md sections, after the TLDR: Tests added, How to run, Results.
Done when the tests pass, failed when one shows a bug.`,
		},
		{
			ID: "push", Name: "Commit and push", Command: "/herd:push {task}", Model: "haiku", Trigger: "manual",
			Writes: "PUSH.md", Commit: true, Push: true, OpenPR: true, IgnoreRules: true, Next: []string{"impl", "discuss"},
			Instructions: PushInstructions,
		},
		{
			ID: CopilotID, Name: "Copilot", Trigger: "after:push", Writes: "COPILOT-{n}.md", Builtin: true, Model: "opus", Instructions: CopilotInstructions,
			Next: []string{"impl", "discuss"}, OnFail: []string{"impl", "discuss"},
		},
	}
}

// PushInstructions: the default PR agent asks for the PR's type, scope and
// title in a form, then writes the commit message and the PR in PUSH.md.
const PushInstructions = `First bring the branch up to date: git fetch origin {base}; when origin/{base} has commits this branch lacks, git rebase --autostash origin/{base}. If the rebase stops on a conflict, git rebase --abort, list the conflicting files and fail.
Then ask the user in one AskUserQuestion form: the type (feat, fix, perf, test, refactor, chore), the scope (the part of the code it touches) and the title, suggesting a scope and a title from the changes.
PUSH.md is the commit message and the PR, so no TLDR or ELI5 in it:
- the title line "<type>(<scope>): <title> #{issue}";
- a short body: what changed and why, from the newest PLAN's Goal and the What changed of the IMPLEMENT and DISCUSS files, ending with "Closes #{issue}" ("Part of" for a sister task; no issue references for a LOCAL task);
- "## Choices", when the newest PLAN lists Options: for each choice, the option taken, the ones set aside in a few words, and why;
- "## Flow": the flow before and after as two mermaid flowcharts (GitHub renders mermaid, not archify), from the newest PLAN's flow diagrams, updated with what was built. About 10 nodes each; mark what changed in the node text, e.g. "(new)" or "(changed)", with no colors, fills, classDef or style lines (unreadable in GitHub's themes). Leave it out when there is no flow to show (docs, config, a one-line fix).
herd then commits with PUSH.md's title line and body, pushes, and opens or updates the draft PR; Copilot reviews it.`

// DefaultKeymap binds only what is needed every day: searching tasks, a new
// task, hiding the side panels, the todo lists, and Escape for popups. config.toml's [keymap]
// can bind any other action the Shortcuts page lists (scope, tab1…tab9, …). "Mod" is
// Cmd on macOS and Ctrl elsewhere; in a terminal, Ctrl+<letter> stays the
// terminal's.
func DefaultKeymap() Keymap {
	return Keymap{
		"palette":     {"Mod+F"},
		"newTask":     {"Mod+N"},
		"toggleLeft":  {"Alt+E"},
		"toggleRight": {"Alt+A"},
		"todos":       {"Alt+T"},
		"close":       {"Escape"},
	}
}
