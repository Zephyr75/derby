<script setup lang="ts">
// A step's Claude Code session as a chat (Settings > Appearance > Session
// view), after Zed's agent panel (its thread view: spacing, cards, folding
// and controls follow Zed's source): the conversation read from Claude Code's
// transcript, and a message editor that types into the same session as the
// terminal.
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { api, errorText } from '../lib/api'
import { jockey, resumeCommand } from '../lib/state'
import { render, renderInline } from '../lib/markdown'
import { entries, toolView, diffStat, questions, todos, response, rawInput, commands, terminalCommand, commandLabel, commandResult, type ToolCall } from '../lib/chat'
import type { ChatItem, SessionStatus, Skill, Step } from '../lib/types'
import Icon from './ui/Icon.vue'
import Diff from './Diff.vue'
import Working from './ui/Working.vue'
import MessagePicker, { type Mark } from './MessagePicker.vue'

const props = defineProps<{ step: Step }>()
const task = computed(() => jockey.task(props.step.taskId))

// ---- the conversation, followed while the session runs ----
const items = ref<ChatItem[]>([])
const status = ref<SessionStatus | null>(null)
const skills = ref<Skill[]>([]) // the session's, for "/" (its transcript lists them once)
const found = ref(true)
const loaded = ref(false)
// The folder Claude Code asks to trust, before it starts (its first run in a new worktree).
const trust = ref('')
let offset = 0
let timer: ReturnType<typeof setTimeout> | undefined
let seq = 0
const thread = ref<HTMLDivElement>()
const open = ref<Record<string, boolean>>({}) // tool cards opened or closed by a click

async function load(id: string, mine: number) {
  try {
    const c = await api.StepChat(id, offset)
    if (mine !== seq) return
    const stick = atBottom()
    if (c.reset) items.value = []
    if (c.items.length) items.value.push(...c.items)
    offset = c.offset
    found.value = c.found
    trust.value = c.trust || ''
    if (c.title) jockey.chatTitles[id] = c.title
    if (c.skills?.length) skills.value = c.skills
    // The statusline's report wins; from the transcript, keep what was read before.
    const st = c.status
    status.value = c.live || !status.value ? st : { ...st, model: st.model || status.value.model, effort: st.effort || status.value.effort, contextUsed: st.contextUsed || status.value.contextUsed }
    if (!loaded.value || stick) scrollDown(!loaded.value)
    loaded.value = true
  } catch {
    /* the next poll retries */
  }
  if (mine === seq) timer = setTimeout(() => load(id, mine), props.step.live ? 700 : 2500)
}
function start(id: string) {
  clearTimeout(timer)
  seq++
  items.value = []
  status.value = null
  offset = 0
  loaded.value = false
  trust.value = ''
  open.value = {}
  void load(id, seq)
}
watch(() => props.step.id, start, { immediate: true })
onBeforeUnmount(() => {
  clearTimeout(timer)
  seq++
})

function atBottom() {
  const el = thread.value
  return !el || el.scrollHeight - el.scrollTop - el.clientHeight < 48
}
function scrollDown(now = false) {
  void nextTick(() => thread.value?.scrollTo({ top: thread.value.scrollHeight, behavior: now ? 'auto' : 'smooth' }))
}

const list = computed(() => entries(items.value))
// What the user sent, for the picker in the thread's corner.
const marks = computed<Mark[]>(() =>
  list.value.flatMap((e) =>
    e.kind === 'item' && (e.item.kind === 'user' || e.item.kind === 'command' || e.item.kind === 'bash')
      ? [{ turn: e.turn, text: e.item.kind === 'bash' ? '! ' + e.item.text : e.item.text, command: e.item.kind === 'command' ? e.item.tool : '', at: e.item.at ?? '' }]
      : [],
  ),
)
const html = (text: string) => render(text, { task: props.step.taskId, dark: jockey.dark })
const view = (t: ToolCall) => toolView(t.call, task.value?.worktree ?? '', task.value?.runDir ?? '')

// Everything starts folded, to read the answers at a glance; a click unfolds.
const isOpen = (id: string) => !!open.value[id]
const toggle = (id: string) => (open.value[id] = !open.value[id])
// A long message (a paste) shows its first lines; a click shows the rest.
function fold(id: string, text: string) {
  const lines = text.split('\n')
  if (lines.length <= 12) return null
  return { text: isOpen(id) ? text : lines.slice(0, 6).join('\n'), more: lines.length - 6 }
}

// ---- the session's state ----
// An Esc in Claude's terminal sends no hook (Jockey learns of it a minute
// later): its transcript ends on the interruption.
const interrupted = computed(() => {
  const last = items.value[items.value.length - 1]
  return last?.kind === 'note' && last.text === 'Interrupted'
})
const running = computed(() => props.step.live && props.step.state === 'running' && !interrupted.value)
const approval = computed(() => props.step.live && props.step.state === 'approval' && !interrupted.value)
// The tool call Claude waits on: the last one without a result.
const pending = computed(() => {
  const l = list.value
  for (let i = l.length - 1; i >= 0; i--) {
    const e = l[i]
    if (e.kind === 'tool') return e.tool.result ? null : e.tool
  }
  return null
})
const busy = (t: ToolCall) => !t.result && props.step.live && (running.value || approval.value)
// Claude's question form waits for the user too (the step is idle, not approval).
const asking = computed(() => props.step.live && !running.value && pending.value?.call.tool === 'AskUserQuestion')
const others = computed(() => (props.step.live ? undefined : jockey.elsewhere[props.step.id]))
const paused = computed(() => props.step.paused && !props.step.live)
const canRestore = computed(() => !!task.value && !task.value.worktreeRemoved && !task.value.projectGone)
const blocked = computed(() => !!others.value || paused.value || (!!task.value?.archived && !canRestore.value))

// ---- the message box ----
const draft = ref('')
const box = ref<HTMLTextAreaElement>()
const sending = ref(false)
// /btw questions and their answers: shown here only, kept out of the session.
interface Btw { id: number; q: string; a: string; error: boolean; loading: boolean }
const btws = ref<Btw[]>([])
let btwId = 0
watch(() => props.step.id, () => (btws.value = []))
async function btw(q: string) {
  const b = reactive<Btw>({ id: ++btwId, q, a: '', error: false, loading: true })
  btws.value.push(b)
  draft.value = ''
  pastes.value = {}
  scrollDown()
  try {
    b.a = await api.SideQuestion(props.step.id, q)
  } catch (e) {
    b.a = errorText(e)
    b.error = true
  } finally {
    b.loading = false
    scrollDown()
  }
}
function grow() {
  const el = box.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 300) + 'px'
}
watch(draft, () => void nextTick(grow))

