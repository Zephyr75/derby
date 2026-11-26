// App state: tasks, settings, tabs and what is selected. Components read it
// and call its methods; Go events keep it current. The instance is wrapped
// in Vue's reactive(), so fields are reactive and getters are derived values.

import { reactive } from 'vue'
import { api, errorText, isDesktop, on } from './api'
import { actionFor } from './keymap'
import { copyText } from './window'
import { modes, themeDef, themes } from './themes'
import { unsuccessful, type Option, type Proc, type Settings, type Step, type TabRef, type Task, type Agent, type Note, type Todo, type StatusKind, type ThemeMode, fill } from './types'

export type Overlay = 'none' | 'palette' | 'scope' | 'new' | 'activity' | 'views' | 'todos'
export type ScopeMode = 'project' | 'worktree' | 'branch'
export type SettingsTab = 'general' | 'agents' | 'context' | 'projects' | 'appearance' | 'shortcuts' | 'about'

export interface Toast {
  id: number
  title: string
  body: string
  error: boolean
}

let toastSeq = 0

/** The id of a task's first step, Task: built in, it shows the task itself (its
 *  issue, comments and instructions). Jockey runs nothing for it, so it is not in task.steps. */
export const taskStepId = (taskId: string) => taskId + '/task'

/** Panel widths: defaults, limits, and the per-viewer saved value. */
export const PANELS = {
  left: { def: 300, min: 220, max: 560 },
  right: { def: 400, min: 300, max: 760 },
} as const
export type PanelSide = keyof typeof PANELS

function savedWidth(side: PanelSide): number {
  try {
    const v = Number(localStorage.getItem('jockey.panel.' + side))
    if (v >= PANELS[side].min && v <= PANELS[side].max) return v
  } catch {
    /* storage unavailable */
  }
  return PANELS[side].def
}

function savedNumber(key: string): number {
  try {
    return Number(localStorage.getItem(key)) || 0
  } catch {
    return 0
  }
}

function savedText(key: string): string {
  try {
    return localStorage.getItem(key) ?? ''
  } catch {
    return ''
  }
}

/** An entry of the Archive's list: a task, or a sister family folded into one row. */
export interface ArchiveEntry {
  key: string
  family: boolean
  tasks: Task[]
}

class Jockey {
  ready = false
  failed = ''
  os = 'linux'
  settings: Settings | null = null
  tasks: Task[] = []
  tabs: TabRef[] = []
  active = -1
  sel = ''
  left: 'tasks' | 'archive' = 'tasks'
  /** The chat's list of the user's messages is open. */
  messages = false
  view: 'board' | 'settings' = 'board'
  settingsTab: SettingsTab = 'general'
  overlay: Overlay = 'none'
  rightTab: 'preview' | 'changes' | 'pr' | 'todos' = 'preview'
  todos: Todo[] = [] // Others: the todos of no task
  hideDone = false // the todo lists hide the done items (Todos tab and pop-up)
  /** Steps whose session another process still runs (the resume guard). */
  elsewhere: Record<string, Proc[]> = {}
  chatTitles: Record<string, string> = {} // Claude Code's title for each step's conversation, by step id (the chat view)
  scope: ScopeMode = 'project'
  /** Projects folded in the left panel, by tab and project ("archive:core"): each tab folds its own. */
  collapsed: Record<string, boolean> = {}
  /** What the Archive lists: a project's archived tasks, the sister families (SISTERS), or '' for everything. */
  archiveProject = savedText('jockey.archive.project')
  guidance = ''
  toasts: Toast[] = []
  systemLight = false
  artifactRev = 0
  diagramRev = 0
  confirmDelete = ''
  leftWidth = savedWidth('left')
  rightWidth = savedWidth('right')
  /** Hidden side panels (the toggles in the status bar). */
  leftHidden = savedNumber('jockey.panel.left.hidden') === 1
  rightHidden = savedNumber('jockey.panel.right.hidden') === 1
  /** Height of the steps list in the right panel; 0 lets it size itself. */
  timelineHeight = savedNumber('jockey.panel.timeline')
  /** Bumped to reconnect the terminal, e.g. after a rerun. */
  termRev = 0

