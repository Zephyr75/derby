<script setup lang="ts">
import { computed, onBeforeUnmount, onUpdated, ref, watch } from 'vue'
import { api } from '../lib/api'
import { derby } from '../lib/state'
import Icon from './ui/Icon.vue'
import TaskRef from './ui/TaskRef.vue'
import Preview from './Preview.vue'
import Splitter from './ui/Splitter.vue'
import { unsuccessful, type Step } from '../lib/types'

const t = computed(() => derby.selTask)
// Agents set to "Show a Diff button" (Implement and Test by default) get one.
const diffs = (s: Step) => !!derby.settings?.agents.find((a) => a.id === s.agent)?.showDiff
const openIssue = () => t.value && derby.attempt('Could not open the issue', () => api.OpenURL(`https://github.com/${t.value!.issueRepo || t.value!.repo}/issues/${t.value!.issue}`))
const openPR = () => t.value && derby.attempt('Could not open the PR', () => api.OpenURL(t.value!.prUrl || `https://github.com/${t.value!.repo}/pull/${t.value!.pr}`))
// GitHub's PR icons, as in the left panel: merged, closed, draft, open.
const prIcon = computed(() => (t.value?.prState === 'merged' ? 'rebase' : t.value?.prState && t.value.prState !== 'open' ? 'pr-closed' : t.value?.draft ? 'pr-draft' : 'pr'))
const behindText = computed(() => `${t.value?.behind} commits behind origin/${t.value?.baseBranch} (Derby fetches it every 5 minutes)`)

// The buttons under the title drop their names, keeping their icons, when
// they don't fit on one line. Hidden names keep their width (max-width: 0
// with overflow hidden), so the row knows when they fit again.
const bar = ref<HTMLDivElement>()
const compact = ref(false)
const LABEL_GAP = 6
function fit() {
  const el = bar.value
  if (!el) return
  const kids = [...el.children] as HTMLElement[]
  let need = kids.reduce((w, k) => w + k.offsetWidth, 0) + 2 * Math.max(0, kids.length - 1)
  if (compact.value) for (const l of el.querySelectorAll<HTMLElement>('.label')) need += l.scrollWidth + LABEL_GAP
  // a few pixels of slack each way so rounding can't make it flicker
  const next = compact.value ? need > el.clientWidth - 6 : need > el.clientWidth - 2
  if (next !== compact.value) compact.value = next
}
const sizes = new ResizeObserver(fit)
watch(bar, (el, old) => {
  if (old) sizes.unobserve(old)
  if (el) sizes.observe(el)
}, { flush: 'post' })
onUpdated(fit)
document.fonts?.ready.then(fit)
onBeforeUnmount(() => sizes.disconnect())

function meta(s: Step): { text: string; cls: string } {
  if (s.paused && s.state !== 'done' && s.state !== 'failed') return { text: 'paused', cls: '' }
  if (s.state === 'running') return { text: s.external ? 'waiting for Copilot' : 'running', cls: 'c-running' }
  if (s.state === 'approval') return { text: 'needs approval', cls: 'c-waiting' }
  if (s.state === 'idle') return { text: 'waiting for you', cls: 'c-waiting' }
  if (s.state === 'failed') return { text: 'ended early', cls: 'c-approval' }
  if (unsuccessful(s)) return { text: 'failed', cls: 'c-approval' }
  const task = t.value
  if (task && s.state === 'done' && task.pr && derby.settings?.agents.find((a) => a.id === s.agent)?.openPR) return { text: 'PR #' + task.pr, cls: '' }
  return { text: '', cls: '' }
}
const dotCls = (s: Step) => (unsuccessful(s) ? 'd-failed' : s.state === 'done' ? 'd-done' : s.paused ? 'd-paused' : s.state === 'running' ? 'd-running' : s.state === 'failed' ? 'd-failed' : 'd-waiting')
// What used to sit under the step's name, now on hover.
const stepInfo = (s: Step) => (s.external ? 'Run by Derby · PR #' + t.value?.pr : s.sha ? 'Commit ' + s.sha : 'Session ' + s.session.slice(0, 8)) + (s.auto ? ' · auto' : '')

