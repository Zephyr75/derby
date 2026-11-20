// Shapes returned by the Go App service (internal/app, internal/settings).

export type StepState = 'running' | 'idle' | 'approval' | 'done' | 'failed'
export type StatusKind = 'running' | 'waiting' | 'approval' | 'idle' | 'done' | 'failed'

export interface Project {
  id: string
  name: string
  path: string
  repo: string
  baseBranch: string
  worktreesDir: string
  commonIssues: boolean // its issues live in the common issues repository (config.issuesRepo)
}

export interface Agent {
  id: string
  name: string
  command: string
  model: string
  trigger: string // manual | after:<id> (when that step succeeds)
  writes: string
  commit: boolean // derby commits when it succeeds: its artifact's title line and body are the message
  push: boolean // then pushes the branch (needs commit)
  openPR: boolean // then opens or updates the draft PR from its artifact (needs push)
  revert: boolean // its latest step can be reverted while nothing is committed after it
  rerun: boolean // its step can be run again while it is the latest
  choose: boolean // when its artifact lists Options, Next step asks which one to take
  builtin: boolean // run by derby itself, not Claude: Copilot
  ignoreRules: boolean // the common rules are not given to this agent
  showDiff: boolean // makes changes: its steps are listed in the diff pop-up
  sisters: boolean // its artifact can ask for sister tasks in other repositories
  allow: string[] // tool rules it runs without asking, e.g. Bash(make test-path:*)
  next: string[] // agents Next step offers after it succeeds, first highlighted
  onFail: string[] // agents Next step offers after it fails (empty: itself again)
  effort: string // low | medium | high | xhigh | max
  projects: string[] // ids of the projects that offer it (empty: every project)
  instructions: string
}

export interface Context {
  id: string
  text: string
  claudeMd: boolean
  knowledge: string // a knowledge base folder sessions can read
}

export type ThemeName = 'fleet' | 'catppuccin' | 'copilot'
export type ThemeMode = 'dark' | 'light' | 'system'

export interface Appearance {
  theme: ThemeName
  mode: ThemeMode // the theme's dark or light variant, or the OS's (system)
  uiFont: string
  uiFontSize: number
  termFont: string
  termFontSize: number
  sessionView: 'terminal' | 'chat' // how the center shows a step's Claude Code session
}

/** One entry of a step's conversation, read from Claude Code's transcript. */
export interface ChatItem {
  kind: 'user' | 'command' | 'bash' | 'bashout' | 'text' | 'thinking' | 'tool' | 'result' | 'note' | 'error'
  id: string // the tool call a tool or result item belongs to
  text: string // the message; a result's output; a command's arguments
  tool: string // a tool call's tool, a command's name
  input: Record<string, unknown> | null // a tool call's input (long values cut)
  diff: string // a result's change to a file, as a unified diff
  isError: boolean
}

/** The part of a step's conversation after a byte offset (StepChat). */
export interface Chat {
  items: ChatItem[]
  offset: number
  reset: boolean // the transcript was replaced: drop what was shown
  title: string
  found: boolean
  status: SessionStatus // from the session's statusline (live), else its transcript
  live: boolean
  skills: Skill[] | null // the session's skills and plugin commands, when the transcript read listed them
  trust: string // the folder Claude Code asks to trust before it starts, while it asks
}

/** A skill or command a session offers after "/". */
export interface Skill {
  name: string
  description: string
}

/** A session's model, effort and context window, and the plan's usage. */
export interface SessionStatus {
  model: string // e.g. claude-opus-5-5
  modelName: string // e.g. Opus 5.5
  effort: string
  contextUsed: number
  contextSize: number // 0: unknown
  plan: { fiveHour: number; fiveHourReset: number; week: number; weekReset: number; at: number } // percents, -1 unknown; resets in unix seconds
}

export interface FileChange {
  path: string
  status: string // A added, M modified, D deleted, ? untracked
  added: number
  deleted: number
  binary: boolean
}