  get projects() {
    return this.settings?.config.projects ?? []
  }
  /** The left panel's groups: every project, then projects removed from
   *  Settings that still have tasks (so they can be found and deleted). */
  get groups(): { id: string; name: string; gone: boolean }[] {
    const out = this.projects.map((p) => ({ id: p.id, name: p.name, gone: false }))
    for (const t of this.tasks) {
      if (!out.some((g) => g.id === t.project)) out.push({ id: t.project, name: t.projectName || t.project, gone: true })
    }
    return out
  }
  get selTask(): Task | null {
    return this.tasks.find((t) => t.id === this.sel) ?? null
  }
  get activeTab(): TabRef | null {
    const tb = this.active >= 0 ? this.tabs[this.active] : undefined
    return tb && tb.task === this.sel ? tb : null
  }
  /** The open tab shows the task's built-in Task step. */
  get onTaskStep(): boolean {
    const tb = this.activeTab
    return !!tb && !!this.selTask && tb.step === taskStepId(tb.task)
  }
  get selStep(): Step | null {
    const tb = this.activeTab
    if (!tb || !this.selTask) return null
    return this.selTask.steps.find((s) => s.id === tb.step) ?? null
  }
  /** themeName and themeMode: the theme picked in Settings > Appearance and
   *  its variant; theme is the data-theme in use. */
  get themeName() {
    return themeDef(this.settings?.config.appearance.theme).id
  }
  get themeMode(): ThemeMode {
    return this.settings?.config.appearance.mode ?? 'dark'
  }
  get theme() {
    const t = themeDef(this.themeName)
    return this.light ? t.light : t.dark
  }
  /** light: the theme in use is a light variant (the mode, or the OS's for system). */
  get light() {
    return this.themeMode === 'light' || (this.themeMode === 'system' && this.systemLight)
  }
  /** sessionView: how the center shows a step's session (Settings > Appearance). */
  get sessionView(): 'terminal' | 'chat' {
    return this.settings?.config.appearance.sessionView ?? 'terminal'
  }
  async setSessionView(view: 'terminal' | 'chat') {
    if (!this.settings || this.sessionView === view) return
    this.settings.config.appearance.sessionView = view
    this.edited()
    await this.attempt('Could not save the session view', () => api.SetSessionView(view))
  }
  /** dark: the theme in use is a dark one (diagrams, mermaid and the logo follow it). */
  get dark() {
    return !this.light
  }

  task(id: string) {
    return this.tasks.find((t) => t.id === id)
  }

  project(id: string) {
    return this.projects.find((p) => p.id === id)
  }

  // ---- startup and events ----

  async init() {
    const mq = window.matchMedia('(prefers-color-scheme: light)')
    this.systemLight = mq.matches
    mq.addEventListener('change', (e) => (this.systemLight = e.matches))
    try {
      const b = await api.Bootstrap()
      this.os = b.os
      this.settings = b.settings
      this.tasks = b.tasks ?? []
      this.todos = b.todos ?? []
      this.collapsed = b.tabs?.collapsed ?? {}
      // One tab per task (older saves have one per step): the step the
      // active tab showed for its task, else the task's last one.
      const saved = b.tabs?.tabs ?? []
      const was = saved[b.tabs?.active ?? -1]
      const tabs: TabRef[] = []
      for (const tb of saved) {
        if (tb.step !== taskStepId(tb.task) && !this.task(tb.task)?.steps.some((s) => s.id === tb.step)) continue
        const cur = tabs.find((x) => x.task === tb.task)
        if (!cur) tabs.push({ task: tb.task, step: tb.step })
        else if (!(was?.task === tb.task && cur.step === was.step)) cur.step = tb.step
      }
      this.tabs = tabs
      this.active = was ? tabs.findIndex((x) => x.task === was.task) : -1
      if (this.active < 0 && this.tabs.length) this.active = 0
      this.sel = this.tabs[this.active]?.task ?? this.tasks.find((t) => !t.archived)?.id ?? this.tasks[0]?.id ?? ''
      const sel = this.task(this.sel)
      if (sel?.archived) this.showInPanel(sel)
      this.trimTabs()
    } catch (e) {
      this.failed = errorText(e)
      return
    }
    if (!isDesktop) this.applyPreviewParams()
    on('task:changed', (t: Task) => this.upsert(t))
    on('task:removed', ({ id }: { id: string }) => this.forget(id))
    on('settings:changed', (s: Settings) => {
      // While the user is typing in Settings, the local copy is newer than
      // the echo of the previous save.
      if (Date.now() - this.lastEdit > 1500) this.settings = s
    })
    on('artifact:changed', (d: { taskId: string; file: string }) => {
      if (d.file.endsWith('.html')) this.diagramRev++ // a diagram page: the embedded ones reload
      else if (d.taskId === this.sel) this.artifactRev++
    })
    on('todos:changed', (ts: Todo[]) => (this.todos = ts ?? []))
    on('toast', (d: { title: string; body: string; error: boolean }) => this.toast(d.title, d.body, d.error))
    on('menu', (what: string) => this.command(what))
    if (isDesktop) {
      // Desktop notifications are only shown while jockey is in the background.
      const report = () => void api.SetFocused(document.hasFocus() && !document.hidden)
      window.addEventListener('focus', report)
      window.addEventListener('blur', report)
      document.addEventListener('visibilitychange', report)
      report()
    }
    window.addEventListener('keydown', (e) => this.onKey(e), true)
    this.ready = true
    void api.WarmTasks() // the parts of each task that need git, now that task:changed is heard
  }

