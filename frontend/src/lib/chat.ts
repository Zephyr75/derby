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
    if (it.kind === 'user' || it.kind === 'command') {
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
    if (it.kind === 'user' || it.kind === 'command') t++
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
      return v('file', 'Edit ' + rel(str(input.file_path)), 'diff')
    case 'NotebookEdit':
      return v('file', 'Edit ' + rel(str(input.notebook_path)), 'diff')
    case 'Write':
      return v('file', 'Write ' + rel(str(input.file_path)), 'diff')
    case 'Bash':
      return v('terminal', str(input.command), 'command')
    case 'Grep':
      return v('search', `grep ${JSON.stringify(str(input.pattern))}${input.path ? ' ' + rel(str(input.path)) : ''}`)
    case 'Glob':
      return v('search', 'Find `' + str(input.pattern) + '`')
    case 'WebFetch':
      return v('globe', 'Fetch ' + str(input.url))
    case 'WebSearch':
      return v('globe', `Search ${JSON.stringify(str(input.query))}`)
    case 'Skill':
      return v('hammer', 'Load skill: ' + (str(input.skill) || str(input.command)))
    case 'Agent':
    case 'Task':
      return v('agent', str(input.description) || 'Agent')
    case 'TodoWrite':
      return v('todos', 'Plan' + (planDone(call) ? ' · ' + planDone(call) : ''), 'todos')
    case 'AskUserQuestion':
      return v('approval', 'Asking for your input', 'question')
  }
  return v('hammer', call.tool)
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
  options: { label: string; description: string }[]
}

/** questions: an AskUserQuestion call's questions and their options. */
export function questions(call: ChatItem): Question[] {
  const qs = (call.input?.questions as unknown[]) ?? []
  return qs.map((q) => {
    const o = q as Record<string, unknown>
    const opts = (o.options as unknown[]) ?? []
    return { question: str(o.question), options: opts.map((x) => ({ label: str((x as Record<string, unknown>).label), description: str((x as Record<string, unknown>).description) })) }
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
