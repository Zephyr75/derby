<script setup lang="ts">
// A step's Claude Code session as a chat (Settings > Appearance > Session
// view), after Zed's agent panel: the conversation read from Claude Code's
// transcript, tool calls as compact cards, and a message box that types into
// the same session as the terminal.
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { api, errorText } from '../lib/api'
import { herd, resumeCommand } from '../lib/state'
import { render } from '../lib/markdown'
import { entries, toolView, diffStat, questions, todos, type ToolCall } from '../lib/chat'
import type { ChatItem, Step } from '../lib/types'
import Icon from './ui/Icon.vue'
import Diff from './Diff.vue'

const props = defineProps<{ step: Step }>()
const task = computed(() => herd.task(props.step.taskId))

// ---- the conversation, followed while the session runs ----
const items = ref<ChatItem[]>([])
const title = ref('')
const found = ref(true)
const loaded = ref(false)
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
    if (c.title) title.value = c.title
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
  title.value = ''
  offset = 0
  loaded.value = false
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
const html = (text: string) => render(text, { task: props.step.taskId, dark: herd.dark })
const view = (t: ToolCall) => toolView(t.call, task.value?.worktree ?? '', task.value?.runDir ?? '')

// A card's open state: edits start open, the rest closed; a click toggles.
const isOpen = (id: string, t: ToolCall) => open.value[id] ?? (view(t).body === 'diff' && !!t.result?.diff)
const toggle = (id: string, t: ToolCall) => (open.value[id] = !isOpen(id, t))

// ---- the session's state ----
const running = computed(() => props.step.live && props.step.state === 'running')
const approval = computed(() => props.step.live && props.step.state === 'approval')
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
const others = computed(() => (props.step.live ? undefined : herd.elsewhere[props.step.id]))
const paused = computed(() => props.step.paused && !props.step.live)
const canRestore = computed(() => !!task.value && !task.value.worktreeRemoved && !task.value.projectGone)
const blocked = computed(() => !!others.value || paused.value || (!!task.value?.archived && !canRestore.value))

// ---- the message box ----
const draft = ref('')
const box = ref<HTMLTextAreaElement>()
const sending = ref(false)
function grow() {
  const el = box.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 220) + 'px'
}
watch(draft, () => void nextTick(grow))
async function send() {
  const text = draft.value.trim()
  if (!text || sending.value || blocked.value) return
  sending.value = true
  try {
    if (!(await herd.restoreFor(props.step))) return
    await api.SendToStep(props.step.id, text)
    draft.value = ''
    scrollDown()
  } catch (e) {
    const procs = await api.SessionElsewhere(props.step.id).catch(() => [])
    if (procs.length) herd.elsewhere[props.step.id] = procs
    else herd.toast('Could not send', errorText(e), true)
  } finally {
    sending.value = false
  }
}
function key(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
    e.preventDefault()
    void send()
  }
}
const interrupt = () => herd.attempt('Could not interrupt Claude', () => api.InterruptStep(props.step.id))
const allow = () => herd.attempt('Could not allow it', () => api.AllowOnce(props.step.id))
const placeholder = computed(() =>
  props.step.live ? (running.value ? 'Queue a message for Claude…' : 'Message Claude… (Enter to send, Shift+Enter for a new line)') : task.value?.archived ? 'Send to restore the task and resume this session…' : 'Send a message to resume this session…',
)
const agentName = computed(() => herd.settings?.agents.find((a) => a.id === props.step.agent)?.name ?? props.step.agent)
const model = computed(() => (props.step.model ? props.step.model[0].toUpperCase() + props.step.model.slice(1) : ''))
async function checkAgain() {
  const procs = await api.SessionElsewhere(props.step.id).catch(() => [])
  if (procs.length) herd.elsewhere[props.step.id] = procs
  else delete herd.elsewhere[props.step.id]
}
</script>

