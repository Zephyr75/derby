package settings

import "strings"

// Prompt is a text herd adds to what agents get, beside their own
// instructions. Settings > Context > Prompts edits it; config.toml's
// [prompts] keeps only the ones that differ from Default.
type Prompt struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Where   string `json:"where"` // where herd uses it
	Vars    string `json:"vars"`  // what herd replaces in it
	Default string `json:"default"`
	Text    string `json:"text"` // the text in use: the user's, else Default
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
)

// DefaultPrompts lists every prompt, in the order Settings shows them. The
// first six are the shared parts of every agent's system prompt, in that
// order; each agent's own instructions only say what is special to it.
func DefaultPrompts() []Prompt {
	return []Prompt{
		{ID: PromptStep, Name: "How a step works", Where: "Every agent's system prompt, after the herd step block", Default: `# How a step works
- Read TASK.md in the run directory ($HERD_RUN_DIR) first. Earlier steps' artifacts are there too: read the newest of each kind your instructions name (older ones were handled by earlier steps; older tasks have PLAN.md).
- {n}, {base}, {sha} and {issue} in your instructions are the values in the herd step block.
- Text after -- in your command is the user's guidance for this step: follow it. When the plan lists Options, take the one the guidance picks, else the recommended one.
- Work in the current directory, the task's own git worktree. Never commit or push: herd does it.
- Write your artifact in the run directory. It starts with YAML front matter; once the step is truly finished, add status: done (it succeeded) or status: failed (it did not: say why) to it. The next automatic step only starts after a success.
- If you need the user, ask in the terminal and stop.
- Skills to use when they help: context7 (current docs of libraries), archify (diagrams), humanizer (TLDR and ELI5).`},
		{ID: PromptWriting, Name: "Writing", Where: "Every agent's system prompt, Copilot's too: for the artifact and every reply", Default: `# Writing
- Write for someone new to this code: short, plain words, intuitive. Prefer a table or a list to a long paragraph. Explain trade-offs in the artifact, not in code comments.
- Open the artifact, right after its front matter, with **TLDR**: two or three sentences on what and why. When the change or the mechanism is not obvious, add **ELI5**: a short paragraph with an everyday metaphor, without jargon, file or function names. Write these two with the humanizer skill, and nothing else with it.
- Verify every claim before you write it: in the code, by running it, or in logs or docs. Leave a claim unverified only when checking it is impossible (e.g. it needs production access or the user's knowledge), not because it takes effort; then mark it with your confidence and why it cannot be checked, in the artifact and in every reply, e.g. **Unverified (medium confidence, needs prod logs):** the retry also covers timeouts.`},
		{ID: PromptCode, Name: "Code changes", Where: "Every agent's system prompt", Default: `# Code changes
- Keep the diff small and in the repo's style; no new dependencies without asking.
- Never add a copy of existing code: reuse it, and merge the repeated code in what you touch.
- Give new code a clean place in the structure (the right file, function and name) instead of bolting it on.
- After changing code, run the existing tests that cover it and the build or type-check. Leave writing tests to the Test step, unless the user asks.
- When there are several reasonable options, lay them out with your recommendation and let the user choose.
- Problems you find outside the task (bugs, debt, confusing code): don't fix them; list them under "Problems found" in your artifact. If some are bugs worth fixing, ask before you finish: write the artifact without its status line, sum the bugs up in a few lines here, ask the user which to fix, and stop. Fix the ones they pick (maybe none), update the artifact, then finish. If you never edit code, only list them.`},
		{ID: PromptDiagrams, Name: "Diagrams", Where: "Every agent's system prompt, and Copilot's prompt", Vars: "{stem}: the artifact's name without .md, e.g. PLAN-1", Default: `# Diagrams
Draw a diagram when it helps to see a change: how a flow works, and how the change fits the code's structure and tools. Use the archify skill; never draw one by hand.
- Save it in the run directory as {stem}.html ({stem}-<topic>.html for more than one). Keep archify's working files in $HERD_RUN_DIR/.archify/<name> (pass it as finalize's --out-dir too), never in the worktree.
- Embed it in the artifact on its own line: ![<what it shows>]({stem}.html). herd shows it there, and full size.
- Show what the reader needs, not every detail; mark what changes.
- Write a mermaid block instead only when your instructions ask for one, or when archify is not in your skills.`},
		{ID: PromptSisters, Name: "Sister tasks", Where: "The system prompt of each agent with Can create sister tasks. Keep the ## Sister tasks and ### <repository> headings: Next step reads them", Vars: "{artifact}: the agent's artifact, e.g. PLAN-1.md", Default: `# Sister tasks
When the work also needs changes in other repositories, do only this repository's part. Above the "---" line below, add a "## Parts overview" section for the whole work: "- <repository folder name>: <what that part does>", this repository included, so the user sees at a glance what each part is and how they fit. Each line is under 10 words and shown in full, so start with what is specific to that part (never the same verb on every line) and end with the link to the others when there is one (e.g. "needs X's new endpoint"). Then end {artifact} with a "---" line and a "## Sister tasks" section: one "### <repository folder name>" per repository, then what that part must do, written as its task's description. herd offers to create each one in a click. Leave out repositories that already have a sister task (listed in the herd step block).`},
		{ID: PromptComments, Name: "Review comments", Where: "The system prompt of the steps of a task with a pull request", Default: `# Review comments
Judge each review comment you handle (from your guidance or the newest FEEDBACK-n.md) on its merits: fix the code only when its concern is real and worth fixing; when it is wrong or unnecessary, reject it: change nothing and say why in your artifact. When a comment asks a question, give the answer to post. List the ids you handled (the number after "comment") in the artifact's front matter, comma-separated: fixed: <ids> and rejected: <ids>. herd marks the rejected ones in the PR tab, for the user to resolve.`},
		{ID: PromptRebase, Name: "Rebase", Where: "The guidance of Rebase with Implement: in the PR tab when the PR conflicts with its base, and in the task header when the branch is behind it", Vars: "{base}: the base branch, {behind}: how many of its commits the branch lacks", Default: `Rebase this branch on origin/{base}, which has {behind} commits it lacks. git fetch origin {base}, then git rebase origin/{base}; resolve each conflict keeping what both sides meant, and continue with git rebase --continue (the only commits allowed in this step). Run the tests of the files you resolved. List the conflicts and how you resolved them, or say there were none.`},
		{ID: PromptFixChecks, Name: "Fix CI checks", Where: "The guidance of Fix with Implement, in the PR tab's CI checks", Vars: "{pr}: the PR number, {checks}: the failing checks, {links}: their links", Default: `Fix the failing CI checks of PR #{pr}: {checks}. Read their logs (gh run view <run id> --log-failed; the run id is in each check's link: {links}), fix the cause in the code, or the test when the test is what is wrong, and run those tests locally when you can.`},
		{ID: PromptFeedback, Name: "Fix review comments", Where: "The guidance of Fix with Implement, in the PR tab's Review comments", Vars: "{pr}: the PR number, {comments}: the chosen comments, with their file, line and replies", Default: `Address these review comments on PR #{pr}, and list in your artifact which comment each change answers.

{comments}`},
		{ID: PromptNotes, Name: "Diff comments", Where: "The guidance of the step that Send to starts, from your comments on the diff (Changes)", Vars: "{comments}: your comments, each with its file, lines and code", Default: `Address my comments on the diff, one by one. Each gives the file, the lines and their code, then what I want. List in your artifact what you changed for each.

{comments}`},
		{ID: PromptTodo, Name: "Todo", Where: "The guidance of the step a todo's play button starts (Implement by default)", Vars: "{todo}: the todo's text", Default: `Do this todo of the task:

{todo}`},
	}
}

// Fill replaces each {name} of vars in text.
func Fill(text string, vars map[string]string) string {
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}