// A long paste shows as "[42 lines pasted]" in the box, as in Claude Code's
// prompt, and goes out in full on send. Backspace or Delete removes it whole.
const pastes = ref<Record<string, string>>({})
function paste(e: ClipboardEvent) {
  const text = (e.clipboardData?.getData('text/plain') ?? '').replace(/\r\n?/g, '\n')
  const lines = text.replace(/\n$/, '').split('\n').length
  if (lines < 10 && text.length < 1000) return
  e.preventDefault()
  for (const k of Object.keys(pastes.value)) if (!draft.value.includes(k)) delete pastes.value[k]
  const what = lines > 1 ? `${lines} lines` : `${text.length.toLocaleString('en-US')} characters`
  let token = `[${what} pasted]`
  for (let n = 2; token in pastes.value; n++) token = `[${what} pasted #${n}]`
  pastes.value[token] = text
  const el = e.target as HTMLTextAreaElement
  // insertText keeps the paste in the box's undo history.
  if (document.execCommand('insertText', false, token)) return
  const [a, b] = [el.selectionStart, el.selectionEnd]
  draft.value = draft.value.slice(0, a) + token + draft.value.slice(b)
  void nextTick(() => {
    el.setSelectionRange(a + token.length, a + token.length)
    sync()
  })
}
function expand(s: string) {
  const keys = Object.keys(pastes.value)
  if (!keys.length) return s
  const re = new RegExp(keys.map((k) => k.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|'), 'g')
  return s.replace(re, (k) => pastes.value[k])
}
// selectPaste selects the paste next to the caret, for Backspace or Delete to remove.
function selectPaste(back: boolean) {
  const el = box.value
  if (!el || el.selectionStart !== el.selectionEnd) return
  const at = el.selectionStart
  for (const k of Object.keys(pastes.value)) {
    if (back ? draft.value.slice(0, at).endsWith(k) : draft.value.startsWith(k, at)) {
      el.setSelectionRange(back ? at - k.length : at, back ? at : at + k.length)
      return
    }
  }
}

async function send() {
  const text = expand(draft.value).trim()
  if (!text || sending.value || blocked.value) return
  const side = /^\/btw(?:\s+([\s\S]*))?$/.exec(text)
  if (side) return side[1]?.trim() ? void btw(side[1].trim()) : jockey.toast('Add your question', 'Type it after /btw, e.g. /btw which file holds the retry budget?')
  // Typed into the trust dialog, it would answer it (Enter on "No" exits).
  if (trust.value) return jockey.toast('Trust the folder first', 'Claude Code starts once the folder is trusted; then send your message.')
  sending.value = true
  try {
    if (!(await jockey.restoreFor(props.step))) return
    await api.SendToStep(props.step.id, text)
    // Commands like /context or /usage show in a panel of Claude Code's own: there.
    if (terminalCommand(text)) {
      jockey.toast(text.split(/\s/)[0] + ' opens in the terminal', 'Switch back with the chat button in the toolbar')
      void jockey.setSessionView('terminal')
    }
    draft.value = ''
    pastes.value = {}
    scrollDown()
  } catch (e) {
    const procs = await api.SessionElsewhere(props.step.id).catch(() => [])
    if (procs.length) jockey.elsewhere[props.step.id] = procs
    else jockey.toast('Could not send', errorText(e), true)
  } finally {
    sending.value = false
  }
}
function key(e: KeyboardEvent) {
  if (sugs.value.length && !e.isComposing) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault()
      const n = sugs.value.length
      sugIx.value = (sugIx.value + (e.key === 'ArrowDown' ? 1 : n - 1)) % n
      return
    }
    if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault()
      accept(sugs.value[sugIx.value])
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      dismissed.value = draft.value.slice(0, caret.value)
      return
    }
  }
  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
    e.preventDefault()
    void send()
  }
  if ((e.key === 'Backspace' || e.key === 'Delete') && !e.isComposing && !e.altKey && !e.metaKey) selectPaste(e.key === 'Backspace')
}

// ---- "/" for commands, "@" for files, "!" for bash mode, as Claude Code's prompt ----
const caret = ref(0)
const sync = () => (caret.value = box.value?.selectionStart ?? draft.value.length)
// The word being completed: "/" at the start of the message, "@" after a space.
const trigger = computed(() => {
  const before = draft.value.slice(0, caret.value)
  if (/^\/[^\s]*$/.test(before)) return { kind: 'command' as const, start: 0, query: before.slice(1) }
  const m = /(^|\s)@("?)([^\s"]*)$/.exec(before)
  if (m) return { kind: 'file' as const, start: before.length - m[3].length - m[2].length - 1, query: m[3] }
  return null
})
const dismissed = ref('') // Esc closes the list until the word changes
const files = ref<string[]>([])
let fseq = 0
let ftimer: ReturnType<typeof setTimeout> | undefined
watch(
  () => (trigger.value?.kind === 'file' ? trigger.value.query : null),
  (q) => {
    clearTimeout(ftimer)
    if (q === null) return
    const mine = ++fseq
    ftimer = setTimeout(() => {
      api.StepFiles(props.step.id, q).then(
        (fs) => mine === fseq && (files.value = fs),
        () => mine === fseq && (files.value = []),
      )
    }, 120)
  },
)
interface Sug {
  label: string
  detail: string
  insert: string // replaces the word, trigger included
  more: boolean // a folder: the list stays open to go into it
}
const sugs = computed<Sug[]>(() => {
  const t = trigger.value
  if (!t || blocked.value || dismissed.value === draft.value.slice(0, caret.value)) return []
  if (t.kind === 'command') return commands(skills.value, t.query).map((c) => ({ label: '/' + c.name, detail: c.description + (c.terminal ? ' (opens in the terminal)' : ''), insert: '/' + c.name + ' ', more: false }))
  return files.value.map((f) => ({ label: f, detail: f.endsWith('/') ? 'folder' : '', insert: (/\s/.test(f) ? `@"${f}"` : '@' + f) + (f.endsWith('/') ? '' : ' '), more: f.endsWith('/') }))
})
const sugIx = ref(0)
watch(sugs, () => (sugIx.value = 0))
function accept(s: Sug | undefined) {
  const t = trigger.value
  if (!s || !t) return
  const end = caret.value
  draft.value = draft.value.slice(0, t.start) + s.insert + draft.value.slice(end)
  const at = t.start + s.insert.length
  void nextTick(() => {
    box.value?.focus()
    box.value?.setSelectionRange(at, at)
    caret.value = at
  })
}
const bash = computed(() => draft.value.startsWith('!'))
const interrupt = () => jockey.attempt('Could not interrupt Claude', () => api.InterruptStep(props.step.id))
const allow = () => jockey.attempt('Could not allow it', () => api.AllowOnce(props.step.id))
const trusting = ref(false)
async function trustFolder() {
  trusting.value = true
  if (await jockey.attempt('Could not trust the folder', () => api.TrustFolder(props.step.id).then(() => true))) trust.value = ''
  trusting.value = false
}
const placeholder = computed(() =>
  props.step.live ? (running.value ? 'Queue a message for Claude…' : 'Message Claude… / for commands, @ for files, ! for bash (Enter to send, Shift+Enter for a new line)') : task.value?.archived ? 'Send to restore the task and resume this session…' : 'Send a message to resume this session…',
)
const agentName = computed(() => jockey.settings?.agents.find((a) => a.id === props.step.agent)?.name ?? props.step.agent)
const model = computed(() => (props.step.model ? props.step.model[0].toUpperCase() + props.step.model.slice(1) : ''))
// ---- each turn's controls, as Zed's: copy its answer, back to its message, to the top ----
const copyTurn = (turn: number) => jockey.copy(response(items.value, turn), 'the agent response')
const toUser = (turn: number) => thread.value?.querySelector<HTMLElement>(`[data-turn="${turn}"]`)?.scrollIntoView({ block: 'start', behavior: 'smooth' })
const toTop = () => thread.value?.scrollTo({ top: 0, behavior: 'smooth' })
const title = (t: ToolCall) => renderInline(view(t).title)
// The tool call waiting for approval is a card with its content and the permission buttons.
const confirming = (t: ToolCall) => (approval.value || asking.value) && pending.value === t

