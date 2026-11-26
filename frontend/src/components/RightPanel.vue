<script setup lang="ts">
import { computed, ref } from 'vue'
import { api } from '../lib/api'
import { jockey, taskName } from '../lib/state'
import Icon from './ui/Icon.vue'
import TaskRef from './ui/TaskRef.vue'
import { sisterTitle } from '../lib/sisters'
import Preview from './Preview.vue'
import Splitter from './ui/Splitter.vue'
import { useCompact } from '../lib/compact'
import { unsuccessful, type Step } from '../lib/types'

const t = computed(() => jockey.selTask)
// The buttons are shortcuts to the task's own steps, so there is no need to
// open GitHub: Issue (or Task) to the Task step, PR to the latest Copilot step,
// which describes the PR and links to it. A PR without a Copilot step opens on GitHub.
const openTask = () => t.value && jockey.openTask(t.value.id)
const openPR = () => {
  const task = t.value
  if (!task) return
  const cop = jockey.latestPR(task)
  if (cop) jockey.openStep(task.id, cop.id)
  else jockey.attempt('Could not open the PR', () => api.OpenURL(task.prUrl || `https://github.com/${task.repo}/pull/${task.pr}`))
}
// GitHub's PR icons, as in the left panel: merged, closed, draft, open.
const prIcon = computed(() => (t.value?.prState === 'merged' ? 'rebase' : t.value?.prState && t.value.prState !== 'open' ? 'pr-closed' : t.value?.draft ? 'pr-draft' : 'pr'))
// A task with an issue has an Issue button, one without a Task button.
const hasIssue = computed(() => !!(t.value?.issue && (t.value.issueRepo || t.value.repo)))
const titleTip = computed(() => {
  const task = t.value
  if (!task) return ''
  const full = task.name && task.name !== task.title ? task.title : ''
  const sisters = sisterTitle(task.id)
  return [full, sisters].filter(Boolean).join('\n\n')
})
// Rebase always shows, before Archive; it's off unless the branch is behind
// and a step can rebase it now.
const canRebase = computed(() => !!t.value?.behind && !t.value.archived && jockey.canFix(t.value))
const rebaseTitle = computed(() => {
  const task = t.value
  if (!task || task.archived) return 'Archived'
  if (!task.behind) return 'Up to date'
  const behind = `${task.behind} behind origin/${task.baseBranch}`
  return canRebase.value ? `Rebase: ${behind}` : `${behind} · wait for the step`
})

// The buttons under the title keep only their icons when their names don't fit.
const bar = ref<HTMLDivElement>()
const compact = useCompact(bar)

function meta(s: Step): { text: string; cls: string } {
  if (s.paused && s.state !== 'done' && s.state !== 'failed') return { text: 'paused', cls: '' }
  if (s.state === 'running') return { text: s.external ? 'waiting for Copilot' : 'running', cls: 'c-running' }
  if (s.state === 'approval') return { text: 'needs approval', cls: 'c-waiting' }
  if (s.state === 'idle') return { text: 'waiting for you', cls: 'c-waiting' }
  if (s.state === 'failed') return { text: 'ended early', cls: 'c-approval' }
  if (unsuccessful(s)) return { text: 'failed', cls: 'c-approval' }
  return { text: '', cls: '' }
}
const dotCls = (s: Step) => (unsuccessful(s) ? 'd-failed' : s.state === 'done' ? 'd-done' : s.paused ? 'd-paused' : s.state === 'running' ? 'd-running' : s.state === 'failed' ? 'd-failed' : 'd-waiting')
// What used to sit under the step's name, now on hover.
const stepInfo = (s: Step) => (s.external ? 'Run by Jockey · PR #' + t.value?.pr : s.sha ? 'Commit ' + s.sha : 'Session ' + s.session.slice(0, 8)) + (s.auto ? ' · auto' : '')

// Rerun (set by jockey): the latest step when it commits, and the cycle's
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
  const move = (m: PointerEvent) => jockey.setTimelineHeight(Math.min(max, Math.max(36, startH + m.clientY - startY)))
  const up = () => {
    handle.removeEventListener('pointermove', move)
    handle.removeEventListener('pointerup', up)
  }
  handle.addEventListener('pointermove', move)
  handle.addEventListener('pointerup', up)
}

async function openRunDir() {
  const task = t.value
  if (task) await jockey.attempt('Could not open the run folder', () => api.OpenRunDir(task.id))
}

const rebase = () => t.value && jockey.fixWith(t.value, jockey.prompt('rebase', { pr: String(t.value.pr), base: t.value.baseBranch, behind: String(t.value.behind) }))

async function archive() {
  const task = t.value
  if (task) await jockey.attempt('Could not archive', () => api.ArchiveTask(task.id))
}
</script>

