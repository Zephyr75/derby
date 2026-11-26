<script setup lang="ts">
// The changes of a step, large: the changed files on the left (as in the
// Changes tab), the selected file's read-only diff on the right.
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { api, errorText } from '../../lib/api'
import { jockey } from '../../lib/state'
import type { Changes, FileChange, FileDiff, Note, Step } from '../../lib/types'
import Icon from '../ui/Icon.vue'
import DiffView from '../Diff.vue'
import NotesBar from '../NotesBar.vue'
import ChangedFiles, { split, statusCls, statusTitle } from '../ChangedFiles.vue'
import Splitter from '../ui/Splitter.vue'
import Dropdown from '../ui/Dropdown.vue'
import { SIDEBAR, useSidebar } from '../../lib/sidebar'

// Unified or split, as Zed's diff toggle; remembered per viewer.
const splitView = ref(read('jockey.diff.split') === '1')
watch(splitView, (v) => write('jockey.diff.split', v ? '1' : '0'))
function read(k: string) {
  try {
    return localStorage.getItem(k)
  } catch {
    return null
  }
}
function write(k: string, v: string) {
  try {
    localStorage.setItem(k, v)
  } catch {
    /* private window: not remembered */
  }
}

const stepId = computed(() => jockey.diffOf?.step ?? '')
const task = computed(() => jockey.tasks.find((t) => t.steps.some((s) => s.id === stepId.value)))
const step = computed(() => task.value?.steps.find((s) => s.id === stepId.value))
const pickStep = (id: string) => jockey.diffOf && (jockey.diffOf = { step: id, path: '' })
// Only steps of agents set to "Makes changes" (Implement and Discuss by
// default) make changes. Full, last, is the whole branch: a Copilot step's changes, the
// same for every one of them.
const changers = computed(() => {
  const steps = task.value?.steps ?? []
  const full = step.value?.external ? step.value : steps.filter((s) => s.external).pop()
  return [...steps.filter((s) => !s.external && jockey.settings?.agents.find((a) => a.id === s.agent)?.showDiff), ...(full ? [full] : [])]
})
const label = (s: Step) => (s.external ? 'Full' : s.name)
const stepItems = computed(() => changers.value.map((s) => ({ value: s.id, label: label(s) })))
const chooseStep = (id: string) => id !== stepId.value && pickStep(id)
const changes = ref<Changes | null>(null)
const files = computed(() => changes.value?.files ?? [])
// Files of the same path can exist in two repositories: a file is told apart by its task.
const keyOf = (f: FileChange) => (f.task ?? '') + '\0' + f.path
// Zed starts a file folded (lock files, generated code…) when its path has one of the fold words.
const foldWords = computed(() => (jockey.settings?.config.foldWords ?? []).map((w) => w.toLowerCase()).filter(Boolean))
const foldedByDefault = (f: FileChange) => foldWords.value.some((w) => f.path.toLowerCase().includes(w))
const folded = ref<Record<string, boolean>>({})
const diffs = ref<Record<string, { diff?: FileDiff; error?: string }>>({})
const selKey = ref('')
const error = ref('')

let cseq = 0
async function loadChanges() {
  const mine = ++cseq
  const c = await api.StepChanges(stepId.value).catch(() => null)
  if (mine !== cseq) return
  // Opened from the artifact pop-up on a step without changes (Plan, an
  // earlier Review…): show the newest step that has some instead.
  if (jockey.diffOf?.auto) {
    jockey.diffOf.auto = false
    if (!c?.files.length) {
      for (const s of changers.value.filter((x) => !x.external).reverse()) { // a step's own changes, not the (full) branch
        if (s.id === stepId.value) continue
        const o = await api.StepChanges(s.id).catch(() => null)
        if (mine !== cseq) return
        if (o?.files.length) return pickStep(s.id)
      }
    }
  }
  changes.value = c
  const next: Record<string, boolean> = {}
  for (const f of c?.files ?? []) next[keyOf(f)] = folded.value[keyOf(f)] ?? foldedByDefault(f)
  folded.value = next
  if (!files.value.some((f) => keyOf(f) === selKey.value)) selKey.value = files.value[0] ? keyOf(files.value[0]) : ''
  void loadVisible()
}
async function loadDiff(f: FileChange) {
  const k = keyOf(f)
  const sid = stepId.value
  try {
    const d = f.task && f.task !== task.value?.id ? await api.SisterFileDiff(sid, f.task, f.path) : await api.FileDiff(sid, f.path)
    if (sid === stepId.value) diffs.value[k] = { diff: d }
  } catch (e) {
    if (sid === stepId.value) diffs.value[k] = { error: errorText(e) }
  }
}
// A file's diff is fetched once it is open.
async function loadVisible(all = false) {
  await Promise.all(files.value.filter((f) => !folded.value[keyOf(f)] && (all || !diffs.value[keyOf(f)])).map(loadDiff))
}
watch(stepId, () => {
  diffs.value = {}
  folded.value = {}
  selKey.value = ''
  void loadChanges()
}, { immediate: true })
// Uncommitted changes keep moving while the agent works.
let timer: ReturnType<typeof setTimeout> | undefined
watch(
  () => [step.value?.sha, step.value?.state, jockey.selTask?.steps.length] as const,
  () => {
    clearTimeout(timer)
    timer = setTimeout(() => void loadChanges().then(() => loadVisible(true)), 800)
  },
)