// Claude's question form, answered in the chat: the options picked (from 1)
// and the typed answers, by question. Each new form starts with the
// recommended option picked (else the first), for you to confirm or change.
const picks = ref<Record<number, number[]>>({})
const typed = ref<Record<number, string>>({})
const answering = ref(false)
function recommended(t: ToolCall | undefined) {
  const out: Record<number, number[]> = {}
  if (t?.call.tool !== 'AskUserQuestion') return out
  questions(t.call).forEach((q, i) => {
    if (q.options.length) out[i] = [1 + Math.max(0, q.options.findIndex((o) => /\(recommended\)/i.test(o.label)))]
  })
  return out
}
watch(
  () => pending.value?.call.id,
  () => {
    picks.value = recommended(pending.value ?? undefined)
    typed.value = {}
  },
  { immediate: true },
)
const picked = (q: number, k: number) => (picks.value[q] ?? []).includes(k)
function pickOption(q: number, k: number, multi: boolean) {
  const cur = picks.value[q] ?? []
  picks.value[q] = multi ? (cur.includes(k) ? cur.filter((x) => x !== k) : [...cur, k].sort()) : [k]
  if (!multi) typed.value[q] = ''
}
const answered = (t: ToolCall) => questions(t.call).every((q, i) => (picks.value[i]?.length ?? 0) > 0 || (!q.multi && !!typed.value[i]?.trim()))
async function submitAnswers(t: ToolCall) {
  const answers = questions(t.call).map((q, i) => ({ multi: q.multi, count: q.options.length, picks: typed.value[i]?.trim() ? [] : picks.value[i] ?? [], other: q.multi ? '' : typed.value[i] ?? '' }))
  answering.value = true
  await jockey.attempt('Could not answer', () => api.AnswerQuestions(props.step.id, answers))
  answering.value = false
}

// ---- the editor's toolbar: attach files, context, plan usage, model, effort ----
async function attach() {
  const files = await jockey.attempt('Could not open the file picker', () => api.ChooseFiles('Attach files to your message'))
  if (!files?.length) return
  const paths = await jockey.attempt('Could not attach the files', () => api.AttachToStep(props.step.id, files))
  if (!paths?.length) return
  const refs = paths.map((p) => (/\s/.test(p) ? `@"${p}"` : '@' + p)).join(' ')
  draft.value = (draft.value.trimEnd() ? draft.value.trimEnd() + ' ' : '') + refs + ' '
  box.value?.focus()
}
const k = (n: number) => (n >= 1000 ? `${Math.round(n / 100) / 10}k` : String(n))
const context = computed(() => {
  const s = status.value
  if (!s?.contextUsed) return null
  const size = s.contextSize || 200000
  const ratio = Math.min(s.contextUsed / size, 1)
  return { ratio, warn: ratio >= 0.85, tip: `Context ${k(s.contextUsed)} / ${k(size)} (${Math.round(ratio * 100)}%)` }
})
const resetAt = (t: number) => (t ? new Date(t * 1000).toLocaleString(undefined, { weekday: 'short', hour: '2-digit', minute: '2-digit' }) : '')
// When the 5-hour session ends, as 14h50 ("5h" while it is unknown).
const endsAt = (t: number) => {
  if (!t) return '5h'
  const d = new Date(t * 1000)
  return `${d.getHours()}h${String(d.getMinutes()).padStart(2, '0')}`
}
// The plan's 5-hour and weekly limits and the context window, each a ring
// with its name before it: the 5-hour one named by when it ends.
const rings = computed(() => {
  const p = status.value?.plan
  const out: { label: string; ratio: number; warn: boolean; tip: string }[] = []
  if (p?.at && p.fiveHour >= 0) out.push({ label: endsAt(p.fiveHourReset), ratio: Math.min(p.fiveHour / 100, 1), warn: p.fiveHour >= 80, tip: `5h limit ${Math.round(p.fiveHour)}% · resets ${resetAt(p.fiveHourReset)}` })
  if (p?.at && p.week >= 0) out.push({ label: 'Week', ratio: Math.min(p.week / 100, 1), warn: p.week >= 80, tip: `Week limit ${Math.round(p.week)}% · resets ${resetAt(p.weekReset)}` })
  if (context.value) out.push({ label: 'Context', ...context.value })
  return out
})
const models: [string, string][] = [['fable', 'Fable'], ['opus', 'Opus'], ['sonnet', 'Sonnet'], ['haiku', 'Haiku']]
const efforts = ['low', 'medium', 'high', 'xhigh', 'max']
const agentOf = computed(() => jockey.settings?.agents.find((a) => a.id === props.step.agent))
const modelId = computed(() => props.step.modelSet || models.find(([id]) => status.value?.model.includes(id))?.[0] || props.step.model)
const modelText = computed(() => (props.step.modelSet ? models.find(([id]) => id === props.step.modelSet)?.[1] : '') || status.value?.modelName || model.value)
const effort = computed(() => props.step.effortSet || status.value?.effort || agentOf.value?.effort || 'xhigh')
const menu = ref<'' | 'model' | 'effort'>('')
const working = computed(() => running.value || approval.value)
const pickTip = computed(() => (working.value ? 'locked while Claude works' : props.step.live ? 'restarts Claude, keeps the chat' : 'used on resume'))
async function pick(model: string, eff: string) {
  menu.value = ''
  await jockey.attempt('Could not change the model', () => api.SetStepModel(props.step.id, model, eff))
}

async function checkAgain() {
  const procs = await api.SessionElsewhere(props.step.id).catch(() => [])
  if (procs.length) jockey.elsewhere[props.step.id] = procs
  else delete jockey.elsewhere[props.step.id]
}
</script>