<template>
  <aside :style="{ width: jockey.rightWidth + 'px' }">
    <Splitter side="right" />
    <template v-if="t">
      <div class="head">
        <div class="title-row" :title="titleTip">
          <TaskRef :issue="t.issue" />
          <span class="title">{{ taskName(t) }}</span>
        </div>
        <div ref="bar" class="bar" :class="{ compact }">
          <button class="hb" :title="hasIssue ? 'Show the issue, its comments and your instructions' : 'Show the task'" :aria-label="hasIssue ? 'Issue' : 'Task'" @click="openTask"><Icon :name="hasIssue ? 'issue' : 'description'" /><span class="label">{{ hasIssue ? '#' + t.issue : 'Task' }}</span></button>
          <button class="hb" :class="{ off: !t.pr }" :aria-disabled="!t.pr" :title="t.pr ? 'Show the PR: its description, review and checks' : 'No PR yet'" aria-label="PR" @click="t.pr && openPR()"><Icon :name="prIcon" /><span class="label">{{ t.pr ? '#' + t.pr : 'PR' }}</span></button>
          <button v-if="!t.archived && !t.worktreeRemoved" class="hb" title="Open in Zed" aria-label="Zed" @click="jockey.openWorktree()"><Icon name="code" /><span class="label">Zed</span></button>
          <button class="hb" title="Open reports folder" aria-label="Reports" @click="openRunDir"><Icon name="folder" /><span class="label">Reports</span></button>
          <button class="hb" :class="{ warn: t.behind && !t.archived, off: !canRebase }" :aria-disabled="!canRebase" :title="rebaseTitle" aria-label="Rebase" @click="canRebase && rebase()"><Icon name="rebase" /><span class="label">Rebase</span></button>
          <button v-if="!t.archived" class="hb" title="Archive task" aria-label="Archive" @click="archive"><Icon name="archive" /><span class="label">Archive</span></button>
        </div>
      </div>

      <div ref="timeline" class="timeline" :class="{ sized: jockey.timelineHeight > 0 }" :style="jockey.timelineHeight > 0 ? { height: jockey.timelineHeight + 'px' } : {}">
        <div class="row" :class="{ on: jockey.onTaskStep }">
          <span class="rail">
            <span class="sdot d-done"></span>
            <span v-if="t.steps.length || t.next.ghost" class="line"></span>
          </span>
          <button class="step" :title="t.source" @click="jockey.openTask(t.id)"><span class="name-row"><span class="name">Task</span></span></button>
        </div>
        <div v-for="(s, i) in t.steps" :key="s.id" class="row" :class="{ on: jockey.selStep?.id === s.id }">
          <span class="rail">
            <span :class="['sdot', dotCls(s)]"></span>
            <span v-if="i < t.steps.length - 1 || t.next.ghost" class="line"></span>
          </span>
          <button class="step" :title="stepInfo(s)" @click="jockey.openStep(t.id, s.id)">
            <span class="name-row">
              <span class="name">{{ s.name }}</span>
              <span v-if="meta(s).text" :class="['meta', meta(s).cls]">{{ meta(s).text }}</span>
              <span v-if="s.skipped || (s.external && s.verdict === 'skipped')" class="meta skipped" :title="s.skipped ? 'Ended by hand' : 'Stopped waiting for Copilot'">skipped</span>
            </span>
          </button>
          <span class="acts">
            <!-- the first click only asks, in words -->
            <button v-if="s.revert" class="act" :class="jockey.confirmRevert === s.id ? 'danger' : 'icon'" :aria-label="jockey.confirmRevert === s.id ? 'Discard its changes?' : `Revert ${s.name}`" :title="`Undo ${s.name} and later steps`" @click="jockey.revertStep(s)">
              <Icon name="revert" :size="13" /><template v-if="jockey.confirmRevert === s.id">Discard its changes?</template>
            </button>
            <template v-if="s.skip">
              <button v-if="!(s.state === 'done' && !unsuccessful(s))" class="act icon" :aria-label="`Skip ${s.name} as succeeded`" title="Mark done" @click="jockey.skipStep(s, true)"><Icon name="check" :size="13" /></button>
              <button v-if="!unsuccessful(s)" class="act icon" :aria-label="`Skip ${s.name} as failed`" title="Mark failed" @click="jockey.skipStep(s, false)"><Icon name="close" :size="13" /></button>
            </template>
            <button v-if="s.rerun" class="act icon" :aria-label="`Rerun ${s.name}`" :title="`Rerun ${s.name}`" @click="jockey.rerunStep(s)"><Icon name="rerun" :size="13" /></button>
          </span>
        </div>
        <div v-if="t.next.ghost" class="row ghost" :title="`Starts after ${t.next.ghost.after}`">
          <span class="rail"><span class="sdot pending"></span></span>
          <span class="step static">
            <span class="name">{{ t.next.ghost.name }}</span>
          </span>
        </div>
      </div>

      <div class="hsplit" role="separator" aria-orientation="horizontal" title="Drag to resize; double-click to reset" @pointerdown.prevent="dragSplit" @dblclick="jockey.setTimelineHeight(0)"></div>
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
  /* 76px = the centre's tab bar (32) and toolbar (44): the lines meet */
  .head {
    height: 76px;
    padding: 0 16px;
    justify-content: center;
    border-bottom: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: none;
  }
  /* The issue ref and the task's name */
  .title-row {
    display: flex;
    align-items: center;
    gap: 8px;
    align-self: flex-start;
    max-width: calc(100% + 10px);
    min-width: 0;
    margin: 0 -5px;
    padding: 2px 5px;
    color: var(--text);
  }
  .title {
    min-width: 0;
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
    justify-content: space-between;
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
  .hb:hover:not(.off) {
    background: var(--hover);
    color: var(--text);
  }
  .hb.warn {
    color: var(--waiting);
  }
  .label {
    margin-left: 6px; /* LABEL_GAP in lib/compact.ts */
    overflow: hidden;
  }
  .compact .label {
    max-width: 0;
    margin-left: 0;
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
    border: 1px solid var(--muted);
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
    overflow: hidden; /* under the step's buttons, e.g. Revert's question */
    white-space: nowrap;
  }
  .name {
    flex: none;
    max-width: 100%; /* the meta gives way first */
    overflow: hidden;
    text-overflow: ellipsis;
    font-weight: 500;
  }
  .meta {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
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
    background: color-mix(in srgb, var(--approval-dot) 14%, var(--panel));
    color: var(--approval);
  }
  .act.danger:hover {
    background: color-mix(in srgb, var(--approval-dot) 22%, var(--panel));
  }
</style>