  /** In the browser mock, URL params pick a screen like the preview's
   *  artboards: ?sel=core:4349&left=archive&overlay=palette&q=unlock&
   *  view=settings&tab=appearance&theme=copilot&mode=light&scope=branch&step=plan-1&session=chat */
  private applyPreviewParams() {
    const q = new URLSearchParams(location.search)
    const sel = q.get('sel')
    if (sel && this.task(sel)) {
      const step = q.get('step')
      if (step) this.openStep(sel, `${sel}/${step}`)
      else this.pickTask(sel)
    }
    const left = q.get('left')
    if (left === 'archive' || left === 'tasks') this.left = left
    const theme = themes.find((t) => t.id === q.get('theme'))
    if (theme && this.settings) this.settings.config.appearance.theme = theme.id
    const mode = modes.find((m) => m === q.get('mode'))
    if (mode && this.settings) this.settings.config.appearance.mode = mode
    if (q.get('view') === 'settings') {
      this.view = 'settings'
      this.settingsTab = (q.get('tab') as SettingsTab) || 'general'
    }
    if (q.get('session') === 'chat' && this.settings) this.settings.config.appearance.sessionView = 'chat'
    const scope = q.get('scope')
    if (scope === 'project' || scope === 'worktree' || scope === 'branch') this.scope = scope
    const overlay = q.get('overlay')
    if (overlay === 'palette' || overlay === 'scope' || overlay === 'new' || overlay === 'activity' || overlay === 'todos') this.overlay = overlay
    if (q.get('right') === 'changes' || q.get('right') === 'pr' || q.get('right') === 'todos') this.rightTab = q.get('right') as 'changes' | 'pr' | 'todos'
    this.previewQuery = q.get('q') ?? ''
    this.previewNew = q.get('new') ?? ''
  }
  previewQuery = ''
  previewNew = ''

  private upsert(t: Task) {
    const i = this.tasks.findIndex((x) => x.id === t.id)
    const before = i >= 0 ? this.tasks[i] : null
    if (i >= 0) this.tasks[i] = t
    else this.tasks.push(t)
    // The task's tab showed a step that is gone (a Revert removed it): it
    // shows the task's last step, or closes when none is left.
    const ix = this.tabs.findIndex((tb) => tb.task === t.id)
    const tb = this.tabs[ix]
    if (tb && tb.step !== taskStepId(t.id) && !t.steps.some((s) => s.id === tb.step)) {
      if (t.steps.length) tb.step = t.steps[t.steps.length - 1].id
      else this.dropTab(ix)
      this.persistTabs()
    }
    if (!before || !tb) return
    // A step that started by itself (automatic agents) shows in the task's
    // tab when the tab showed the step before it.
    const known = new Set(before.steps.map((s) => s.id))
    const prev = before.steps[before.steps.length - 1]
    for (const s of t.steps) {
      if (known.has(s.id) || !s.auto) continue
      if (tb.step === taskStepId(t.id) && before.steps.length) continue // reading the task: stay
      if (tb.step !== prev?.id && known.has(tb.step)) continue
      tb.step = s.id
      this.persistTabs()
    }
  }

  /** dropTab removes a tab, keeping the active one (or its neighbour) active. */
  private dropTab(i: number) {
    this.tabs.splice(i, 1)
    if (i < this.active) this.active--
    else if (i === this.active) this.active = Math.min(i, this.tabs.length - 1)
  }

  /** forget drops a deleted task, its tabs and its selection. */
  private forget(id: string) {
    const i = this.tasks.findIndex((t) => t.id === id)
    if (i < 0) return
    const next = this.visibleTasks().find((t) => t.id !== id)
    this.tasks.splice(i, 1)
    const active = this.tabs[this.active]
    this.tabs = this.tabs.filter((tb) => tb.task !== id)
    this.active = active && active.task !== id ? this.tabs.indexOf(active) : Math.min(this.active, this.tabs.length - 1)
    if (this.sel === id) {
      this.sel = ''
      if (next) this.pickTask(next.id)
    }
    this.persistTabs()
  }

  toast(title: string, body = '', error = false) {
    const id = ++toastSeq
    this.toasts.push({ id, title, body, error })
    setTimeout(() => (this.toasts = this.toasts.filter((x) => x.id !== id)), error ? 9000 : 5000)
  }

  async attempt<T>(title: string, fn: () => Promise<T>): Promise<T | undefined> {
    try {
      return await fn()
    } catch (e) {
      this.toast(title, errorText(e), true)
      return undefined
    }
  }

  /** setPanelWidth resizes a side panel (clamped) and remembers it. */
  setPanelWidth(side: PanelSide, w: number) {
    const { min, max } = PANELS[side]
    // Keep at least 360px for the terminal between the panels.
    const other = side === 'left' ? (this.rightHidden ? 0 : this.rightWidth) : this.leftHidden ? 0 : this.leftWidth
    const room = window.innerWidth - other - 360
    const v = Math.round(Math.max(min, Math.min(max, room, w)))
    if (side === 'left') this.leftWidth = v
    else this.rightWidth = v
    try {
      localStorage.setItem('jockey.panel.' + side, String(v))
    } catch {
      /* storage unavailable */
    }
  }