export interface Changes {
  source: '' | 'commit' | 'worktree' | 'branch' // branch: the Copilot step, the whole branch as its PR shows it
  sha: string
  base?: string // branch: what it is compared to, e.g. origin/dev
  via: string // the step that made the commit, when not this one
  files: FileChange[]
}

/** The user's ~/.claude/CLAUDE.md compared with General context + Rules. */
export interface SystemMemory {
  path: string
  exists: boolean
  synced: boolean
}

/** One file's unified diff in a step's changes. */
export interface FileDiff {
  path: string
  diff: string
  binary: boolean
  truncated: boolean
  old: string // the whole file before and after, '' where it did not exist
  new: string
  sources: boolean // false: too big, old and new are empty
}

/** A process outside derby running a session. */
export interface Proc {
  pid: number
  cmd: string
}

export interface Config {
  appearance: Appearance
  zedCommand: string
  claudeCommand: string
  maxLiveSessions: number
  notifications: boolean
  maxTabs: number
  rules: string // followed by every agent
  branchPrefix: string // every new task's branch starts with it
  issuesRepo: string // the common issues repository (owner/name)
  projects: Project[] | null
}

export type Keymap = Record<string, string[]>

export interface Settings {
  config: Config
  agents: Agent[]
  context: Record<string, Context>
  keymap: Keymap
  prompts: Prompt[] // what derby adds to agents and fix steps (Settings > Context > Prompts)
  dir: string
}

export interface Prompt {
  id: string
  name: string
  where: string // where derby uses it
  vars: string // what derby replaces in it
  default: string
  text: string // the text in use
}

/** fill replaces each {name} of vars in text. */
export function fill(text: string, vars: Record<string, string>): string {
  return Object.entries(vars).reduce((t, [k, v]) => t.split('{' + k + '}').join(v), text)
}

export interface Step {
  id: string
  taskId: string
  key: string
  agent: string
  n: number
  seq: number
  state: StepState
  guidance: string
  session: string
  sha: string
  verdict: string
  skipped?: boolean // ended by the check or cross button
  modelSet?: string // the model picked in the chat view for this session
  effortSet?: string // likewise its effort
  model: string
  completed: boolean
  startedAt: string
  endedAt: string
  name: string
  live: boolean
  auto: boolean
  artifact: string
  commits: boolean
  diff: boolean // Diff has something to open: its segment's commit, or the uncommitted work
  revert: boolean // after the latest commit: Revert puts the worktree back to its start and removes it and later steps
  paused: boolean // stopped by the user until Resume
  rerun: boolean // Rerun applies: the latest step, of an agent that can be rerun
  skip: boolean // the check and cross buttons apply: the latest step, or one not finished
  gitError: string // why derby's commit, push or PR after it failed
  external: boolean // run by derby, not Claude: the Copilot review step
}

export interface Option {
  agent: string
  n: number
  label: string
}

/** A repository the plan also needs; project is '' until it is a derby project. */
export interface SisterOffer {
  repo: string
  scope: string
  part: string // its line in the plan's overview
  project: string
}

export interface Next {
  options: Option[]
  text: string
  approval: string
  markDone: string
  warning: string // e.g. the plan lists options: name one in the guidance
  elsewhere: string[] // before a step that commits: repositories outside the worktree with uncommitted agent edits
  sisters: SisterOffer[] // repositories the plan also needs, with no sister task yet
  ghost: { name: string; after: string } | null
}