// Rerun (set by derby): the latest step when it commits, and the cycle's
// Review once you changed the worktree by hand (Zed opens it).

// Drag the line above the preview to give it more or less room.
const timeline = ref<HTMLDivElement>()
function dragSplit(e: PointerEvent) {
  const el = timeline.value
  if (!el) return
  const startY = e.clientY
  const startH = el.offsetHeight
  const max = (el.parentElement?.clientHeight ?? 800) - 160
  const handle = e.currentTarget as HTMLElement
  try {
    handle.setPointerCapture(e.pointerId)
  } catch {
    /* synthetic event */
  }
  const move = (m: PointerEvent) => derby.setTimelineHeight(Math.min(max, Math.max(36, startH + m.clientY - startY)))
  const up = () => {
    handle.removeEventListener('pointermove', move)
    handle.removeEventListener('pointerup', up)
  }
  handle.addEventListener('pointermove', move)
  handle.addEventListener('pointerup', up)
}

async function openRunDir() {
  const task = t.value
  if (task) await derby.attempt('Could not open the run folder', () => api.OpenRunDir(task.id))
}

const rebase = () => t.value && derby.fixWith(t.value, derby.prompt('rebase', { pr: String(t.value.pr), base: t.value.baseBranch, behind: String(t.value.behind) }))

async function archive() {
  const task = t.value
  if (task) await derby.attempt('Could not archive', () => api.ArchiveTask(task.id))
}
</script>