  togglePanel(side: PanelSide) {
    const hidden = side === 'left' ? (this.leftHidden = !this.leftHidden) : (this.rightHidden = !this.rightHidden)
    try {
      localStorage.setItem(`jockey.panel.${side}.hidden`, hidden ? '1' : '0')
    } catch {
      /* storage unavailable */
    }
  }

  setTimelineHeight(h: number) {
    this.timelineHeight = Math.max(0, Math.round(h))
    try {
      localStorage.setItem('jockey.panel.timeline', String(this.timelineHeight))
    } catch {
      /* storage unavailable */
    }
  }

  /** edited marks a local settings edit (see settings:changed). */
  lastEdit = 0
  edited() {
    this.lastEdit = Date.now()
  }

  private persistTimer = 0
  persistTabs() {
    clearTimeout(this.persistTimer)
    this.persistTimer = window.setTimeout(() => {
      void api.SaveTabs(JSON.stringify({ tabs: this.tabs, active: this.active, collapsed: this.collapsed })).catch(() => {})
    }, 300)
  }

  // ---- selection and tabs ----

  /** When each tab was last opened or focused, by task id. */
  private tabUsed: Record<string, number> = {}

  /** trimTabs closes the least recently used tabs above the limit in Settings. */
  trimTabs() {
    const max = Math.max(1, this.settings?.config.maxTabs || 5)
    const keep = this.tabs[this.active]
    if (keep) this.tabUsed[keep.task] = Date.now()
    while (this.tabs.length > max) {
      let old = -1
      this.tabs.forEach((tb, i) => {
        if (tb !== keep && (old < 0 || (this.tabUsed[tb.task] ?? 0) < (this.tabUsed[this.tabs[old].task] ?? 0))) old = i
      })
      if (old < 0) break
      delete this.tabUsed[this.tabs[old].task]
      this.tabs.splice(old, 1)
      if (old < this.active) this.active--
    }
    this.persistTabs()
  }

  /** openStep shows a step in its task's tab: one tab per task, to switch
   *  between recent tasks (the right panel switches between its steps). */
  openStep(taskId: string, stepId: string) {
    let ix = this.tabs.findIndex((tb) => tb.task === taskId)
    if (ix < 0) {
      this.tabs.push({ task: taskId, step: stepId })
      ix = this.tabs.length - 1
    } else this.tabs[ix].step = stepId
    this.active = ix
    this.trimTabs()
    this.sel = taskId
    this.overlay = 'none'
    this.view = 'board'
    const t = this.task(taskId)
    if (t) this.showInPanel(t)
    this.persistTabs()
  }

  /** showInPanel opens the left panel's tab with the task (Tasks or Archive) and makes it visible there. */
  private showInPanel(t: Task) {
    this.left = t.archived ? 'archive' : 'tasks'
    this.reveal(t)
  }

  pickTask(id: string) {
    const t = this.task(id)
    if (!t) return
    this.overlay = 'none'
    this.view = 'board'
    this.showInPanel(t)
    this.confirmDelete = ''
    const ix = this.tabs.findIndex((tb) => tb.task === id)
    if (ix >= 0) {
      this.active = ix
      this.tabUsed[id] = Date.now()
      this.sel = id
      this.persistTabs()
    } else this.openStep(id, t.steps.length ? t.steps[t.steps.length - 1].id : taskStepId(id))
  }

  /** openTask shows the task's Task step. */
  openTask(taskId: string) {
    this.openStep(taskId, taskStepId(taskId))
  }

  /** latestPR: the task's latest Copilot step, which describes its PR. */
  latestPR(t: Task): Step | undefined {
    return [...t.steps].reverse().find((s) => s.external)
  }

  focusTab(i: number) {
    const tb = this.tabs[i]
    if (!tb) return
    this.active = i
    this.tabUsed[tb.task] = Date.now()
    this.sel = tb.task
    this.view = 'board'
    const t = this.task(tb.task)
    if (t) this.showInPanel(t)
    this.persistTabs()
  }

  closeTab(i: number) {
    this.dropTab(i)
    const a = this.active
    if (a >= 0 && this.tabs[a]) this.sel = this.tabs[a].task
    this.persistTabs()
  }

  /** isCollapsed: whether a project or a sister family is folded; the
   *  Archive's families start folded. */
  isCollapsed(projectId: string): boolean {
    return this.collapsed[this.left + ':' + projectId] ?? (this.left === 'archive' && projectId.startsWith(SISTERS))
  }

  toggleCollapsed(projectId: string) {
    this.collapsed[this.left + ':' + projectId] = !this.isCollapsed(projectId)
    this.persistTabs()
  }

  pickArchiveProject(id: string) {
    this.archiveProject = id
    try {
      localStorage.setItem('jockey.archive.project', id)
    } catch {
      /* storage unavailable */
    }
  }