export interface Task {
  id: string
  project: string
  slug: string
  issue: string
  issueRepo: string // the repository holding the issue, when not the project's (a sister task)
  sister: boolean // created as a sister task; the family's task without it started the others
  sisters: { id: string; project: string; part: string }[] // tasks for the other repositories of the same work, with what each does
  part: string // what this task's part of the work does, in a line
  title: string // the issue's title, else the description's first line
  name: string // a few words an agent picked to name the task in lists; '' until then (see taskName)
  description: string
  createdAt: string
  archived: boolean
  archivedAt: string
  pr: number
  prUrl: string
  prState: string
  worktreeRemoved: boolean
  projectName: string
  repo: string
  baseBranch: string
  worktree: string
  worktreeRel: string
  branch: string
  runDir: string
  source: string
  status: { kind: StatusKind; label: string; short: string }
  steps: Step[]
  next: Next
  projectGone: boolean // its project was removed from Settings > Projects
  checks: CICheck[] // CI checks of the open PR
  labels: string[] // labels of the open PR
  draft: boolean // the open PR is a draft (derby opens them so)
  conflicting: boolean // the open PR conflicts with its base: GitHub runs no checks
  feedback: Feedback[] // what reviewers still ask for on the open PR
  rejected: Record<string, { step: string; name: string }> // open threads a step found wrong or unnecessary, by thread id
  behind: number // commits of origin/<base> the branch lacks
  notes: Note[] // the user's comments on the diff, until sent to a step
  todos: Todo[] // the user's todo list for the task
}


export interface Field {
  k: string
  v: string
}

/** A diagram page (archify HTML) in a task's run directory. */
export interface Diagram {
  file: string
  title: string
  step: string // the step whose artifact embeds it, e.g. Implement 2
  mod: string
}

export interface Artifact {
  path: string
  file: string
  exists: boolean
  front: Field[] | null
  body: string
}

export interface Check {
  project: string
  ok: boolean
  label: string
  detail: string
}

export interface TabRef {
  task: string
  step: string
}

/** One CI check on a PR; bucket is pass, fail, pending, skipping or cancel. */
export interface CICheck {
  name: string
  workflow: string
  bucket: string
  link: string
}

/** An open GitHub issue assigned to you, for New task. */
/** A task's issue with its comments (GetTaskIssue); images load through derby. */
export interface TaskIssue {
  number: number
  repo: string
  title: string
  url: string
  state: string
  author: string
  created: string
  labels: string[]
  body: string
  comments: { author: string; created: string; body: string }[]
  offline: string // why GitHub could not be read: body is the copy in ISSUE.md
}

export interface Issue {
  number: number
  title: string
  priority: string // the issue's Priority field (Urgent, High, …), '' when unset
}

export interface Bootstrap {
  settings: Settings
  tasks: Task[]
  tabs: { tabs: TabRef[]; active: number } | null
  os: string
  todos: Todo[] // Others: the todos of no task
}

/** afterOf is the agent id a trigger starts after ("after:review" is review), or "". */
export const afterOf = (trigger: string) => (trigger.startsWith('after:') ? trigger.slice(6) : '')

/** unsuccessful: the step finished, and its agent reported status: failed ("changes": older reviews). */
export const unsuccessful = (s: { state: string; verdict: string }) => s.state === 'done' && (s.verdict === 'failed' || s.verdict === 'changes')

/** An unresolved review thread, or a review's own comment, on the task's PR. */
export interface Feedback {
  id: string
  author: string
  path: string // '' for a review's own comment
  line: number
  outdated: boolean // the code it points at changed since
  resolvedBy: string // who resolved the thread; '' while open (resolved ones are only shown)
  body: string
  url: string
  replies: { author: string; body: string }[]
}

/** The user's comment on lines of a diff, kept with the task until sent to a step. */
export interface Note {
  id: string
  path: string
  side: 'new' | 'old' // which line numbers from and to are
  from: number
  to: number
  code: string // the lines it is about
  text: string
}

/** One item of the user's todo lists: a task's, or Others. */
export interface Todo {
  id: string
  text: string
  done: boolean
  urgent?: boolean
}

/** One part of what an agent's session gets (Settings > Agents > Preview full context). */
export interface ContextPart {
  title: string
  where: string // how it reaches the session
  text: string
}
