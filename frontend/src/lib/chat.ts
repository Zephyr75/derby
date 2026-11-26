// The chat view's entries, after Zed's agent panel (crates/agent_ui, its
// thread view): a step's transcript items, each tool call joined with its
// result, grouped in turns (a user message and the answer to it), and each
// tool call titled and iconed as Zed shows Claude Code's (the Claude Agent
// ACP adapter's titles: "Read a.ts", "Edit src/app.ts", the command…).

import type { ChatItem } from './types'

export interface ToolCall {
  call: ChatItem
  result: ChatItem | null
}
export type Entry =
  | { kind: 'tool'; id: string; turn: number; tool: ToolCall }
  | { kind: 'item'; id: string; turn: number; item: ChatItem }
  | { kind: 'end'; id: string; turn: number; bottom: boolean } // a turn's controls: copy, scroll

/** entries joins each tool call with its result (results come later in the
 *  transcript: a call without one still runs, or was cut short) and ends each
 *  answered turn with its controls. */
export function entries(items: ChatItem[]): Entry[] {
  const out: Entry[] = []
  const calls = new Map<string, ToolCall>()
  let turn = 0
  let answered = false
  const end = () => answered && out.push({ kind: 'end', id: 'e' + turn, turn, bottom: false })
  items.forEach((it, i) => {
    if (it.kind === 'user' || it.kind === 'command' || it.kind === 'bash') {
      end()
      turn++
      answered = false
      out.push({ kind: 'item', id: 'i' + i, turn, item: it })
    } else if (it.kind === 'tool') {
      const tool = { call: it, result: null }
      calls.set(it.id, tool)
      out.push({ kind: 'tool', id: 't' + i, turn, tool })
    } else if (it.kind === 'result') {
      const t = calls.get(it.id)
      if (t) t.result = it
    } else {
      if (it.kind === 'text') answered = true
      out.push({ kind: 'item', id: 'i' + i, turn, item: it })
    }
  })
  if (answered) out.push({ kind: 'end', id: 'e' + turn, turn, bottom: true })
  return out
}

/** response: a turn's answer, as markdown (Copy This Agent Response). */
export function response(items: ChatItem[], turn: number): string {
  let t = 0
  const out: string[] = []
  for (const it of items) {
    if (it.kind === 'user' || it.kind === 'command' || it.kind === 'bash') t++
    else if (t === turn && it.kind === 'text') out.push(it.text)
  }
  return out.join('\n\n')
}

export interface ToolView {
  icon: string // Zed's icon for the tool's kind
  title: string // markdown, one line
  body: 'diff' | 'command' | 'todos' | 'question' | 'output' // what unfolding shows
  card: boolean // edits and commands are cards; other tools a muted line
}

const str = (v: unknown) => (typeof v === 'string' ? v : '')

/** toolView: a tool call's title and icon. Paths are shown relative to the
 *  worktree, and run files by name. */
export function toolView(call: ChatItem, worktree: string, runDir: string): ToolView {
  const input = call.input ?? {}
  const rel = (p: string) => {
    if (worktree && p.startsWith(worktree + '/')) return p.slice(worktree.length + 1)
    if (runDir && p.startsWith(runDir + '/')) return p.slice(runDir.length + 1)
    return p
  }
  const v = (icon: string, title: string, body: ToolView['body'] = 'output'): ToolView => ({ icon, title, body, card: body === 'diff' || body === 'command' })
  switch (call.tool) {
    case 'Read':
      return v('search', 'Read ' + rel(str(input.file_path)))
    case 'Edit':
    case 'MultiEdit':
      return v('pencil', 'Edit ' + rel(str(input.file_path)), 'diff')
    case 'NotebookEdit':
      return v('pencil', 'Edit ' + rel(str(input.notebook_path)), 'diff')
    case 'Write':
      return v('pencil', 'Write ' + rel(str(input.file_path)), 'diff')
    case 'Bash':
      return v('run', str(input.command), 'command')
    case 'Grep':
      return v('search', `grep ${JSON.stringify(str(input.pattern))}${input.path ? ' ' + rel(str(input.path)) : ''}`)
    case 'Glob':
      return v('search', 'Find `' + str(input.pattern) + '`')
    case 'WebFetch':
      return v('globe', 'Fetch ' + str(input.url))
    case 'WebSearch':
      return v('globe', `Search ${JSON.stringify(str(input.query))}`)
    case 'Skill':
      return v('extension', 'Load skill: ' + (str(input.skill) || str(input.command)))
    case 'Agent':
    case 'Task':
      return v('delegate', str(input.description) || 'Agent')
    case 'TodoWrite':
      return v('todo', 'Plan' + (planDone(call) ? ' · ' + planDone(call) : ''), 'todos')
    case 'AskUserQuestion':
      return v('ask', 'Asking for your input', 'question')
  }
  return v('tool', call.tool)
}

