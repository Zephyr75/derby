// The chat view's entries: a step's transcript items, each tool call joined
// with its result, and how each tool call reads ("Edit app.go", "Run tests").

import type { ChatItem } from './types'

export interface ToolCall {
  call: ChatItem
  result: ChatItem | null
}
export type Entry = { kind: 'tool'; id: string; tool: ToolCall } | { kind: 'item'; id: string; item: ChatItem }

/** entries joins each tool call with its result; results come later in the
 *  transcript, so a call without one is still running (or was cut short). */
export function entries(items: ChatItem[]): Entry[] {
  const out: Entry[] = []
  const calls = new Map<string, ToolCall>()
  items.forEach((it, i) => {
    if (it.kind === 'tool') {
      const tool = { call: it, result: null }
      calls.set(it.id, tool)
      out.push({ kind: 'tool', id: 't' + i, tool })
    } else if (it.kind === 'result') {
      const t = calls.get(it.id)
      if (t) t.result = it
    } else out.push({ kind: 'item', id: 'i' + i, item: it })
  })
  return out
}

export interface ToolView {
  icon: string
  verb: string // "Edit", "Run", "Read"…
  target: string // a file, a command or a pattern, shown in mono
  dir: string // the file's folder, muted
  body: 'diff' | 'command' | 'todos' | 'question' | 'output' // what expanding shows
}

const str = (v: unknown) => (typeof v === 'string' ? v : '')

/** toolView: how a tool call reads in the chat. Paths are shown relative to
 *  the worktree, and run files by name. */
export function toolView(call: ChatItem, worktree: string, runDir: string): ToolView {
  const input = call.input ?? {}
  const path = (p: string) => {
    let rel = p
    if (worktree && p.startsWith(worktree + '/')) rel = p.slice(worktree.length + 1)
    else if (runDir && p.startsWith(runDir + '/')) rel = p.slice(runDir.length + 1)
    const i = rel.lastIndexOf('/')
    return { target: i >= 0 ? rel.slice(i + 1) : rel, dir: i >= 0 ? rel.slice(0, i) : '' }
  }
  const v = (icon: string, verb: string, target = '', body: ToolView['body'] = 'output', dir = ''): ToolView => ({ icon, verb, target, dir, body })
  switch (call.tool) {
    case 'Read': {
      const f = path(str(input.file_path))
      return v('file', 'Read', f.target, 'output', f.dir)
    }
    case 'Edit':
    case 'MultiEdit':
    case 'NotebookEdit': {
      const f = path(str(input.file_path) || str(input.notebook_path))
      return v('pencil', 'Edit', f.target, 'diff', f.dir)
    }
    case 'Write': {
      const f = path(str(input.file_path))
      return v('pencil', 'Write', f.target, 'diff', f.dir)
    }
    case 'Bash':
      return v('terminal', str(input.description) || 'Run', str(input.command).split('\n')[0], 'command')
    case 'Grep':
      return v('search', 'Search', str(input.pattern), 'output', str(input.path) ? path(str(input.path)).target : '')
    case 'Glob':
      return v('search', 'Find', str(input.pattern))
    case 'WebFetch':
      return v('globe', 'Fetch', str(input.url))
    case 'WebSearch':
      return v('globe', 'Search the web', str(input.query))
    case 'Skill':
      return v('skill', 'Use skill', str(input.skill) || str(input.command))
    case 'Agent':
    case 'Task':
      return v('agent', 'Agent', str(input.description))
    case 'TodoWrite':
      return v('todos', 'Update the plan', '', 'todos')
    case 'AskUserQuestion':
      return v('approval', 'Ask you', '', 'question')
    case 'ToolSearch':
      return v('tool', 'Load tools', str(input.query))
  }
  const m = /^mcp__(.+?)__(.+)$/.exec(call.tool)
  if (m) return v('tool', m[2].replace(/_/g, ' '), m[1].replace(/^claude_ai_/, '').replace(/_/g, ' '))
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

/** todos: a TodoWrite call's list. */
export function todos(call: ChatItem): { text: string; status: string }[] {
  const ts = (call.input?.todos as unknown[]) ?? []
  return ts.map((t) => {
    const o = t as Record<string, unknown>
    return { text: str(o.content) || str(o.subject), status: str(o.status) }
  })
}