<template>
  <aside :style="{ width: derby.rightWidth + 'px' }">
    <Splitter side="right" />
    <template v-if="t">
      <div class="head">
        <div class="title-row">
          <TaskRef :issue="t.issue" :task="t.id" />
          <span class="title" :title="t.title">{{ t.title }}</span>
        </div>
        <div ref="bar" class="bar" :class="{ compact }">
          <button v-if="t.issue && (t.issueRepo || t.repo)" class="hb" :title="`Open issue #${t.issue} of ${t.issueRepo || t.repo} on GitHub`" aria-label="Issue" @click="openIssue"><Icon name="issue" /><span class="label">Issue</span></button>
          <button v-if="t.pr" class="hb" :title="`Open PR #${t.pr} on GitHub`" :aria-label="`PR #${t.pr}`" @click="openPR"><Icon :name="prIcon" /><span class="label">PR #{{ t.pr }}</span></button>
          <button v-if="!t.archived && !t.worktreeRemoved" class="hb" title="Open the task's worktree in Zed to edit it by hand" aria-label="Zed" @click="derby.openWorktree()"><Icon name="code" /><span class="label">Zed</span></button>
          <button class="hb" :title="`Open the folder with this task's reports, the markdown files its steps wrote (${t.runDir})`" aria-label="Reports" @click="openRunDir"><Icon name="folder" /><span class="label">Reports</span></button>
          <template v-if="t.behind && !t.archived">
            <button v-if="derby.canFix(t)" class="hb warn" :title="`${behindText}. Start a ${derby.fixers(t)[0]?.name} step that rebases the branch on origin/${t.baseBranch}`" aria-label="Rebase" @click="rebase"><Icon name="rebase" /><span class="label">Rebase</span></button>
            <span v-else class="hb warn static" :title="behindText"><Icon name="rebase" /><span class="label">{{ t.behind }} behind</span></span>
          </template>
          <button v-if="!t.archived" class="hb" title="Archive task" aria-label="Archive" @click="archive"><Icon name="archive" /><span class="label">Archive</span></button>
        </div>
        <p v-if="t.description" class="desc">{{ t.description }}</p>
      </div>

      <div ref="timeline" class="timeline" :class="{ sized: derby.timelineHeight > 0 }" :style="derby.timelineHeight > 0 ? { height: derby.timelineHeight + 'px' } : {}">
        <p v-if="!t.steps.length" class="none">No steps yet.</p>
        <div v-for="(s, i) in t.steps" :key="s.id" class="row" :class="{ on: derby.selStep?.id === s.id }">
          <span class="rail">
            <span :class="['sdot', dotCls(s)]"></span>
            <span v-if="i < t.steps.length - 1 || t.next.ghost" class="line"></span>
          </span>
          <button class="step" :title="stepInfo(s)" @click="derby.openStep(t.id, s.id)">
            <span class="name-row">
              <span class="name">{{ s.name }}</span>
              <span v-if="meta(s).text" :class="['meta', meta(s).cls]">{{ meta(s).text }}</span>
              <span v-if="s.skipped || (s.external && s.verdict === 'skipped')" class="meta skipped" :title="s.skipped ? 'Ended with the check or cross button' : 'Derby stopped waiting for Copilot: you moved on'">skipped</span>
            </span>
          </button>
          <span class="acts">
            <button v-if="s.revert" class="act" :class="{ danger: derby.confirmRevert === s.id }" :title="`Put the worktree back to how it was when ${s.name} started, and remove it and the steps after it`" @click="derby.revertStep(s)">
              <Icon name="revert" :size="12" />{{ derby.confirmRevert === s.id ? 'Discard its changes?' : 'Revert' }}
            </button>
            <template v-if="s.skip">
              <button v-if="!(s.state === 'done' && !unsuccessful(s))" class="act icon" :aria-label="`Skip ${s.name} as succeeded`" :title="`Skip ${s.name} as succeeded: end it now and move on (Derby does not commit or push for it)`" @click="derby.skipStep(s, true)"><Icon name="check" :size="13" /></button>
              <button v-if="!unsuccessful(s)" class="act icon" :aria-label="`Skip ${s.name} as failed`" :title="`Skip ${s.name} as failed: end it now and get the options for a failure`" @click="derby.skipStep(s, false)"><Icon name="close" :size="13" /></button>
            </template>
            <button v-if="s.rerun" class="act" :title="`Run ${s.name} again in its own session, on the current worktree`" @click="derby.rerunStep(s)"><Icon name="rerun" :size="12" />Rerun</button>
            <button v-if="s.external" class="act" :title="`Every change of the branch against ${t.baseBranch}, as PR #${t.pr} shows it`" @click="derby.openDiffPopup(s.id, '')"><Icon name="diff" :size="13" />Full diff</button>
            <button v-else-if="s.diff && diffs(s)" class="act" aria-label="Open this step's changes in Zed" title="Open this step's changes in Zed: the commit it belongs to, or the uncommitted work" @click="derby.openDiff(s)"><Icon name="diff" :size="13" />Diff</button>
          </span>
        </div>
        <div v-if="t.next.ghost" class="row ghost" :title="`Starts automatically if ${t.next.ghost.after} succeeds`">
          <span class="rail"><span class="sdot pending"></span></span>
          <span class="step static">
            <span class="name">{{ t.next.ghost.name }}</span>
          </span>
        </div>
      </div>

      <div class="hsplit" role="separator" aria-orientation="horizontal" title="Drag to resize; double-click to reset" @pointerdown.prevent="dragSplit" @dblclick="derby.setTimelineHeight(0)"></div>
      <Preview />
    </template>
  </aside>
</template>