<template>
  <div class="chat" :style="{ '--code-blue': jockey.dark ? '#b1b9f9' : '#5769f7', '--bash': jockey.dark ? '#fd5db1' : '#ff0087' }">
    <div class="pane">
      <div ref="thread" class="thread">
        <div class="inner">
          <p v-if="loaded && !found && !trust" class="empty">Claude Code has not saved this session's conversation yet. It shows here as soon as Claude starts.</p>

          <template v-for="e in list" :key="e.id">
            <!-- a command: a quiet line, what it does and under it how it went; the command and its output unfold -->
            <div v-if="e.kind === 'tool' && view(e.tool).body === 'command'" class="tool cmd">
              <button class="lrow two" :aria-expanded="isOpen(e.id) || confirming(e.tool)" @click="toggle(e.id)">
                <Icon name="run" :size="14" class="ico" />
                <span class="stack">
                  <span class="ttl" :title="String(e.tool.call.input?.command ?? '')">{{ commandLabel(e.tool.call.input) }}</span>
                  <span class="res" :class="{ bad: e.tool.result?.isError }">{{ confirming(e.tool) ? 'Waiting for your approval' : e.tool.result ? commandResult(e.tool.result.text, e.tool.result.isError) : busy(e.tool) ? 'Running…' : 'No result' }}</span>
                </span>
                <Icon v-if="busy(e.tool) && !confirming(e.tool)" name="spinner" :size="13" class="spin" />
                <Icon :name="isOpen(e.id) ? 'chevron-up' : 'chevron'" :size="12" class="disc" />
              </button>
              <div v-if="isOpen(e.id) || confirming(e.tool)" class="cbox">
                <div class="chold">
                  <pre class="command">{{ e.tool.call.input?.command }}</pre>
                  <button class="copyc" title="Copy Command" aria-label="Copy Command" @click="jockey.copy(String(e.tool.call.input?.command ?? ''), 'the command')"><Icon name="copy" :size="12" /></button>
                </div>
                <pre v-if="e.tool.result" class="output">{{ e.tool.result.text || 'No output.' }}</pre>
              </div>
              <div v-if="confirming(e.tool)" class="perm">
                <button class="pb" @click="allow"><Icon name="check" :size="13" class="ok" />Allow</button>
                <button class="pb" title="Decline (Esc)" @click="interrupt"><Icon name="close" :size="13" class="bad" />Reject</button>
                <button class="pb" @click="jockey.setSessionView('terminal')"><Icon name="terminal" :size="13" class="ico" />Answer in the terminal</button>
              </div>
            </div>

            <!-- an edit (or a tool waiting for approval): a card, its header the title -->
            <div v-else-if="e.kind === 'tool' && (view(e.tool).card || confirming(e.tool))" class="card" :class="{ failed: e.tool.result?.isError, quiet: !confirming(e.tool) }">
              <button class="chead" :aria-expanded="isOpen(e.id) || confirming(e.tool)" @click="toggle(e.id)">
                <Icon :name="view(e.tool).icon" :size="14" class="ico" />
                <span class="ttl" v-html="title(e.tool)"></span>
                <span v-if="e.tool.result?.diff" class="stat"><span class="add">+{{ diffStat(e.tool.result.diff).add }}</span><span class="del">−{{ diffStat(e.tool.result.diff).del }}</span></span>
                <Icon v-if="busy(e.tool) && !confirming(e.tool)" name="spinner" :size="13" class="spin" />
                <Icon v-else-if="e.tool.result?.isError" name="close" :size="13" class="bad" />
                <Icon v-if="!confirming(e.tool)" :name="isOpen(e.id) ? 'chevron-up' : 'chevron'" :size="12" class="disc" />
              </button>
              <template v-if="isOpen(e.id) || confirming(e.tool)">
                <div v-if="e.tool.result?.diff" class="diffw"><Diff :diff="e.tool.result.diff" :path="String(e.tool.call.input?.file_path ?? '')" /></div>
                <!-- Claude's question form, answered here: Jockey types the answers into Claude Code's form -->
                <div v-else-if="view(e.tool).body === 'question' && confirming(e.tool)" class="qs form">
                  <div v-for="(q, i) in questions(e.tool.call)" :key="i" class="qblock">
                    <div class="qq"><span v-if="q.header" class="qh">{{ q.header }}</span>{{ q.question }}<span v-if="q.multi" class="muted"> (pick any)</span></div>
                    <button v-for="(o, k) in q.options" :key="o.label" class="opt" :class="{ on: picked(i, k + 1) }" @click="pickOption(i, k + 1, q.multi)">
                      <span :class="q.multi ? 'box' : 'radio'"></span>
                      <span class="ol"><b>{{ o.label }}</b><span v-if="o.description" class="muted">{{ o.description }}</span></span>
                    </button>
                    <input v-if="!q.multi" v-model="typed[i]" class="other" :class="{ on: !!typed[i]?.trim() }" placeholder="Or type your own answer" data-own-escape @focus="picks[i] = []" />
                  </div>
                </div>
                <div v-else-if="view(e.tool).body === 'question'" class="qs">
                  <div v-for="(q, i) in questions(e.tool.call)" :key="i">
                    <div class="qq">{{ q.question }}</div>
                    <div v-for="o in q.options" :key="o.label" class="qo"><b>{{ o.label }}</b> <span class="muted">{{ o.description }}</span></div>
                  </div>
                </div>
                <pre v-else class="output">{{ e.tool.result?.text || rawInput(e.tool.call) }}</pre>
              </template>
              <div v-if="confirming(e.tool)" class="perm">
                <template v-if="e.tool.call.tool !== 'AskUserQuestion'">
                  <button class="pb" @click="allow"><Icon name="check" :size="13" class="ok" />Allow</button>
                  <button class="pb" title="Decline (Esc)" @click="interrupt"><Icon name="close" :size="13" class="bad" />Reject</button>
                  <button class="pb" @click="jockey.setSessionView('terminal')"><Icon name="terminal" :size="13" class="ico" />Answer in the terminal</button>
                </template>
                <template v-else>
                  <button class="pb" :disabled="!answered(e.tool) || answering" @click="submitAnswers(e.tool)"><Icon name="check" :size="13" class="ok" />{{ answering ? 'Sending…' : 'Submit answers' }}</button>
                  <button class="pb" title="Skip the form (Esc)" @click="interrupt"><Icon name="close" :size="13" class="bad" />Skip</button>
                </template>
              </div>
            </div>

            <!-- any other tool: a muted line; unfolded, its raw input and output on a left rule -->
            <div v-else-if="e.kind === 'tool'" class="tool">
              <button class="lrow" :aria-expanded="isOpen(e.id)" @click="toggle(e.id)">
                <Icon :name="view(e.tool).icon" :size="14" class="ico" />
                <span class="ttl" v-html="title(e.tool)"></span>
                <Icon v-if="busy(e.tool)" name="spinner" :size="13" class="spin" />
                <Icon v-else-if="e.tool.result?.isError" name="close" :size="13" class="bad" />
                <Icon :name="isOpen(e.id) ? 'chevron-up' : 'chevron'" :size="12" class="disc" />
              </button>
              <div v-if="isOpen(e.id)" class="rule">
                <ul v-if="view(e.tool).body === 'todos'" class="todos">
                  <li v-for="(td, i) in todos(e.tool.call)" :key="i" :class="td.status"><span class="box"></span>{{ td.text }}</li>
                </ul>
                <template v-else>
                  <div class="io">Raw Input:</div>
                  <pre class="raw">{{ rawInput(e.tool.call) }}</pre>
                  <div class="io">Output:</div>
                  <pre class="raw">{{ e.tool.result?.text || (busy(e.tool) ? 'Running…' : 'No output.') }}</pre>
                </template>
                <button class="fold" aria-label="Collapse" @click="toggle(e.id)"><Icon name="chevron-up" :size="13" /></button>
              </div>
            </div>

            <!-- the user's message, or the /command Jockey ran -->
            <div v-else-if="e.kind === 'item' && e.item.kind === 'bash'" class="umsg" :data-turn="e.turn">
              <div class="ubox bashbox"><span class="bang">!</span> <code>{{ e.item.text }}</code></div>
            </div>
            <pre v-else-if="e.kind === 'item' && e.item.kind === 'bashout'" class="bashout">{{ e.item.text || 'No output.' }}</pre>
            <div v-else-if="e.kind === 'item' && (e.item.kind === 'user' || e.item.kind === 'command')" class="umsg" :data-turn="e.turn">
              <div class="ubox"><code v-if="e.item.kind === 'command'" class="slash">{{ e.item.tool }}</code> {{ fold(e.id, e.item.text)?.text ?? e.item.text }}<button v-if="fold(e.id, e.item.text)" class="more" :aria-expanded="isOpen(e.id)" @click="toggle(e.id)"><Icon :name="isOpen(e.id) ? 'chevron-up' : 'chevron'" :size="12" />{{ isOpen(e.id) ? 'Show less' : `${fold(e.id, e.item.text)!.more} more lines` }}</button></div>
            </div>

            <!-- Claude's answer -->
            <div v-else-if="e.kind === 'item' && e.item.kind === 'text'" class="amsg md" v-html="html(e.item.text)"></div>
            <div v-else-if="e.kind === 'item' && e.item.kind === 'thinking'" class="amsg">
              <button class="lrow" :aria-expanded="isOpen(e.id)" @click="toggle(e.id)">
                <Icon name="think" :size="14" class="ico" /><span class="ttl">Thinking</span>
                <Icon :name="isOpen(e.id) ? 'chevron-up' : 'chevron'" :size="12" class="disc" />
              </button>
              <div v-if="isOpen(e.id)" class="thought md" v-html="html(e.item.text)"></div>
            </div>
            <div v-else-if="e.kind === 'item' && e.item.kind === 'error'" class="callout"><Icon name="error" :size="14" />{{ e.item.text }}</div>
            <div v-else-if="e.kind === 'item'" class="note">{{ e.item.text }}</div>

            <!-- a turn's end: copy its answer, back to its message; at the bottom, to the top -->
            <div v-else-if="e.kind === 'end' && !(e.bottom && (running || approval))" class="controls">
              <button class="ib" title="Copy response" aria-label="Copy response" @click="copyTurn(e.turn)"><Icon name="copy" :size="14" /></button>
              <button class="ib" title="Go to your message" aria-label="Go to your message" @click="toUser(e.turn)"><Icon name="user-up" :size="14" /></button>
              <button v-if="e.bottom" class="ib" title="Go to top" aria-label="Go to top" @click="toTop"><Icon name="arrow-up" :size="14" /></button>
            </div>
          </template>

          <!-- Claude waits on something that is not a tool call -->
          <div v-if="approval && !pending" class="card">
            <div class="chead static"><Icon name="approval" :size="14" class="ico" /><span class="ttl">Claude waits for your answer</span></div>
            <div class="perm">
              <button class="pb" @click="allow"><Icon name="check" :size="13" class="ok" />Allow</button>
              <button class="pb" @click="jockey.setSessionView('terminal')"><Icon name="terminal" :size="13" class="ico" />Answer in the terminal</button>
            </div>
          </div>
          <!-- Claude Code's folder trust dialog, shown before the session starts -->
          <div v-if="trust" class="card">
            <div class="chead static"><Icon name="folder" :size="14" class="ico" /><span class="ttl">Claude Code asks to trust this folder</span></div>
            <div class="trust">
              <code>{{ trust }}</code>
              <span>Claude has not worked in it before. Trusting it lets Claude read, edit and run files there.</span>
            </div>
            <div class="perm">
              <button class="pb" :disabled="trusting" @click="trustFolder"><Icon name="check" :size="13" class="ok" />Trust folder</button>
              <button class="pb" @click="jockey.setSessionView('terminal')"><Icon name="terminal" :size="13" class="ico" />Answer in the terminal</button>
            </div>
          </div>
          <div v-else-if="approval" class="generating"><span class="sand"></span><span class="lbl">Awaiting Confirmation</span></div>
          <div v-else-if="running" class="generating"><Working /></div>
          <!-- /btw: a side question and its answer, not part of the session -->
          <div v-for="b in btws" :key="b.id" class="card btw">
            <div class="chead static">
              <Icon name="chat" :size="14" class="ico" /><span class="ttl">/btw {{ b.q }}</span>
              <button class="ib" title="Dismiss" aria-label="Dismiss" @click="btws = btws.filter((x) => x !== b)"><Icon name="close" :size="13" /></button>
            </div>
            <div class="btw-body">
              <Working v-if="b.loading" />
              <div v-else-if="b.error" class="btw-err">{{ b.a }}</div>
              <div v-else class="amsg md" v-html="html(b.a)"></div>
            </div>
          </div>
        </div>
      </div>
      <MessagePicker :thread="thread" :marks="marks" @go="toUser" />
    </div>

    <!-- the message editor: across the bottom, its toolbar under the text -->
    <div class="foot">
      <div class="col">
        <div v-if="others" class="banner warn">
          <Icon name="warn" :size="14" />
          <span class="grow">Claude is still running this session elsewhere (pid {{ others[0].pid }}). Exit it there first, so only one Claude writes the conversation.</span>
          <button class="btn" @click="checkAgain">Check again</button>
          <button class="btn danger" @click="jockey.resumeAnyway(step)">Resume anyway</button>
        </div>
        <div v-else-if="paused" class="banner">
          <Icon name="stop" :size="14" />
          <span class="grow">Claude is stopped in Jockey: continue elsewhere with the resume command, or resume here.</span>
          <button class="btn" @click="jockey.copyResume(step)"><Icon name="copy" :size="13" />Copy claude --resume</button>
          <button class="btn primary" @click="jockey.resumeStep(step)"><Icon name="play" :size="13" />Resume here</button>
        </div>
        <div v-else-if="task?.archived && !canRestore" class="banner">
          <Icon name="archive" :size="14" />
          <span class="grow">This task is archived and can no longer run{{ task.worktreeRemoved ? ': its worktree was removed' : '' }}.</span>
          <button class="btn" @click="jockey.copyResume(step)"><Icon name="copy" :size="13" />Copy {{ resumeCommand(step).split(' ').slice(0, 2).join(' ') }}</button>
        </div>
        <div v-if="sugs.length" class="sugs" role="listbox">
          <button v-for="(s, i) in sugs" :key="s.label" role="option" :aria-selected="i === sugIx" :class="{ on: i === sugIx }" @mousedown.prevent="accept(s)" @mouseenter="sugIx = i">
            <span class="sl">{{ s.label }}</span><span class="sd">{{ s.detail }}</span>
          </button>
        </div>
        <textarea ref="box" v-model="draft" rows="4" :class="{ bash }" :placeholder="placeholder" :disabled="blocked" data-own-escape @keydown="key" @paste="paste" @input="sync" @click="sync" @keyup="sync"></textarea>
        <div class="tools">
          <button class="ib" title="Attach files" aria-label="Attach files" :disabled="blocked" @click="attach"><Icon name="plus" :size="14" /></button>
          <span v-if="bash" class="bashpill" title="Runs in the worktree's shell">! Bash mode</span>
          <span v-else class="sel who"><Icon name="agent" :size="13" />{{ agentName }}</span>
          <span class="grow"></span>
          <span v-for="r in rings" :key="r.label" class="ring" :class="{ warn: r.warn }" :title="r.tip">
            {{ r.label }}<svg width="16" height="16" viewBox="0 0 16 16"><circle cx="8" cy="8" r="6" fill="none" stroke-width="2" class="track" /><circle cx="8" cy="8" r="6" fill="none" stroke-width="2" class="fill" :stroke-dasharray="`${r.ratio * 37.7} 37.7`" transform="rotate(-90 8 8)" /></svg>
          </span>
          <span class="dd">
            <button class="sel pick" :disabled="working" :title="'Model · ' + pickTip" @click="menu = menu === 'model' ? '' : 'model'">{{ modelText }}<Icon name="chevron" :size="12" /></button>
            <div v-if="menu === 'model'" class="menu">
              <button v-for="[id, name] in models" :key="id" :class="{ on: id === modelId }" @click="pick(id, '')">{{ name }}<Icon v-if="id === modelId" name="check" :size="12" /></button>
            </div>
          </span>
          <span class="dd">
            <button class="sel pick" :disabled="working" :title="'Effort · ' + pickTip" @click="menu = menu === 'effort' ? '' : 'effort'">{{ effort }}<Icon name="chevron" :size="12" /></button>
            <div v-if="menu === 'effort'" class="menu">
              <button v-for="e in efforts" :key="e" :class="{ on: e === effort }" @click="pick('', e)">{{ e }}<Icon v-if="e === effort" name="check" :size="12" /></button>
            </div>
          </span>
          <button v-if="(running || approval) && !draft.trim()" class="send stop" title="Stop" aria-label="Stop" @click="interrupt"><Icon name="stop" :size="13" /></button>
          <button v-else class="send" :disabled="!draft.trim() || sending || blocked" :title="!draft.trim() ? 'Type a message' : running ? 'Queue' : step.live ? 'Send' : 'Resume with this message'" aria-label="Send" @click="send">
            <Icon :name="running && draft.trim() ? 'queue' : 'send-msg'" :size="14" />
          </button>
        </div>
        <div v-if="menu" class="scrim" @click="menu = ''"></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Zed's agent panel, in its units (one Zed unit is 4px): the thread on the
   panel background, user messages and cards on the editor background. */