  /** reveal makes an archived task visible in the Archive (unless every
   *  project is listed): lists the sister families for a family's task, and
   *  unfolds its family; else lists its project. */
  private reveal(t: Task) {
    if (!t.archived) return
    const scope = this.archiveScope
    if (this.sisterGroups.has(t.id)) {
      if (scope && scope !== SISTERS) this.pickArchiveProject(SISTERS)
      this.collapsed['archive:' + this.familyKey(t)] = false
    } else if (scope && scope !== t.project) this.pickArchiveProject(t.project)
  }

  /** sisterGroups: the families of tasks linked as sisters (the same work in
   *  several repositories), each task's family by its id. */
  get sisterGroups(): Map<string, string[]> {
    const out = new Map<string, string[]>()
    const byId = new Map(this.tasks.map((t) => [t.id, t]))
    for (const t of this.tasks) {
      if (out.has(t.id) || !t.sisters.length) continue
      // Links go both ways, but follow them all in case one side is missing.
      const ids: string[] = []
      const todo = [t.id]
      while (todo.length) {
        const id = todo.pop()!
        if (ids.includes(id)) continue
        ids.push(id)
        for (const x of byId.get(id)?.sisters ?? []) if (byId.has(x.id)) todo.push(x.id)
      }
      if (ids.length < 2) continue
      for (const id of ids) out.set(id, ids)
    }
    return out
  }

  /** The tasks of the list the left panel shows (Tasks or Archive). */
  private listed(): Task[] {
    return this.tasks.filter((t) => (this.left === 'archive' ? t.archived : !t.archived))
  }
  private newest(a: Task, b: Task) {
    const key = this.left === 'archive' ? 'archivedAt' : 'createdAt'
    const time = (iso: string) => Date.parse(iso) || 0 // Go's times vary in offset and precision: compare them as dates
    return time(b[key]) - time(a[key])
  }

  /** panelFamilies: the sister families in the list, with the big sister
   *  (the task that started the others) first. In Tasks each is shown
   *  first in its own category; in the Archive it is folded into one entry
   *  (archiveEntries). A family keeps its category with a single task in the
   *  list, e.g. once its sisters' PRs were merged and they moved to the
   *  Archive. */
  panelFamilies(): Task[][] {
    const listed = this.listed()
    const groups = this.sisterGroups
    const seen = new Set<string[]>()
    const out: Task[][] = []
    for (const t of listed.sort((a, b) => this.newest(a, b))) {
      const ids = groups.get(t.id)
      if (!ids || seen.has(ids)) continue
      seen.add(ids)
      const members = listed.filter((x) => ids.includes(x.id))
      out.push(members.sort((a, b) => Number(a.sister) - Number(b.sister) || -this.newest(a, b)))
    }
    return out
  }

  /** A project's tasks in the left panel's tab, newest first: tasks by
   *  creation, archives by when they were closed (archived). */
  panelTasks(projectId: string): Task[] {
    const inFamily = new Set(this.panelFamilies().flat().map((t) => t.id))
    return this.listed()
      .filter((t) => t.project === projectId && !inFamily.has(t.id))
      .sort((a, b) => this.newest(a, b))
  }

  /** familyKey: a sister family's category, collapsed like a project; the
   *  same whichever of its tasks are listed. */
  familyKey(t: Task): string {
    return SISTERS + [...(this.sisterGroups.get(t.id) ?? [t.id])].sort()[0]
  }

  /** The tasks in the left panel, in display order: the sister families, then each project's
   *  (in the Archive, its entries' unfolded tasks). */
  visibleTasks(): Task[] {
    if (this.left === 'archive') return this.archiveEntries().flatMap((e) => (e.family && this.isCollapsed(e.key) ? [] : e.tasks))
    const families = this.panelFamilies().filter((f) => !this.isCollapsed(this.familyKey(f[0]))).flat()
    return [...families, ...this.groups.filter((p) => !this.isCollapsed(p.id)).flatMap((p) => this.panelTasks(p.id))]
  }

  /** What the Archive lists: the picked project while it still exists, the
   *  sister families (SISTERS), else '' (everything). */
  get archiveScope(): string {
    const p = this.archiveProject
    return p === SISTERS || this.groups.some((g) => g.id === p) ? p : ''
  }

  /** archiveEntries: the Archive's list, newest first: the picked project's
   *  single archived tasks, or the sister families (several projects' work,
   *  so a pick of their own), each folded into one entry; or both. */
  archiveEntries(): ArchiveEntry[] {
    const p = this.archiveScope
    const time = (t: Task) => Date.parse(t.archivedAt) || 0
    const all = this.panelFamilies()
    const families = !p || p === SISTERS ? all.map((tasks) => ({ key: this.familyKey(tasks[0]), family: true, tasks })) : []
    const inFamily = new Set(all.flat().map((t) => t.id))
    const singles = this.listed()
      .filter((t) => p !== SISTERS && (!p || t.project === p) && !inFamily.has(t.id))
      .map((t) => ({ key: t.id, family: false, tasks: [t] }))
    const newest = (e: ArchiveEntry) => Math.max(...e.tasks.map(time))
    return [...families, ...singles].sort((a, b) => newest(b) - newest(a))
  }