/** diffStat counts a unified diff's added and removed lines. */
export function diffStat(diff: string): { add: number; del: number } {
  let add = 0
  let del = 0
  for (const l of diff.split('\n')) {
    if (l.startsWith('+')) add++
    else if (l.startsWith('-')) del++
  }
  return { add, del }
}

export interface Question {
  question: string
  header: string
  multi: boolean // several options may be picked
  options: { label: string; description: string }[]
}

/** questions: an AskUserQuestion call's questions and their options. */
export function questions(call: ChatItem): Question[] {
  const qs = (call.input?.questions as unknown[]) ?? []
  return qs.map((q) => {
    const o = q as Record<string, unknown>
    const opts = (o.options as unknown[]) ?? []
    return { question: str(o.question), header: str(o.header), multi: o.multiSelect === true, options: opts.map((x) => ({ label: str((x as Record<string, unknown>).label), description: str((x as Record<string, unknown>).description) })) }
  })
}

function planDone(call: ChatItem) {
  const ts = todos(call)
  return ts.length ? `${ts.filter((t) => t.status === 'completed').length}/${ts.length} done` : ''
}

/** todos: a TodoWrite call's list. */
export function todos(call: ChatItem): { text: string; status: string }[] {
  const ts = (call.input?.todos as unknown[]) ?? []
  return ts.map((t) => {
    const o = t as Record<string, unknown>
    return { text: str(o.content) || str(o.subject), status: str(o.status) }
  })
}

/** rawInput: a tool call's input, as Zed's "Raw Input:" shows it. */
export function rawInput(call: ChatItem): string {
  return JSON.stringify(call.input ?? {}, null, 2)
}

/** A "/" suggestion: a built-in command, or one of the session's skills. */
export interface Command {
  name: string
  description: string
  terminal: boolean // Claude Code shows it in a panel of its own: the chat switches to the terminal
}

/** builtins: Claude Code's commands (names and descriptions from Claude Code
 *  itself) that make sense in a step's session. /model and /effort are left
 *  out: they change the user's default for every new session, while the
 *  editor's dropdowns change only this one. */
const builtins: Command[] = [
  ['compact', 'Free up context by summarizing the conversation so far', false],
  ['clear', 'Start a new session with empty context; the previous one stays resumable', false],
  ['recap', 'Generate a one-line session recap now', false],
  ['reload-skills', 'Pick up skills added or changed on disk during this session', false],
  ['context', 'Show what fills the context', true],
  ['usage', 'Show session cost, plan usage, and activity stats', true],
  ['status', 'Show version, account, model, MCP servers, and setup warnings', true],
  ['btw', 'Ask a side question without interrupting Claude', false],
  ['plan', 'Turn on plan mode or view the plan', true],
  ['goal', 'Set a goal Claude checks before stopping', true],
  ['tasks', 'View and manage everything running in the background', true],
  ['add-dir', 'Add a working directory', true],
  ['permissions', 'Manage allow and deny tool permission rules', true],
  ['memory', 'Edit CLAUDE.md files and memory settings', true],
  ['mcp', 'Manage MCP servers', true],
  ['hooks', 'View hook configurations for tool events', true],
  ['skills', 'List available skills', true],
  ['export', 'Export this session to a file or clipboard', true],
  ['help', 'Show help and available commands', true],
].map(([name, description, terminal]) => ({ name: name as string, description: description as string, terminal: terminal as boolean }))

/** commands: the "/" suggestions for query, names that start with it first. */
export function commands(skills: { name: string; description: string }[], query: string): Command[] {
  const all = [...builtins, ...skills.filter((s) => !builtins.some((b) => b.name === s.name)).map((s) => ({ ...s, terminal: false }))]
  const q = query.toLowerCase()
  const rank = (c: Command) => {
    const n = c.name.toLowerCase()
    return n.startsWith(q) ? 0 : n.split(':').some((p) => p.startsWith(q)) ? 1 : n.includes(q) ? 2 : -1
  }
  return all
    .filter((c) => rank(c) >= 0)
    .sort((a, b) => rank(a) - rank(b))
    .slice(0, 30)
}