.chat {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg);
  --card-border: color-mix(in srgb, var(--border) 80%, transparent);
  --card-head: var(--surface);
  --tool-size: var(--fs-13);
}
/* the thread, and the picker of the user's messages over its corner */
.pane {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.thread {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.inner,
.col {
  max-width: 860px;
  margin: 0 auto;
}
.inner {
  padding: 8px 0 16px;
  display: flex;
  flex-direction: column;
}
.empty {
  margin: 24px 20px;
  color: var(--muted);
  text-align: center;
  font-size: var(--fs-12-5);
}
.grow {
  flex: 1;
}
.muted {
  color: var(--muted);
}
.ico {
  flex: none;
  color: var(--muted);
}
.bad {
  flex: none;
  color: var(--del);
}
.ok {
  flex: none;
  color: var(--done);
}
.disc {
  flex: none;
  color: var(--muted);
  opacity: 0; /* Zed shows the disclosure on hover */
}
button:hover > .disc,
[aria-expanded='true'] > .disc {
  opacity: 1;
}
.spin {
  flex: none;
  color: var(--muted);
  animation: spin 2s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* user message: pt-2 pb-3 px-2; its box py-3 px-2, rounded, bordered, on the editor background */
.umsg {
  padding: 8px 20px 12px; /* as wide as the answers */
}
/* the one place Jockey leaves Zed: user messages stand out, tinted with the accent */
.ubox {
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--accent) 45%, var(--border));
  border-radius: 8px;
  background: color-mix(in srgb, var(--accent) 11%, var(--elev));
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.18);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-size: var(--fs-13);
  line-height: 1.5;
}
.ubox:hover {
  border-color: var(--accent);
}
.ubox .more {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 6px;
  padding: 0;
  border: 0;
  background: none;
  color: var(--muted);
  font: inherit;
  font-size: var(--fs-12);
  cursor: pointer;
}
.ubox .more:hover {
  color: var(--text);
}
.slash {
  font-family: var(--mono);
  font-size: var(--fs-12-5);
  color: var(--accent);
}