  moveTask(delta: number) {
    const list = this.visibleTasks()
    if (!list.length) return
    const i = list.findIndex((t) => t.id === this.sel)
    const next = list[Math.max(0, Math.min(list.length - 1, i < 0 ? 0 : i + delta))]
    this.pickTask(next.id)
  }

  // ---- actions on the selected task ----

  /** The step whose Revert waits for its second click. */
  confirmRevert = ''

  /** revertStep puts the worktree back to the step's start and removes it
   *  and later steps; the first click only asks. */
  async revertStep(step: Step) {
    if (this.confirmRevert !== step.id) {
      this.confirmRevert = step.id
      return
    }
    this.confirmRevert = ''
    if (await this.attempt('Could not revert ' + step.name, () => api.RevertStep(step.id).then(() => true))) this.toast('Reverted ' + step.name, 'The worktree is back to how it was when it started.')
  }

  /** skipStep ends a step at once as succeeded (ok) or failed, to move on. */
  skipStep(step: Step, ok: boolean) {
    return this.attempt('Could not skip ' + step.name, () => api.SkipStep(step.id, ok))
  }

  /** rerunStep runs a finished step again in its own session. */
  async rerunStep(step: Step) {
    const ok = await this.attempt('Could not rerun ' + step.name, () => api.RerunStep(step.id).then(() => true))
    if (!ok) return
    this.termRev++
    this.openStep(step.taskId, step.id)
  }

  /** The step whose artifact the artifact pop-up shows; '' when closed. */
  artifactOf = ''
  /** The step and file shown in the changes pop-up; null when closed. */
  diffOf: { step: string; path: string; auto?: boolean } | null = null

  /** The diagram page shown full size; null when closed. */
  diagramOf: { task: string; file: string } | null = null

  /** Opens a task's diagram page full size, over the other pop-ups. */
  openDiagram(task: string, file: string) {
    this.diagramOf = { task, file }
    this.overlay = 'views'
  }
  closeDiagram() {
    this.diagramOf = null
    if (!this.artifactOf && !this.diffOf) this.overlay = 'none'
  }

  /** Opens the artifact pop-up alone. */
  openArtifactPopup(step: string) {
    this.artifactOf = step
    this.diffOf = null
    this.overlay = 'views'
  }

  /** Opens the changes pop-up alone, on a file. */
  openDiffPopup(step: string, path: string) {
    this.diffOf = { step, path }
    this.artifactOf = ''
    this.overlay = 'views'
  }

  /** Opens the artifact pop-up beside the changes, or the reverse ('' path: the first file). */
  showArtifactToo(step: string) {
    this.artifactOf = step
  }
  showDiffToo(step: string) {
    this.diffOf = { step, path: '', auto: true } // auto: another step's changes when this one has none
  }

  closeArtifactPopup() {
    this.artifactOf = ''
    if (!this.diffOf) this.overlay = 'none'
  }
  closeDiffPopup() {
    this.diffOf = null
    if (!this.artifactOf) this.overlay = 'none'
  }
  closeViews() {
    this.artifactOf = ''
    this.diffOf = null
    this.diagramOf = null
    this.overlay = 'none'
  }

  /** run starts opt's step with the guidance typed in Next step, or the given one. */
  async run(opt: Option, taskId = this.sel, given?: string) {
    const guidance = given ?? this.guidance
    const step = await this.attempt('Could not start ' + opt.label, () => api.StartStep(taskId, opt.agent, guidance, opt.n))
    if (!step) return
    if (given === undefined) this.guidance = ''
    this.openStep(taskId, step.id)
  }

  runPrimary() {
    const t = this.selTask
    const opt = t?.next.options[0]
    if (t && opt && !t.archived) void this.run(opt)
  }

  /** prReady: '' without an open PR, 'draft' while jockey's draft PR waits
   *  for the Mark ready button, 'ready' once it is ready for review. */
  prReady(t: Task | null): '' | 'draft' | 'ready' {
    if (!t || t.archived || !t.pr || t.prState !== 'open') return ''
    return t.draft ? 'draft' : 'ready'
  }

  /** prBadge: the task's PR at a glance, for the left panel: its CI (label)
   *  and where the PR stands (state: Draft, Ready, Closed, Merged, then In QA
   *  and In Prod as the merge reaches those branches). */
  prBadge(t: Task): { kind: StatusKind; label: string; state: string } | null {
    if (!t.pr) return null
    const b = new Set(t.checks.map((c) => c.bucket))
    const ci: { kind: StatusKind; label: string } = t.conflicting
      ? { kind: 'failed', label: 'Conflict' }
      : b.has('fail') || b.has('cancel')
        ? { kind: 'failed', label: 'Failed' }
        : b.has('pending')
          ? { kind: 'running', label: 'Running' }
          : { kind: b.size ? 'done' : 'idle', label: b.size ? 'Passed' : 'No CI' }
    if (t.prState === 'merged') {
      const at = t.release && t.release !== t.baseBranch ? t.release : ''
      return { ...ci, kind: 'done', state: !at ? 'Merged' : /^(master|main)$/.test(at) ? 'In Prod' : at === 'qa' ? 'In QA' : 'In ' + at }
    }
    if (t.prState && t.prState !== 'open') return { ...ci, kind: 'idle', state: 'Closed' }
    return { ...ci, state: t.draft ? 'Draft' : 'Ready' }
  }