const cards = ref<Record<string, HTMLElement>>({})
const scroller = ref<HTMLDivElement>()
function toggle(f: FileChange) {
  const k = keyOf(f)
  folded.value[k] = !folded.value[k]
  if (!folded.value[k]) void loadVisible()
}
const allFolded = computed(() => files.value.length > 0 && files.value.every((f) => folded.value[keyOf(f)]))
function foldAll() {
  const to = !allFolded.value
  for (const f of files.value) folded.value[keyOf(f)] = to
  void loadVisible()
}
// A click in the list goes to the file, opening it if it is folded.
async function pick(path: string, taskId?: string) {
  const f = files.value.find((x) => x.path === path && (taskId === undefined || (x.task ?? '') === taskId))
  if (!f) return
  const k = keyOf(f)
  selKey.value = k
  pickedAt = Date.now()
  if (folded.value[k]) folded.value[k] = false
  void loadVisible()
  await nextTick()
  const el = cards.value[k]
  if (el && scroller.value) scroller.value.scrollTo({ top: el.offsetTop, behavior: 'smooth' })
}
// The file picked from outside (the Changes tab).
watch(() => jockey.diffOf?.path, (p) => { if (p && files.value.length) void pick(p) })
watch(changes, () => {
  const p = jockey.diffOf?.path
  if (p) void pick(p)
}, { once: true })
// The file in view is the one marked in the list.
let pickedAt = 0
function spy() {
  if (Date.now() - pickedAt < 800) return
  const top = (scroller.value?.scrollTop ?? 0) + 4
  let cur = ''
  for (const f of files.value) {
    const el = cards.value[keyOf(f)]
    if (el && el.offsetTop <= top) cur = keyOf(f)
  }
  if (cur) selKey.value = cur
}

// Comments on the diff, kept with the task until sent to a step. A sister's
// file is named with its repository.
const notePath = (f: FileChange) => (f.task && f.task !== task.value?.id && f.repo ? f.repo + '/' + f.path : f.path)
const notesOf = (f: FileChange) => (task.value?.notes ?? []).filter((n) => n.path === notePath(f))
function addNote(f: FileChange, n: Omit<Note, 'id' | 'path'>) {
  const t = task.value
  if (t) void jockey.setNotes(t, [...t.notes, { ...n, id: Date.now().toString(36) + Math.random().toString(36).slice(2, 6), path: notePath(f) }])
}
function updateNote(id: string, text: string) {
  const t = task.value
  if (t) void jockey.setNotes(t, t.notes.map((n) => (n.id === id ? { ...n, text } : n)))
}
function removeNote(id: string) {
  const t = task.value
  if (t) void jockey.setNotes(t, t.notes.filter((n) => n.id !== id))
}
const sb = useSidebar('diff', 300)
const close = () => jockey.closeDiffPopup()

// Zed shows this step's own diff: its commit (an Implement's is its
// Review's), in the task's worktree; uncommitted work opens the worktree.
async function openInZed() {
  const s = step.value
  if (!s) return
  const lines = await jockey.attempt('Could not open Zed', () => api.OpenInZed(s.taskId, s.id))
  if (lines) jockey.toast('Opening the diff in Zed', lines.join('\n'))
}

const box = ref<HTMLDivElement>()
onMounted(() => box.value?.focus())