/* assistant message: px-5 py-1.5 */
.amsg {
  padding: 6px 20px;
}
.md {
  font-size: var(--fs-13-5);
  line-height: 1.6;
  overflow-wrap: anywhere;
}
.md :deep(p),
.md :deep(ul),
.md :deep(ol),
.md :deep(pre),
.md :deep(table),
.md :deep(blockquote) {
  margin: 0 0 12px;
}
.md > :deep(:last-child) {
  margin-bottom: 0;
}
.md :deep(ul),
.md :deep(ol) {
  padding-left: 22px;
}
.md :deep(h1),
.md :deep(h2),
.md :deep(h3),
.md :deep(h4) {
  margin: 16px 0 8px;
  font-size: var(--fs-14);
  font-weight: 600;
}
/* inline code in Claude Code's blue (its theme's "permission" color, which its terminal gives code spans) */
.md :deep(code) {
  font-family: var(--mono);
  font-size: var(--fs-12);
  padding: 1px 4px;
  border-radius: 4px;
  background: var(--surface);
  color: var(--code-blue);
}
.md :deep(pre code) {
  color: var(--text);
}
.md :deep(pre) {
  padding: 8px 10px;
  border-radius: 6px;
  border: 1px solid var(--card-border);
  background: var(--elev);
  overflow: auto;
}
.md :deep(pre code) {
  padding: 0;
  background: transparent;
}
.md :deep(a) {
  color: var(--accent);
}
.md :deep(th),
.md :deep(td) {
  padding: 4px 8px;
  border: 1px solid var(--border);
  text-align: left;
}
.md :deep(table) {
  border-collapse: collapse;
  font-size: var(--fs-12-5);
}
.md :deep(blockquote) {
  padding-left: 10px;
  border-left: 2px solid var(--border);
  color: var(--muted);
}