<style scoped>
  aside {
    position: relative;
    flex: 0 1 auto; /* narrow windows: shrink before the terminal is cut */
    min-width: 240px;
    background: var(--panel); /* Zed's panel shade; the artifact below is on the editor's */
    border-left: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    min-height: 0;
  }
  .head {
    padding: 14px 16px;
    border-bottom: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: none;
  }
  .title-row {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }
  .title {
    flex: 1;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  /* Issue, PR, Zed, Reports, Archive: Zed's ghost buttons, names dropped
     when the panel is too narrow for them */
  .bar {
    display: flex;
    align-items: center;
    gap: 2px;
    min-width: 0;
    margin: 0 -7px; /* the icons line up with the title's reference */
    overflow: hidden;
  }
  .hb {
    flex: none;
    display: flex;
    align-items: center;
    height: 26px;
    padding: 0 7px;
    border: 0;
    border-radius: 6px;
    background: transparent;
    color: var(--muted);
    font-size: var(--fs-12-5);
    white-space: nowrap;
  }
  .hb:hover:not(.static) {
    background: var(--hover);
    color: var(--text);
  }
  .hb.warn {
    color: var(--waiting);
  }
  .hb.static {
    cursor: default;
  }
  .label {
    margin-left: 6px; /* LABEL_GAP */
    overflow: hidden;
  }
  .compact .label {
    max-width: 0;
    margin-left: 0;
  }
  .desc {
    margin: 0;
    padding: 6px 8px;
    border-radius: 6px;
    background: var(--surface);
    border: 1px solid var(--border);
    font-size: var(--fs-12-5);
    line-height: 1.5;
    max-height: 120px;
    overflow: auto;
    white-space: pre-wrap;
  }
  .timeline {
    display: flex;
    flex-direction: column;
    padding: 8px 8px 6px;
    gap: 2px;
    flex: none;
    max-height: 40%;
    overflow: auto;
  }
  .timeline.sized {
    max-height: none;
  }
  .hsplit {
    flex: none;
    height: 7px;
    margin: 0 0 -4px;
    position: relative;
    z-index: 2;
    cursor: row-resize;
  }
  .hsplit:hover,
  .hsplit:active {
    background: linear-gradient(to bottom, transparent 2px, var(--accent) 2px, var(--accent) 4px, transparent 4px);
  }
  .none {
    margin: 8px;
    color: var(--muted);
    font-size: var(--fs-12-5);
  }
  /* one line per step: name, state and actions on a 36px row */
  .row {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    padding: 0 10px;
    border-radius: 6px;
  }
  .row.on {
    background: var(--sel);
  }
  .rail {
    display: flex;
    flex-direction: column;
    align-items: center;
    align-self: stretch;
    width: 12px;
  }
  .sdot {
    flex: none;
    width: 9px;
    height: 9px;
    border-radius: 50%;
    margin-top: 13.5px; /* centred on the name's line */
  }
  .d-done { background: var(--done-dot); }
  .d-paused { background: var(--faint); }
  .d-running { background: var(--running-dot); }
  .d-waiting { background: var(--waiting-dot); } /* Claude waits for an answer */
  .d-failed { background: var(--approval-dot); }
  .pending {
    border: 1px dashed var(--muted);
  }
  .line {
    flex: 1;
    width: 1px;
    background: var(--border);
    margin: 4px 0 -11.5px; /* down to 4px above the next step's dot */
  }
  .step {
    flex: 1;
    min-width: 0;
    text-align: left;
    background: transparent;
    border: 0;
    color: var(--text);
    padding: 8px 0;
    display: flex;
    flex-direction: column;
    font-size: var(--fs-13);
    line-height: 20px;
  }
  .step.static .name {
    color: var(--muted);
    font-style: italic;
  }
  .name-row {
    display: flex;
    gap: 8px;
    align-items: baseline;
  }
  .name {
    font-weight: 500;
  }
  .meta {
    font-size: var(--fs-12);
    color: var(--muted);
  }
  .meta.skipped {
    font-size: var(--fs-11);
  }
  .c-running { color: var(--running); }
  .c-approval { color: var(--approval); }
  .c-waiting { color: var(--waiting); }
  .c-done { color: var(--done); }
  .acts {
    flex: none;
    display: flex;
    gap: 4px;
    margin-top: 7px; /* centred on the name's line, like the dot */
  }
  .act {
    display: flex;
    align-items: center;
    gap: 5px;
    height: 22px;
    padding: 0 8px;
    border-radius: 4px;
    border: 1px solid transparent; /* Zed: ghost buttons, a fill on hover */
    background: transparent;
    color: var(--text);
    font-size: var(--fs-12);
  }
  .act.icon {
    padding: 0 4px;
  }
  .act:hover {
    background: var(--hover);
  }
  .act.danger {
    border-color: var(--approval-dot);
    color: var(--approval);
  }
</style>