  /** fixers: the agents Copilot offers when there is something to fix (Implement, Discuss by default); the first does the fixes. */
  fixers(_t: Task): Agent[] {
    const ags = this.settings?.agents ?? []
    const ids = ags.find((a) => a.builtin)?.onFail ?? []
    return ids.map((id) => ags.find((a) => a.id === id)).filter((a): a is Agent => !!a && !a.builtin)
  }

  /** busy: a step of the task is working or waiting for the user (a waiting Copilot step aside). */
  busy(t: Task): boolean {
    return t.steps.some((s) => !s.external && (s.state === 'running' || s.state === 'idle' || s.state === 'approval'))
  }

  /** canFix: jockey can start a fix step on the task now. */
  canFix(t: Task): boolean {
    return !t.archived && !t.worktreeRemoved && !!this.fixers(t).length && !this.busy(t)
  }

  /** fixWith starts agent (the first fixer by default) on the task with the given guidance. */
  async fixWith(t: Task, guidance: string, agent = this.fixers(t)[0]): Promise<boolean> {
    if (!agent) return false
    const step = await this.attempt('Could not start ' + agent.name, () => api.StartStep(t.id, agent.id, guidance, 0))
    if (!step) return false
    this.openStep(t.id, step.id)
    return true
  }

  /** todoAgent: the agent a todo's play button starts, the first fixer (Implement by default). */
  todoAgent(t: Task): Agent | undefined {
    return this.fixers(t)[0]
  }

  /** todoBlocked: why a todo's play button can't start a step on the task now, or ''. */
  todoBlocked(t: Task): string {
    if (t.archived) return 'The task is archived'
    if (t.worktreeRemoved) return 'The worktree was removed'
    if (!this.todoAgent(t)) return 'No Implement agent'
    if (this.busy(t)) return 'A step is busy'
    return ''
  }

  /** runTodo starts the todo agent on the task, with the todo as guidance. */
  runTodo(t: Task, todo: Todo) {
    return this.fixWith(t, this.prompt('todo', { todo: todo.text }) || todo.text, this.todoAgent(t))
  }

  /** setNotes saves the task's comments on the diff. */
  async setNotes(t: Task, notes: Note[]) {
    t.notes = notes
    await this.attempt('Could not save the comments', () => api.SetNotes(t.id, JSON.parse(JSON.stringify(notes))))
  }

  /** setTodos saves a task's todo list, or Others when task is null. */
  async setTodos(t: Task | null, todos: Todo[]) {
    if (t) t.todos = todos
    else this.todos = todos
    await this.attempt('Could not save the todos', () => api.SetTodos(t?.id ?? '', JSON.parse(JSON.stringify(todos))))
  }

  /** prompt is one of jockey's prompts (Settings > Context > Prompts), filled with vars. */
  prompt(id: string, vars: Record<string, string>): string {
    return fill(this.settings?.prompts.find((p) => p.id === id)?.text ?? '', vars)
  }

  async markReady(t: Task) {
    if (!(await this.attempt('Could not mark the PR ready', () => api.MarkPRReady(t.id).then(() => true)))) return
    t.draft = false
    if (!t.labels.includes('ready')) t.labels.push('ready')
  }

  async openWorktree() {
    const t = this.selTask
    if (!t) return
    const lines = await this.attempt('Could not open Zed', () => api.OpenInZed(t.id, ''))
    if (lines) this.toast('Opening worktree in Zed', lines.join('\n'))
  }

  async openDiff(step: Step | null = this.selStep) {
    const t = this.selTask
    if (!t || !step) return
    const lines = await this.attempt('Could not open the diff', () => api.OpenInZed(t.id, step.id))
    if (lines) this.toast('Opening ' + step.name + "'s changes in Zed", lines.join('\n'))
  }

  // ---- a step's Claude Code session, e.g. to continue it in Zed ----

  /** copy puts text on the clipboard; the toast names what, or shows the text. */
  async copy(text: string, what = text) {
    if (await copyText(text)) this.toast('Copied', what)
    else this.toast('Could not copy', what, true)
  }

  copyResume(step: Step) {
    return this.copy(resumeCommand(step))
  }

  stopStep(step: Step) {
    return this.attempt('Could not stop Claude', () => api.StopStep(step.id))
  }

  /** Stop Claude in jockey and copy the command to continue it elsewhere. */
  async stopAndCopy(step: Step) {
    await this.copyResume(step)
    await this.stopStep(step)
  }