/* a tool line (and Thinking's): icon, muted title, disclosure on the right */
.tool {
  margin: 5px 20px;
}
.tool.cmd {
  margin: 8px 20px;
}
.lrow {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  min-width: 0;
  height: 20px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--muted);
  font-size: var(--tool-size);
  text-align: left;
}
.ttl {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  /* Zed fades a long title out instead of cutting it with an ellipsis */
  mask-image: linear-gradient(to right, #000 calc(100% - 48px), transparent);
}
.ttl :deep(code) {
  font-family: var(--mono);
  font-size: var(--fs-12);
  padding: 0 3px;
  border-radius: 3px;
  background: var(--surface);
  color: var(--code-blue);
}
.lrow:hover .ttl {
  color: var(--text);
}
/* unfolded: ml 0.4rem, px-3.5, a left rule; Raw Input and Output, then a collapse button */
.rule {
  margin: 6px 0 0 6px;
  padding: 0 14px 4px;
  border-left: 1px solid var(--card-border);
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.io {
  font-family: var(--mono);
  font-size: var(--fs-11);
  color: var(--muted);
}
.raw {
  margin: 0 0 4px;
  max-height: 240px;
  overflow: auto;
  font-family: var(--mono);
  font-size: var(--fs-12);
  line-height: 1.45;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.fold {
  height: 22px;
  margin-top: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: transparent;
  color: var(--muted);
}
.fold:hover {
  background: var(--hover);
}
.thought {
  margin: 4px 0 0 6px;
  padding-left: 14px;
  border-left: 1px solid var(--card-border);
  max-height: 256px;
  overflow: auto;
  color: var(--muted);
  font-size: var(--fs-13);
}

/* cards: my-1.5 mx-5, rounded, bordered; the header on its own background */
.card {
  margin: 6px 20px;
  border: 1px solid var(--card-border);
  border-radius: 6px;
  background: var(--elev);
  overflow: hidden;
}
.card.failed {
  border-color: var(--approval-dot);
}
.chead {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  min-width: 0;
  height: 26px;
  padding: 2px 8px 2px 6px;
  border: 0;
  background: var(--card-head);
  color: var(--text);
  font-size: var(--tool-size);
  text-align: left;
}
.chead.static {
  cursor: default;
}
.stat {
  flex: none;
  display: flex;
  gap: 6px;
  font-family: var(--mono);
  font-size: var(--fs-11-5);
}
.stat .add {
  color: var(--add);
}
.stat .del {
  color: var(--del);
}
.diffw {
  max-height: 360px;
  overflow: auto;
  border-top: 1px solid var(--card-border);
  background: var(--bg);
}
.output {
  margin: 0;
  max-height: 288px;
  overflow: auto;
  padding: 8px 10px;
  border-top: 1px solid var(--card-border);
  font-family: var(--mono);
  font-size: var(--fs-12);
  line-height: 1.45;
  color: var(--muted);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
/* a command: a quiet two-line row (what it does, how it went); unfolded, the
   command and its output in a box */
.lrow.two {
  height: auto;
  align-items: flex-start;
  padding: 1px 0;
}
.lrow.two > .ico {
  margin-top: 3px;
}
.stack {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.stack .ttl {
  flex: none;
  color: var(--text);
}
.res {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--fs-12);
  color: var(--faint);
}
.res.bad {
  color: var(--del);
}
.cbox {
  margin: 6px 0 2px 20px;
  border: 1px solid var(--card-border);
  border-radius: 6px;
  background: var(--elev);
  overflow: hidden;
}
.chold {
  position: relative;
}
.command {
  margin: 0;
  padding: 6px 30px 6px 10px;
  font-family: var(--mono);
  font-size: var(--fs-12);
  line-height: 17px;
  color: var(--text);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.copyc {
  position: absolute;
  right: 4px;
  top: 4px;
  width: 22px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--muted);
  opacity: 0;
}
.chold:hover .copyc {
  opacity: 1;
}
.copyc:hover {
  background: var(--hover);
}
/* an edit, quiet like the other tool lines: no box until its diff unfolds */
.card.quiet {
  margin: 5px 20px;
  border: 0;
  border-radius: 0;
  background: transparent;
  overflow: visible;
}
.card.quiet .chead {
  height: 20px;
  padding: 0;
  background: transparent;
  color: var(--muted);
}
.card.quiet .chead:hover .ttl {
  color: var(--text);
}
.card.quiet .diffw,
.card.quiet .output {
  margin: 6px 0 2px 20px;
  border: 1px solid var(--card-border);
  border-radius: 6px;
}
.qs {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 10px;
  border-top: 1px solid var(--card-border);
  font-size: var(--fs-12-5);
}
.qq {
  font-weight: 600;
  margin-bottom: 2px;
}
.qo {
  padding-left: 10px;
}
/* the question form */
.form {
  gap: 14px;
}
.qblock {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.qh {
  margin-right: 8px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--surface);
  color: var(--muted);
  font-size: var(--fs-11);
  font-weight: 500;
}
.opt {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 6px 8px;
  border: 1px solid var(--card-border);
  border-radius: 6px;
  background: transparent;
  color: var(--text);
  text-align: left;
  font-size: var(--fs-12-5);
}
.opt:hover {
  background: var(--hover);
}
.opt.on {
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent) 10%, transparent);
}
.opt .ol {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.opt .radio,
.opt .box {
  flex: none;
  width: 12px;
  height: 12px;
  margin-top: 2px;
  border: 1px solid var(--muted);
  border-radius: 50%;
}
.opt .box {
  border-radius: 3px;
}
.opt.on .radio,
.opt.on .box {
  border-color: var(--accent);
  background: var(--accent);
  box-shadow: inset 0 0 0 2px var(--elev);
}
.other {
  height: 28px;
  padding: 0 8px;
  border: 1px solid var(--card-border);
  border-radius: 6px;
  background: transparent;
  color: var(--text);
  font-size: var(--fs-12-5);
  outline: 0;
}
.other.on,
.other:focus {
  border-color: var(--accent);
}
.trust {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 10px;
  font-size: var(--fs-12);
  color: var(--muted);
}
.trust code {
  font-family: var(--mono);
  color: var(--text);
  overflow-wrap: anywhere;
}
/* permission buttons: p-1 under a rule, small labels with Zed's check and cross */
.btw-body {
  padding: 6px 12px 8px;
}
.btw-body .amsg {
  margin: 0;
}
.btw-err {
  color: var(--approval);
  font-size: var(--tool-size);
}
.btw .ib {
  margin-left: auto;
}
.perm {
  display: flex;
  flex-wrap: wrap;
  gap: 2px;
  padding: 4px;
  border-top: 1px solid var(--card-border);
}
.pb {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 24px;
  padding: 0 8px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--text);
  font-size: var(--fs-12);
}
.pb:hover:not(:disabled) {
  background: var(--hover);
}
.todos {
  list-style: none;
  margin: 0;
  padding: 2px 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: var(--fs-12-5);
}
.todos li {
  display: flex;
  align-items: center;
  gap: 8px;
}
.todos .box {
  flex: none;
  width: 11px;
  height: 11px;
  border: 1px solid var(--muted);
  border-radius: 3px;
}
.todos .completed {
  color: var(--muted);
  text-decoration: line-through;
}
.todos .completed .box {
  background: var(--done);
  border-color: var(--done);
}
.todos .in_progress .box {
  border-color: var(--accent);
  box-shadow: inset 0 0 0 2px var(--bg), inset 0 0 0 6px var(--accent);
}
.note {
  align-self: center;
  margin: 4px 20px;
  font-size: var(--fs-11-5);
  color: var(--faint);
}
.callout {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 6px 20px;
  padding: 8px 10px;
  border: 1px solid color-mix(in srgb, var(--del) 40%, var(--border));
  border-radius: 6px;
  background: var(--del-bg);
  color: var(--text);
  font-size: var(--fs-12-5);
}
.callout :deep(svg) {
  color: var(--del);
}

/* a turn's controls: right-aligned, faint until hovered (py-1.5 px-4, opacity 0.4) */
.controls {
  display: flex;
  justify-content: flex-end;
  padding: 6px 16px;
  opacity: 0.4;
}
.controls:hover {
  opacity: 1;
}
.ib {
  width: 24px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--muted);
}
.ib:hover:not(:disabled) {
  background: var(--hover);
  color: var(--text);
}
/* generating: py-2 px-22px, a dots spinner; awaiting confirmation, a breathing dot and its label */
.generating {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 22px;
  font-size: var(--fs-12);
  color: var(--muted);
}
.dots::before {
  display: inline-block;
  width: 8px;
  font-family: var(--mono);
  color: var(--muted);
  content: '⠋';
  animation: dots 0.8s steps(1) infinite;
}
.sand {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--muted);
  animation: sand 1.6s ease-in-out infinite; /* transform and opacity only: runs on the compositor, no steps */
}
@keyframes dots {
  0% { content: '⠋'; }
  10% { content: '⠙'; }
  20% { content: '⠹'; }
  30% { content: '⠸'; }
  40% { content: '⠼'; }
  50% { content: '⠴'; }
  60% { content: '⠦'; }
  70% { content: '⠧'; }
  80% { content: '⠇'; }
  90% { content: '⠏'; }
}
@keyframes sand {
  0%, 100% { transform: scale(0.6); opacity: 0.35; }
  50% { transform: scale(1); opacity: 1; }
}