<template>
  <div class="chat">
    <div ref="thread" class="thread">
      <div class="inner">
        <div v-if="title" class="ttl">{{ title }}</div>
        <p v-if="loaded && !found" class="empty">Claude Code has not saved this session's conversation yet. It shows here as soon as Claude starts.</p>

        <template v-for="e in list" :key="e.id">
          <!-- a tool call and its result -->
          <div v-if="e.kind === 'tool'" class="tool" :class="[view(e.tool).body, { open: isOpen(e.id, e.tool), err: e.tool.result?.isError }]">
            <button class="thead" :aria-expanded="isOpen(e.id, e.tool)" @click="toggle(e.id, e.tool)">
              <Icon :name="view(e.tool).icon" :size="14" class="ticon" />
              <span class="verb">{{ view(e.tool).verb }}</span>
              <code v-if="view(e.tool).target" class="target">{{ view(e.tool).target }}</code>
              <span v-if="view(e.tool).dir" class="dir">{{ view(e.tool).dir }}</span>
              <span v-if="e.tool.result?.diff" class="stat">
                <span class="add">+{{ diffStat(e.tool.result.diff).add }}</span><span class="del">−{{ diffStat(e.tool.result.diff).del }}</span>
              </span>
              <span class="grow"></span>
              <Icon v-if="busy(e.tool)" name="spinner" :size="14" class="spin" />
              <Icon v-else-if="e.tool.result?.isError" name="error" :size="14" class="bad" />
              <Icon :name="isOpen(e.id, e.tool) ? 'chevron' : 'chevron-right'" :size="13" class="chev" />
            </button>
            <div v-if="isOpen(e.id, e.tool)" class="tbody">
              <div v-if="view(e.tool).body === 'diff' && e.tool.result?.diff" class="diffw"><Diff :diff="e.tool.result.diff" /></div>
              <template v-else-if="view(e.tool).body === 'command'">
                <pre class="cmd"><span class="prompt">$</span> {{ e.tool.call.input?.command }}</pre>
                <pre v-if="e.tool.result?.text" class="out">{{ e.tool.result.text }}</pre>
              </template>
              <ul v-else-if="view(e.tool).body === 'todos'" class="todos">
                <li v-for="(td, i) in todos(e.tool.call)" :key="i" :class="td.status"><span class="box"></span>{{ td.text }}</li>
              </ul>
              <div v-else-if="view(e.tool).body === 'question'" class="qs">
                <div v-for="(q, i) in questions(e.tool.call)" :key="i" class="q">
                  <div class="qq">{{ q.question }}</div>
                  <div v-for="o in q.options" :key="o.label" class="qo"><b>{{ o.label }}</b> <span class="muted">{{ o.description }}</span></div>
                </div>
                <pre v-if="e.tool.result?.text" class="out">{{ e.tool.result.text }}</pre>
              </div>
              <pre v-else class="out">{{ e.tool.result?.text || (busy(e.tool) ? 'Running…' : 'No output.') }}</pre>
            </div>
          </div>

          <!-- the user's message, or the /command herd ran -->
          <div v-else-if="e.item.kind === 'user'" class="user">{{ e.item.text }}</div>
          <div v-else-if="e.item.kind === 'command'" class="user cmdmsg">
            <code class="slash">{{ e.item.tool }}</code><span v-if="e.item.text" class="args">{{ e.item.text }}</span>
          </div>

          <!-- Claude's answer -->
          <div v-else-if="e.item.kind === 'text'" class="md" v-html="html(e.item.text)"></div>
          <details v-else-if="e.item.kind === 'thinking'" class="thinking">
            <summary><Icon name="thinking" :size="13" />Thinking</summary>
            <div class="md" v-html="html(e.item.text)"></div>
          </details>
          <div v-else-if="e.item.kind === 'error'" class="error"><Icon name="error" :size="14" />{{ e.item.text }}</div>
          <div v-else class="note">{{ e.item.text }}</div>
        </template>

        <!-- what Claude waits for -->
        <div v-if="approval && pending" class="ask">
          <div class="askh">
            <Icon name="approval" :size="15" />
            <span v-if="pending.call.tool === 'AskUserQuestion'">Claude asks you</span>
            <span v-else>Claude wants to <b>{{ view(pending).verb.toLowerCase() }}</b> <code v-if="view(pending).target && pending.call.tool !== 'Bash'">{{ view(pending).target }}</code></span>
          </div>
          <pre v-if="pending.call.tool === 'Bash'" class="cmd"><span class="prompt">$</span> {{ pending.call.input?.command }}</pre>
          <div v-if="pending.call.tool === 'AskUserQuestion'" class="qs">
            <div v-for="(q, i) in questions(pending.call)" :key="i" class="q">
              <div class="qq">{{ q.question }}</div>
              <div v-for="o in q.options" :key="o.label" class="qo"><b>{{ o.label }}</b> <span class="muted">{{ o.description }}</span></div>
            </div>
          </div>
          <div class="askb">
            <template v-if="pending.call.tool !== 'AskUserQuestion'">
              <button class="btn primary" @click="allow"><Icon name="check" :size="13" />Allow</button>
              <button class="btn" title="Decline, as Esc does in the terminal; then tell Claude what to do instead" @click="interrupt">Deny</button>
            </template>
            <button class="btn" @click="herd.setSessionView('terminal')"><Icon name="terminal" :size="13" />Answer in the terminal</button>
          </div>
        </div>
        <div v-else-if="approval" class="ask">
          <div class="askh"><Icon name="approval" :size="15" />Claude waits for your answer</div>
          <div class="askb">
            <button class="btn primary" @click="allow"><Icon name="check" :size="13" />Allow</button>
            <button class="btn" @click="herd.setSessionView('terminal')"><Icon name="terminal" :size="13" />Answer in the terminal</button>
          </div>
        </div>
        <div v-else-if="running" class="working"><Icon name="spinner" :size="14" class="spin" /><span>Working<span class="dots"></span></span></div>
      </div>
    </div>

    <div class="foot">
      <div v-if="others" class="banner warn">
        <Icon name="warn" :size="14" />
        <span class="grow">Claude is still running this session elsewhere (pid {{ others[0].pid }}). Exit it there first, so only one Claude writes the conversation.</span>
        <button class="btn" @click="checkAgain">Check again</button>
        <button class="btn danger" @click="herd.resumeAnyway(step)">Resume anyway</button>
      </div>
      <div v-else-if="paused" class="banner">
        <Icon name="stop" :size="14" />
        <span class="grow">Claude is stopped in Herd: continue elsewhere with the resume command, or resume here.</span>
        <button class="btn" @click="herd.copyResume(step)"><Icon name="copy" :size="13" />Copy claude --resume</button>
        <button class="btn primary" @click="herd.resumeStep(step)"><Icon name="play" :size="13" />Resume here</button>
      </div>
      <div v-else-if="task?.archived && !canRestore" class="banner">
        <Icon name="archive" :size="14" />
        <span class="grow">This task is archived and can no longer run{{ task.worktreeRemoved ? ': its worktree was removed' : '' }}.</span>
        <button class="btn" @click="herd.copyResume(step)"><Icon name="copy" :size="13" />Copy {{ resumeCommand(step).split(' ').slice(0, 2).join(' ') }}</button>
      </div>

      <div class="composer" :class="{ off: blocked }" @click="box?.focus()">
        <textarea ref="box" v-model="draft" rows="1" :placeholder="placeholder" :disabled="blocked" data-own-escape @keydown="key"></textarea>
        <div class="cbar">
          <span class="who"><Icon name="agent" :size="13" />{{ agentName }}<template v-if="model"> · {{ model }}</template></span>
          <span class="grow"></span>
          <button v-if="running || approval" class="stopb" title="Interrupt Claude, as Esc does in the terminal" @click.stop="interrupt"><Icon name="stop" :size="12" /></button>
          <button class="sendb" :disabled="!draft.trim() || sending || blocked" :title="step.live ? 'Send (Enter)' : 'Resume the session with this message (Enter)'" @click.stop="send"><Icon name="send" :size="15" /></button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--bg);
}
.thread {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.inner {
  max-width: 860px;
  margin: 0 auto;
  padding: 18px 20px 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.ttl {
  font-size: 13px;
  font-weight: 600;
  color: var(--muted);
  padding-bottom: 4px;
}
.empty {
  margin: 24px 0;
  color: var(--muted);
  text-align: center;
  font-size: 12.5px;
}
.grow {
  flex: 1;
}

/* the user's messages: a bordered box, as Zed's */
.user {
  padding: 9px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font-size: 13px;
  line-height: 1.5;
  margin-top: 6px;
}
.cmdmsg {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.slash {
  flex: none;
  font-family: var(--mono);
  font-size: 12px;
  color: var(--accent);
}
.args {
  color: var(--text);
}

/* Claude's answers: plain prose */
.md {
  font-size: 13.5px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}
.md :deep(p),
.md :deep(ul),
.md :deep(ol) {
  margin: 0 0 8px;
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
  margin: 12px 0 6px;
  font-size: 14px;
  font-weight: 600;
}
.md :deep(code) {
  font-family: var(--mono);
  font-size: 12px;
  padding: 1px 4px;
  border-radius: 4px;
  background: var(--surface);
}
.md :deep(pre) {
  margin: 0 0 8px;
  padding: 10px 12px;
  border-radius: 6px;
  border: 1px solid var(--border);
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
.md :deep(table) {
  border-collapse: collapse;
  margin: 0 0 8px;
  font-size: 12.5px;
}
.md :deep(th),
.md :deep(td) {
  padding: 4px 8px;
  border: 1px solid var(--border);
  text-align: left;
}
.md :deep(blockquote) {
  margin: 0 0 8px;
  padding-left: 10px;
  border-left: 3px solid var(--border);
  color: var(--muted);
}

/* tool calls: a one-line card that opens on click */
.tool {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--bg);
  overflow: hidden;
}
.tool.err {
  border-color: color-mix(in srgb, var(--del) 45%, var(--border));
}
.thead {
  display: flex;
  align-items: center;
  gap: 7px;
  width: 100%;
  min-height: 30px;
  padding: 4px 10px;
  border: 0;
  background: transparent;
  color: var(--text);
  font-size: 12.5px;
  text-align: left;
  min-width: 0;
}
.thead:hover {
  background: var(--hover);
}
.tool.open .thead {
  border-bottom: 1px solid var(--border);
}
.ticon {
  flex: none;
  color: var(--muted);
}
.verb {
  flex: none;
  color: var(--muted);
}
.target {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--mono);
  font-size: 11.5px;
  padding: 1px 5px;
  border-radius: 4px;
  background: var(--surface);
}
.dir {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11.5px;
  color: var(--faint);
}
.stat {
  flex: none;
  display: flex;
  gap: 6px;
  font-family: var(--mono);
  font-size: 11.5px;
}
.stat .add {
  color: var(--add);
}
.stat .del {
  color: var(--del);
}
.chev {
  flex: none;
  color: var(--faint);
}
.bad {
  flex: none;
  color: var(--del);
}
.spin {
  flex: none;
  color: var(--accent);
  animation: spin 0.9s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
.tbody {
  max-height: 360px;
  overflow: auto;
  background: var(--elev);
}
.diffw {
  overflow: auto;
}
.cmd,
.out {
  margin: 0;
  padding: 8px 12px;
  font-family: var(--mono);
  font-size: 11.5px;
  line-height: 1.5;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.cmd {
  color: var(--text);
}
.prompt {
  color: var(--accent);
  user-select: none;
}
.out {
  color: var(--muted);
}
.cmd + .out {
  padding-top: 0;
}
.todos {
  list-style: none;
  margin: 0;
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12.5px;
}
.todos li {
  display: flex;
  align-items: center;
  gap: 8px;
}
.todos .box {
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
  box-shadow: inset 0 0 0 2px var(--elev), inset 0 0 0 6px var(--accent);
}
.qs {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 12px;
  font-size: 12.5px;
}
.qq {
  font-weight: 600;
  margin-bottom: 2px;
}
.qo {
  padding-left: 10px;
}
.muted {
  color: var(--muted);
}

.thinking {
  font-size: 12.5px;
  color: var(--muted);
}
.thinking summary {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: default;
  list-style: none;
}
.thinking summary::-webkit-details-marker {
  display: none;
}
.thinking .md {
  margin-top: 6px;
  padding-left: 10px;
  border-left: 2px solid var(--border);
  font-size: 12.5px;
}
.note {
  align-self: center;
  font-size: 11.5px;
  color: var(--faint);
}
.error {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--del-bg);
  color: var(--del);
  font-size: 12.5px;
}

/* what Claude waits for */
.ask {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px 12px;
  border: 1px solid color-mix(in srgb, var(--waiting) 55%, var(--border));
  border-radius: 8px;
  background: var(--waiting-bg);
}
.askh {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text);
}
.askh :deep(svg) {
  color: var(--waiting);
  flex: none;
}
.askh code {
  font-family: var(--mono);
  font-size: 12px;
}
.ask .cmd {
  border-radius: 6px;
  background: var(--elev);
}
.askb {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.working {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
  color: var(--muted);
}
.dots::after {
  content: '';
  animation: dots 1.2s steps(4, end) infinite;
}
@keyframes dots {
  0% {
    content: '';
  }
  25% {
    content: '.';
  }
  50% {
    content: '..';
  }
  75% {
    content: '...';
  }
}

/* the message box, as Zed's: a bordered editor with its toolbar inside */
.foot {
  flex: none;
  max-width: 860px;
  width: 100%;
  margin: 0 auto;
  padding: 0 20px 14px;
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
  border-radius: 8px;
  background: var(--surface);
  font-size: 12.5px;
  color: var(--muted);
}
.banner.warn :deep(svg) {
  color: var(--waiting);
}
.banner .grow {
  min-width: 180px;
}
.composer {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px 10px 8px 12px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--elev);
  cursor: text;
}
.composer:focus-within {
  border-color: color-mix(in srgb, var(--accent) 60%, var(--border));
}
.composer.off {
  opacity: 0.55;
}
textarea {
  width: 100%;
  min-height: 20px;
  max-height: 220px;
  resize: none;
  padding: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text);
  font-family: var(--ui-font);
  font-size: 13.5px;
  line-height: 1.5;
}
textarea::placeholder {
  color: var(--faint);
}
.cbar {
  display: flex;
  align-items: center;
  gap: 8px;
}
.who {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--muted);
  white-space: nowrap;
}
.sendb,
.stopb {
  flex: none;
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 6px;
}
.sendb {
  background: var(--button);
  color: var(--button-text);
}
.sendb:disabled {
  background: var(--surface);
  color: var(--faint);
}
.stopb {
  background: var(--del-bg);
  color: var(--del);
}
.stopb:hover {
  background: color-mix(in srgb, var(--del) 25%, transparent);
}
</style>