// Up and down move through the files.
function key(e: KeyboardEvent) {
  if ((e.target as HTMLElement).tagName === 'TEXTAREA') return
  const i = files.value.findIndex((f) => keyOf(f) === selKey.value)
  const to = e.key === 'ArrowDown' ? files.value[i + 1] : e.key === 'ArrowUp' ? files.value[i - 1] : undefined
  if (!to) return
  void pick(to.path, to.task ?? '')
  e.preventDefault()
}
</script>

<template>
  <div ref="box" class="box popup" role="dialog" aria-label="Changes" tabindex="-1" @keydown="key">
    <div class="head pop-head">
      <button v-if="!jockey.artifactOf" class="other" title="Show the report, on the left" @click="jockey.showArtifactToo(stepId)"><Icon name="file" :size="13" />Report</button>
      <button class="toggle" :aria-label="sb.hidden.value ? 'Show the side bar' : 'Hide the side bar'" :title="sb.hidden.value ? 'Show sidebar' : 'Hide sidebar'" @click="sb.hidden.value = !sb.hidden.value"><Icon :name="sb.hidden.value ? 'panel-left-off' : 'panel-left'" :size="16" /></button>
      <Dropdown class="picker" :items="stepItems" :value="stepId" :placeholder="step ? label(step) : 'Step'" strong title="Other steps" @pick="chooseStep" />
      <span class="title">changes</span>
      <span class="grow"></span>
      <span class="tools">
        <button v-if="files.length > 1" class="ibtn" :title="allFolded ? 'Unfold all files' : 'Fold all files'" :aria-label="allFolded ? 'Unfold all files' : 'Fold all files'" @click="foldAll"><Icon :name="allFolded ? 'unfold' : 'fold'" :size="15" /></button>
        <span class="views" role="radiogroup" aria-label="Diff view">
          <button role="radio" :aria-checked="!splitView" :class="{ on: !splitView }" title="Unified" aria-label="Unified" @click="splitView = false">
            <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><rect x="3" y="2.6" width="10" height="4.4" rx="1" /><rect x="3" y="9" width="10" height="4.4" rx="1" fill="currentColor" fill-opacity="0.5" /></svg>
          </button>
          <button role="radio" :aria-checked="splitView" :class="{ on: splitView }" title="Split" aria-label="Split" @click="splitView = true">
            <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><rect x="2.6" y="3" width="4.4" height="10" rx="1" fill="currentColor" fill-opacity="0.5" /><rect x="9" y="3" width="4.4" height="10" rx="1" /></svg>
          </button>
        </span>
        <button class="ibtn" title="Open the diff in Zed" aria-label="Open the diff in Zed" @click="openInZed"><Icon name="diff" :size="15" /></button>
      </span>
      <button class="pop-x" aria-label="Close" title="Close (Esc)" @click="close"><Icon name="close" :size="14" /></button>
    </div>
    <div class="body">
      <div v-if="!sb.hidden.value" class="side-wrap" :style="{ width: sb.width.value + 'px' }">
        <div class="side">
          <p v-if="!changes" class="empty">Loading…</p>
          <p v-else-if="!changes.files.length" class="empty">No changed files.</p>
          <ChangedFiles v-else :changes="changes" :selected="selKey" @pick="pick" />
        </div>
        <Splitter side="left" :width="sb.width.value" :min="SIDEBAR.min" :max="SIDEBAR.max" :def="sb.def" label="Resize the side bar" @resize="(w) => (sb.width.value = w)" />
      </div>
      <div class="main">
        <NotesBar v-if="task?.notes.length" class="notes" :task="task" />
        <div ref="scroller" class="scroll" @scroll.passive="spy">
          <p v-if="!changes" class="empty">Loading…</p>
          <p v-else-if="!files.length" class="empty">No changed files.</p>
          <section v-for="(f, i) in files" :key="keyOf(f)" :ref="(el) => { if (el) cards[keyOf(f)] = el as HTMLElement }" class="card">
            <p v-if="f.repo && (i === 0 || files[i - 1].repo !== f.repo) && files.some((x) => x.repo !== f.repo)" class="repo">{{ f.repo }}</p>
            <button class="ch" :class="{ on: keyOf(f) === selKey }" :aria-expanded="!folded[keyOf(f)]" @click="toggle(f)">
              <Icon :name="folded[keyOf(f)] ? 'chevron-right' : 'chevron'" :size="12" />
              <span class="path mono" :title="f.path"><span :class="['name', statusCls(f.status)]">{{ split(f.path).name }}</span> <span class="dir">{{ split(f.path).dir }}</span></span>
              <span v-if="f.binary" class="muted small">binary</span>
              <template v-else>
                <span v-if="f.added" class="c-add small">+{{ f.added }}</span>
                <span v-if="f.deleted" class="c-del small">−{{ f.deleted }}</span>
              </template>
            </button>
            <template v-if="!folded[keyOf(f)]">
              <p v-if="diffs[keyOf(f)]?.error" class="empty err">{{ diffs[keyOf(f)].error }}</p>
              <p v-else-if="!diffs[keyOf(f)]?.diff" class="empty">Loading…</p>
              <p v-else-if="diffs[keyOf(f)].diff!.binary" class="empty">Binary file: no text diff.</p>
              <p v-else-if="!diffs[keyOf(f)].diff!.diff" class="empty">No text changes (mode or whitespace only).</p>
              <template v-else>
                <DiffView :diff="diffs[keyOf(f)].diff!.diff" :path="f.path" :sources="diffs[keyOf(f)].diff!.sources ? diffs[keyOf(f)].diff : undefined" :notes="notesOf(f)" :split="splitView" commentable @add="(n) => addNote(f, n)" @update="updateNote" @remove="removeNote" />
                <p v-if="diffs[keyOf(f)].diff!.truncated" class="empty">The diff is too long to show in full, open the file in Zed for the rest.</p>
              </template>
            </template>
          </section>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
  .box {
    flex: none;
    min-width: 0;
    display: flex;
    flex-direction: column;
    outline: none;
  }
  .other {
    flex: none;
    display: flex;
    align-items: center;
    gap: 5px;
    height: 28px;
    margin-right: 4px;
    padding: 0 10px;
    border: 1px solid transparent; /* Zed: ghost buttons, a fill on hover */
    border-radius: 6px;
    background: transparent;
    color: var(--text);
    font-size: var(--fs-12);
  }
  .notes {
    margin: 8px 12px 0;
  }
  .other:hover {
    background: var(--hover);
  }
  .title {
    font-weight: 600;
  }
  .picker {
    margin-right: 6px;
  }
  .grow {
    flex: 1;
  }
  .body {
    flex: 1;
    min-height: 0;
    display: flex;
  }
  .side-wrap {
    position: relative;
    flex: none;
    max-width: 50%;
    border-right: 1px solid var(--border);
  }
  .toggle {
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 4px;
    border: 0;
    border-radius: 6px;
    background: transparent;
    color: var(--muted);
  }
  .toggle:hover {
    background: var(--hover);
    color: var(--text);
  }
  .side {
    height: 100%;
    padding: 12px 14px;
    overflow: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .tools {
    flex: none;
    display: flex;
    align-items: center;
    gap: 2px;
    margin-right: 6px;
  }
  .st {
    flex: none;
    font-family: var(--mono);
    font-size: var(--fs-11);
    font-weight: 600;
  }
  .path {
    flex: 1;
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .dir {
    color: var(--muted);
  }
  .views {
    flex: none;
    display: flex;
    gap: 2px;
  }
  .ibtn,
  .views button {
    flex: none;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--muted);
  }
  .ibtn:hover,
  .views button:hover,
  .views button.on {
    background: var(--hover);
    color: var(--text);
  }
  .scroll {
    position: relative;
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
  .ch {
    position: sticky;
    top: 6px;
    z-index: 2;
    width: calc(100% - 16px);
    margin: 6px 8px;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 30px;
    padding: 0 10px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--surface);
    color: var(--text);
    text-align: left;
    font-size: var(--fs-12-5);
  }
  .name.c-mod {
    color: var(--accent);
  }
  .ch:hover {
    background: var(--hover);
  }
  .small {
    flex: none;
    font-family: var(--mono);
    font-size: var(--fs-11-5);
  }
  .repo {
    margin: 0;
    padding: 10px 14px 4px;
    color: var(--muted);
    font-size: var(--fs-11-5);
    font-weight: 600;
  }
  .empty {
    margin: 12px 14px;
    color: var(--muted);
    font-size: var(--fs-12-5);
  }
  .side .empty {
    margin: 0;
  }
  .err {
    color: var(--approval);
  }
  .c-add { color: var(--done); }
  .c-del { color: var(--approval); }
  .c-mod { color: var(--waiting); }
</style>