/* the message editor: py-2, top border, on the chat's own background; its toolbar under the text */
.foot {
  flex: none;
  min-height: var(--dock-h); /* its natural height, with 4 lines of text and the toolbar: Next step's box matches it */
  box-sizing: border-box;
  padding: 8px 0;
  border-top: 1px solid var(--border);
  background: var(--bg);
}
.col {
  padding: 0 8px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.banner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 10px;
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface);
  font-size: var(--fs-12-5);
  color: var(--muted);
}
.banner.warn :deep(svg) {
  color: var(--waiting);
}
.banner .grow {
  min-width: 180px;
}
textarea {
  width: 100%;
  min-height: 81px; /* Zed's message editor: 4 lines at least */
  max-height: 300px;
  resize: none;
  padding: 4px 10px 0 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text);
  font-family: var(--ui-font);
  font-size: var(--fs-13-5);
  line-height: 1.5;
}
textarea::placeholder {
  color: var(--faint);
}
textarea:disabled {
  opacity: 0.55;
}
.tools {
  display: flex;
  align-items: center;
  gap: 4px;
}
.sel {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 22px;
  padding: 0 6px;
  font-size: var(--fs-12);
  color: var(--muted);
  white-space: nowrap;
}
/* plan usage and the context window, as Zed's context ring: 16px, 2px stroke,
   warning past 80% (85% for the context) */
.ring {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 0 4px;
  font-size: var(--fs-11-5);
  color: var(--muted);
  white-space: nowrap;
}
.ring.warn {
  color: var(--waiting);
}
.ring .track {
  stroke: var(--border);
}
.ring .fill {
  stroke: var(--muted);
}
.ring.warn .fill {
  stroke: var(--waiting);
}
.dd {
  position: relative;
}
.pick {
  border: 0;
  border-radius: 4px;
  background: transparent;
}
.pick:hover:not(:disabled) {
  background: var(--hover);
  color: var(--text);
}
.menu {
  position: absolute;
  right: 0;
  bottom: calc(100% + 6px);
  z-index: 20;
  min-width: 120px;
  padding: 4px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--bg);
  box-shadow: var(--shadow);
  display: flex;
  flex-direction: column;
}
.menu button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  height: 26px;
  padding: 0 8px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--text);
  font-size: var(--fs-12-5);
  text-align: left;
}
.menu button:hover {
  background: var(--hover);
}
.menu button.on {
  color: var(--accent);
}
.scrim {
  position: fixed;
  inset: 0;
  z-index: 10;
}
/* "/" and "@" suggestions, above the editor */
.col {
  position: relative;
}
.sugs {
  position: absolute;
  left: 8px;
  right: 8px;
  bottom: calc(100% + 4px);
  z-index: 20;
  max-height: 280px;
  overflow: auto;
  padding: 4px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--elev);
  box-shadow: var(--shadow);
  display: flex;
  flex-direction: column;
}
.sugs button {
  display: flex;
  align-items: baseline;
  gap: 10px;
  min-height: 26px;
  padding: 4px 8px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--text);
  text-align: left;
}
.sugs button.on {
  background: var(--hover);
}
.sl {
  flex: none;
  font-family: var(--mono);
  font-size: var(--fs-12);
  color: var(--code-blue);
}
.sd {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--fs-12);
  color: var(--muted);
}
/* bash mode: Claude Code's pink */
textarea.bash {
  font-family: var(--mono);
  font-size: var(--fs-12-5);
}
.bashpill {
  display: flex;
  align-items: center;
  height: 22px;
  padding: 0 8px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--bash) 16%, transparent);
  color: var(--bash);
  font-family: var(--mono);
  font-size: var(--fs-11-5);
}
.bashbox {
  border-color: color-mix(in srgb, var(--bash) 45%, var(--border));
  background: color-mix(in srgb, var(--bash) 9%, var(--elev));
  font-family: var(--mono);
  font-size: var(--fs-12-5);
}
.bang {
  color: var(--bash);
  font-weight: 600;
}
.bashout {
  margin: 0 20px 6px;
  max-height: 240px;
  overflow: auto;
  padding: 8px 10px;
  border: 1px solid var(--card-border);
  border-radius: 6px;
  background: var(--elev);
  font-family: var(--mono);
  font-size: var(--fs-12);
  line-height: 1.45;
  color: var(--muted);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
/* send: a filled button with an accent icon; muted while empty; stop: tinted red */
.send {
  flex: none;
  width: 26px;
  height: 22px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 4px;
  background: var(--surface);
  color: var(--accent);
}
.send:hover:not(:disabled) {
  background: var(--hover);
}
.send.stop {
  background: var(--del-bg);
  color: var(--del);
}
</style>