/** terminalCommand: a message that runs a command Claude Code shows in its own panel. */
export function terminalCommand(text: string): boolean {
  const m = /^\/([\w:-]+)/.exec(text.trim())
  return !!m && builtins.some((b) => b.terminal && b.name === m[1])
}

const base = (p: string) => p.replace(/^['"]|['"]$/g, '').split('/').filter(Boolean).pop() ?? p

// What a command without a description of its own is called: a sentence by
// program (and its first subcommand), else "Run <program> <subcommand>".
const sentences: Record<string, string | Record<string, string>> = {
  git: { status: 'Check the git status', diff: 'Show the diff', log: 'Read the git log', show: 'Show a commit', add: 'Stage the changes', commit: 'Commit the changes', push: 'Push the branch', pull: 'Pull from origin', fetch: 'Fetch from origin', checkout: 'Switch branch', switch: 'Switch branch', rebase: 'Rebase the branch', stash: 'Stash the changes', branch: 'List the branches', worktree: 'Manage worktrees', merge: 'Merge a branch', reset: 'Reset the branch', 'rev-parse': 'Look up a commit' },
  go: { test: 'Run the Go tests', build: 'Build the Go code', vet: 'Vet the Go code', run: 'Run the Go program', mod: 'Update the Go modules' },
  gh: { pr: 'Look at the pull request', issue: 'Look at the issue', api: 'Call the GitHub API', run: 'Look at the CI run', repo: 'Look at the repository' },
  npm: { test: 'Run the tests', ci: 'Install the dependencies', install: 'Install the dependencies', run: 'Run an npm script' },
  make: 'Run make',
  pytest: 'Run the Python tests',
  python: 'Run a Python script', python3: 'Run a Python script',
  node: 'Run a Node script',
  ls: 'List files', tree: 'List files', find: 'Find files', fd: 'Find files',
  grep: 'Search the code', rg: 'Search the code',
  mkdir: 'Create a folder', rm: 'Delete files', mv: 'Move files', cp: 'Copy files', chmod: 'Change permissions',
  curl: 'Call a URL', open: 'Open a file', kill: 'Stop a process', pkill: 'Stop a process', ps: 'List processes', wc: 'Count lines', diff: 'Compare files', echo: 'Print text', sleep: 'Wait',
  docker: 'Run docker', kubectl: 'Run kubectl', gcloud: 'Run gcloud',
}
const reads = new Set(['cat', 'head', 'tail', 'sed', 'awk', 'less', 'bat'])

/** commandLabel: what a command line says it does: the agent's own
 *  description, else a sentence made from the command (the "cd dir &&" and
 *  VAR=value prefixes dropped, only its first pipeline stage looked at). */
export function commandLabel(input: Record<string, unknown> | null): string {
  const description = str(input?.description).trim()
  if (description) return description
  let cmd = str(input?.command).trim().split('\n')[0]
  cmd = cmd.replace(/^cd\s+\S+\s*&&\s*/, '').replace(/^(?:[A-Za-z_][A-Za-z0-9_]*=\S*\s*[;&]*\s*)+/, '')
  const stage = cmd.split(/\s*(?:\||&&|;|2>&1)\s*/)[0].trim()
  const words = stage.split(/\s+/).filter(Boolean)
  const prog = base(words[0] ?? '')
  if (!prog) return 'Run a command'
  const args = words.slice(1).filter((w) => !w.startsWith('-'))
  if (/^(cat|tee)$/.test(prog) && />>?\s*\S+/.test(cmd) && cmd.includes('<<')) return 'Write ' + base(/>>?\s*(\S+)/.exec(cmd)![1])
  if (reads.has(prog) && args.length) return 'Read ' + base(args[args.length - 1])
  const known = sentences[prog]
  if (typeof known === 'string') return known + (prog === 'make' && args[0] ? ' ' + args[0] : '')
  if (known && args[0] && known[args[0]]) return known[args[0]]
  return ['Run', prog, args[0]].filter(Boolean).join(' ')
}

/** commandFailure: the short tag of a failed command: "2 failed" when its
 *  output counts failures (tests), else "failed". */
export function commandFailure(text: string): string {
  const n = /(\d+)\s+(?:failed|failures?|errors?)\b/i.exec(text)?.[1] ?? String(text.split('\n').filter((l) => /^--- FAIL/.test(l)).length || '')
  return n ? `${n} failed` : 'failed'
}