  /** Resume, unless another process (e.g. claude --resume in Zed) still
   *  runs the session: then the terminal shows who, and Resume anyway. */
  async resumeStep(step: Step) {
    if (!(await this.restoreFor(step))) return
    try {
      await api.ResumeStep(step.id)
      delete this.elsewhere[step.id]
    } catch (e) {
      const procs = await api.SessionElsewhere(step.id).catch(() => [])
      if (procs.length) this.elsewhere[step.id] = procs
      else this.toast('Could not resume', errorText(e), true)
    }
  }

  /** restoreTask moves an archived task back to Tasks. */
  async restoreTask(t: Task): Promise<boolean> {
    return !!(await this.attempt('Could not restore', () => api.RestoreTask(t.id).then(() => true)))
  }

  /** restoreFor restores the step's task first when it is archived, so the
   *  step can resume; the task list follows it to Tasks. */
  async restoreFor(step: Step): Promise<boolean> {
    const t = this.task(step.taskId)
    if (!t?.archived) return true
    if (!(await this.restoreTask(t))) return false
    t.archived = false
    this.left = 'tasks'
    return true
  }

  async resumeAnyway(step: Step) {
    delete this.elsewhere[step.id]
    await this.attempt('Could not resume', () => api.ResumeStepAnyway(step.id))
  }

  // ---- overlays and commands ----

  openOverlay(o: Overlay, scope?: ScopeMode) {
    if (scope) this.scope = scope
    this.overlay = o
  }

  command(action: string) {
    if (action.startsWith('task:')) {
      // A click on a desktop notification.
      this.overlay = 'none'
      this.view = 'board'
      return this.pickTask(action.slice(5))
    }
    const board = this.view === 'board'
    switch (action) {
      case 'palette':
        return this.openOverlay(this.overlay === 'palette' ? 'none' : 'palette')
      case 'toggleLeft':
        return this.togglePanel('left')
      case 'toggleRight':
        return this.togglePanel('right')
      case 'newTask':
        return this.openOverlay('new')
      case 'messages':
        if (board) this.messages = !this.messages
        return
      case 'todos':
        return this.openOverlay(this.overlay === 'todos' ? 'none' : 'todos')
      case 'settings':
        this.overlay = 'none'
        this.view = this.view === 'settings' ? 'board' : 'settings'
        return
      case 'close':
        if (this.diagramOf) this.closeDiagram() // back to the pop-ups under it
        else if (this.overlay !== 'none') this.overlay = 'none'
        else if (this.view === 'settings') this.view = 'board'
        return
    }
    if (!board || this.overlay !== 'none') return
    switch (action) {
      case 'scope':
        return this.openOverlay('scope', 'project')
      case 'runPrimary':
        return this.runPrimary()
      case 'toggleArchive':
        this.left = this.left === 'tasks' ? 'archive' : 'tasks'
        return
      case 'nextTask':
        return this.moveTask(1)
      case 'prevTask':
        return this.moveTask(-1)
      case 'closeTab':
        if (this.activeTab) this.closeTab(this.active)
        return
      case 'nextTab':
      case 'prevTab':
        if (this.tabs.length) this.focusTab((Math.max(this.active, 0) + (action === 'nextTab' ? 1 : this.tabs.length - 1)) % this.tabs.length)
        return
      case 'openWorktree':
        return void this.openWorktree()
      case 'openDiff':
        return void this.openDiff()
    }
    const m = /^tab([1-9])$/.exec(action)
    if (m) this.focusTab(Number(m[1]) - 1)
  }

  private onKey(e: KeyboardEvent) {
    if (!this.settings) return
    // Ctrl+<letter> in a terminal is the terminal's (Ctrl+F moves the cursor,
    // Ctrl+N recalls history): jockey's shortcuts take it everywhere else.
    const inTerm = (e.target as HTMLElement | null)?.closest?.('.xterm')
    if (inTerm && e.ctrlKey && !e.altKey && !e.metaKey && !e.shiftKey && /^[a-z]$/i.test(e.key)) return
    const action = actionFor(this.settings.keymap, e)
    if (!action) return
    // ⌘⏎ inside a popup belongs to the popup (Create and run Plan, open in Zed).
    if (this.overlay !== 'none' && action === 'runPrimary') return
    // Escape in an editor that handles it (a diff comment) cancels the edit, not the popup.
    if (action === 'close' && (e.target as HTMLElement | null)?.closest?.('[data-own-escape]')) return
    // Escape with nothing to close belongs to the focused element, e.g. Claude Code in the terminal.
    if (action === 'close' && this.overlay === 'none' && this.view === 'board') return
    e.preventDefault()
    e.stopPropagation()
    this.command(action)
  }
}

/** resumeCommand continues a step's session elsewhere. */
/** How a task shows in lists, tabs and title bars: its agent-given name, else its title. */
export const taskName = (t: Pick<Task, 'name' | 'title'>) => t.name || t.title
export const resumeCommand = (step: Step) => `claude --resume ${step.session}`

/** The prefix of a sister family's category in the left panel. */
export const SISTERS = ':sisters:' // no project id has a colon

export const jockey = reactive(new Jockey())
