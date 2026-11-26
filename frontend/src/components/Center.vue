<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, reactive, ref, watch } from 'vue'
import { api } from '../lib/api'
import { jockey, resumeCommand, taskName, taskStepId } from '../lib/state'
import { usePR } from '../lib/pr'
import { unsuccessful } from '../lib/types'
import Icon from './ui/Icon.vue'
import ChatView from './ChatView.vue'
import PRTracker from './PRTracker.vue'
import TaskPage from './TaskPage.vue'
import Toasts from './Toasts.vue'

const Terminal = defineAsyncComponent(() => import('./Terminal.vue')) // xterm loads with the first terminal

// One tab per task: its dot says whether one of its steps waits for approval
// or runs, whichever step the tab shows.
const tabs = computed(() =>
  jockey.tabs.map((tb, i) => {
    const t = jockey.task(tb.task)
    const steps = t?.steps ?? []
    const kind = steps.some((s) => s.state === 'approval') ? 'approval' : steps.some((s) => s.state === 'running' || s.state === 'idle') ? 'live' : 'off'
    const step = tb.step === taskStepId(tb.task) ? 'Task' : (steps.find((x) => x.id === tb.step)?.name ?? '')
    return { i, id: tb.task, kind, label: `${t?.issue ? '#' + t.issue : 'LOCAL'} · ${step}`, tip: t ? taskName(t) : '', project: t?.projectName ?? '', on: i === jockey.active && !!jockey.activeTab }
  }),
)
// The bar scrolls to the open tab when the tabs overflow it.
const bar = ref<HTMLElement>()
const reveal = () => bar.value?.querySelector('.tab.on')?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
watch(() => [jockey.active, jockey.tabs.length], reveal, { flush: 'post' })
onMounted(reveal)
const step = computed(() => jockey.selStep)
const t = computed(() => jockey.selTask)
const agent = computed(() => (step.value ? jockey.settings?.agents.find((a) => a.id === step.value!.agent) : undefined))
const model = (m: string) => (m ? m[0].toUpperCase() + m.slice(1) : '')
const chip = computed(() => {
  const s = step.value
  if (!s) return null
  if (s.paused && !s.live) return { cls: 's-idle', text: 'Paused' }
  if (s.state === 'approval') return { cls: 's-approval', text: 'Live · waiting for approval' }
  if (s.state === 'running' || s.state === 'idle') return { cls: 's-running', text: s.state === 'idle' ? 'Live · waiting for you' : 'Live' }
  if (s.state === 'failed') return { cls: 's-waiting', text: s.live ? 'Resumed · step ended early' : 'Ended early' }
  if (!s.live && jockey.sessionView === 'chat') return { cls: 's-idle', text: 'Ended' } // the chat reads it without resuming it
  return { cls: 's-idle', text: s.live ? 'Resumed' : 'Resumed on click' }
})

// The task's Task step: a link to the issue when there is one.
const issueUrl = computed(() => (t.value?.issue ? `https://github.com/${t.value.issueRepo || t.value.repo}/issues/${t.value.issue}` : ''))
const openIssue = () => jockey.attempt('Could not open the issue', () => api.OpenURL(issueUrl.value))

// The Copilot step, which describes the PR: its toolbar has the PR's buttons.
const pr = reactive(usePR(reactive({ get task() { return t.value! }, get step() { return step.value ?? undefined } })))
const prChip = computed(() => {
  const s = step.value
  if (!s) return null
  if (s.state === 'running') return { cls: 's-running', text: 'Waiting for Copilot' }
  if (unsuccessful(s)) return { cls: 's-approval', text: 'Needs fixes' }
  return s.verdict === 'skipped' ? { cls: 's-idle', text: 'Stopped waiting' } : { cls: 's-done', text: 'Passed' }
})
</script>

<template>
  <section>
    <div ref="bar" class="tabs" role="tablist">
      <div v-for="tb in tabs" :key="tb.id" class="tab" :class="{ on: tb.on }">
        <button class="open" role="tab" :aria-selected="tb.on" :title="tb.tip" @click="jockey.focusTab(tb.i)">
          <span :class="['dot', `k-${tb.kind}`]"></span>
          <span>{{ tb.label }}</span>
          <span class="proj">{{ tb.project }}</span>
        </button>
        <button class="x" aria-label="Close tab" @click="jockey.closeTab(tb.i)"><Icon name="close" :size="14" /></button>
      </div>
      <span class="grow"></span>
    </div>

    <template v-if="jockey.onTaskStep && t">
      <div class="toolbar">
        <span class="chip badge s-idle"><span class="dot"></span>Task</span>
        <span class="muted small">{{ t.source }}</span>
        <span class="tbs">
          <button v-if="issueUrl" class="tb" title="Open the issue on GitHub" @click="openIssue"><Icon name="external" :size="13" />Open issue</button>
        </span>
      </div>
      <TaskPage />
    </template>
    <template v-else-if="step && t && step.external">
      <div class="toolbar">
        <span v-if="prChip" :class="['chip', 'badge', prChip.cls]"><span class="dot"></span>{{ prChip.text }}</span>
        <span class="muted small ttl" :title="`${t.branch} → ${t.baseBranch}`">PR #{{ t.pr }}{{ pr.ready ? ' · ' + (pr.ready === 'ready' ? 'Ready' : 'Draft') : '' }} · {{ t.branch }} → {{ t.baseBranch }}</span>
        <span class="tbs">
          <button v-if="pr.ready === 'draft'" class="tb" title="Mark ready for review" @click="pr.markReady"><Icon name="check" :size="13" />Mark ready</button>
          <button v-if="pr.live" class="tb" :disabled="pr.refreshing" title="Refresh now" @click="pr.refresh"><Icon name="rerun" :size="13" />Refresh</button>
          <button v-if="t.prUrl" class="tb" title="Copy the link" @click="pr.copyLink"><Icon name="copy" :size="13" />Copy link</button>
          <button v-if="t.prUrl" class="tb" title="Open the PR on GitHub" @click="pr.open(t.prUrl)"><Icon name="external" :size="13" />Open PR</button>
        </span>
      </div>
      <PRTracker :task="t" :step="step" />
    </template>
    <template v-else-if="step && t">
      <div class="toolbar">
        <span v-if="chip" :class="['chip', 'badge', chip.cls]"><span class="dot"></span>{{ chip.text }}</span>
        <span v-if="jockey.sessionView === 'chat' && jockey.chatTitles[step.id]" class="small ttl" :title="jockey.chatTitles[step.id]">{{ jockey.chatTitles[step.id] }}</span>
        <span v-else class="muted small">{{ agent?.name ?? step.agent }} agent · {{ model(step.model) }}</span>
        <span class="pill views" role="radiogroup" aria-label="Session view">
          <button role="radio" :aria-checked="jockey.sessionView === 'terminal'" :class="{ on: jockey.sessionView === 'terminal' }" title="Terminal view" @click="jockey.setSessionView('terminal')"><Icon name="terminal" :size="13" /></button>
          <button role="radio" :aria-checked="jockey.sessionView === 'chat'" :class="{ on: jockey.sessionView === 'chat' }" title="Chat view" @click="jockey.setSessionView('chat')"><Icon name="chat" :size="13" /></button>
        </span>
        <span class="tbs">
          <button v-if="step.live" class="tb" title="Stop and copy the resume command" @click="jockey.stopAndCopy(step)">
            <Icon name="stop" :size="14" />Stop and copy <span class="mono-s">claude --resume {{ step.session.slice(0, 8) }}</span>
          </button>
          <template v-else>
            <button class="tb" :title="'Copy ' + resumeCommand(step)" @click="jockey.copyResume(step)">
              <Icon name="copy" :size="14" />Copy <span class="mono-s">claude --resume {{ step.session.slice(0, 8) }}</span>
            </button>
            <button v-if="step.paused" class="tb" title="Resume" @click="jockey.resumeStep(step)"><Icon name="play" :size="13" />Resume</button>
          </template>
        </span>
      </div>
      <ChatView v-if="jockey.sessionView === 'chat'" :step="step" />
      <Terminal v-else :key="step.id + ':' + jockey.termRev" :step="step" />
    </template>
    <div v-else-if="t" class="empty">
      <span class="ico"><Icon name="terminal" :size="26" /></span>
      <span class="title">{{ t.steps.length ? 'No step open for this task' : 'No session yet' }}</span>
      <span class="text">
        <template v-if="t.steps.length">Pick a step on the right to open it in a tab.</template>
        <template v-else-if="t.issue">
          Press {{ t.next.options[0]?.label ?? 'Plan' }} in the left panel: Claude reads issue #{{ t.issue }}{{ t.description ? ' and your description' : '' }} in {{ t.projectName }}.
        </template>
        <template v-else>Press {{ t.next.options[0]?.label ?? 'Plan' }} in the left panel: Claude works from the description only.</template>
      </span>
    </div>
    <div v-else class="empty">
      <span class="ico"><Icon name="terminal" :size="26" /></span>
      <span class="title">No task selected</span>
      <span class="text">Create one with New task, or pick a task on the left.</span>
    </div>

    <Toasts />
  </section>
</template>

<style scoped>
section {
  flex: 1 1 0;
  min-width: 320px;
  display: flex;
  flex-direction: column;
  background: var(--bg);
  position: relative;
}
.tabs {
  display: flex;
  align-items: stretch;
  flex: none;
  height: 32px; /* Zed's tab bar */
  background: var(--chrome);
  overflow-x: auto;
  overflow-y: hidden;
}
.tabs::-webkit-scrollbar {
  height: 0;
}
/* As Zed's tab bar: the inactive tabs and the empty space draw the line under
   it, the active tab does not, so it opens into the toolbar below (the bar
   scrolls sideways, which would clip a tab overlapping a line of the bar's own). */
.tab {
  flex: none;
  display: flex;
  align-items: center;
  border-right: 1px solid var(--border);
  border-bottom: 1px solid var(--border);
  background: var(--chrome);
  color: var(--muted);
}
.tab.on {
  background: var(--bg);
  color: var(--text);
  border-bottom-color: transparent;
}
.tabs > .grow {
  border-bottom: 1px solid var(--border);
}
.open {
  display: flex;
  align-items: center;
  gap: 7px;
  height: 100%;
  padding: 0 4px 0 12px;
  border: 0;
  background: transparent;
  color: inherit;
  white-space: nowrap;
  font-size: var(--fs-13);
}
.proj {
  font-size: var(--fs-11-5);
  color: var(--muted);
}
.dot {
  width: 6px;
  height: 6px;
}
.k-live {
  background: var(--running-dot);
}
.k-approval {
  background: var(--waiting-dot);
}
.k-off {
  background: var(--faint);
}
.x {
  width: 20px;
  height: 20px;
  margin-right: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--muted);
}
.x:hover {
  background: var(--hover);
  color: var(--text);
}
/* as Zed's default: a tab's close button shows on hover, and on the active tab */
.tab:not(.on):not(:hover) .x {
  visibility: hidden;
}
.grow {
  flex: 1;
}
.toolbar {
  display: flex;
  flex-wrap: wrap; /* narrow windows: the buttons move to a second line instead of being cut */
  align-items: center;
  gap: 4px 10px;
  flex: none;
  min-height: 44px; /* Zed's toolbar: 32px rows, 6px above and below */
  padding: 6px 12px;
  background: var(--bg); /* the active tab's: the two read as one ⊥ */
  border-bottom: 1px solid var(--border);
  min-width: 0;
}
.chip {
  font-size: var(--fs-11-5);
  align-self: center;
}
.chip .dot {
  background: var(--dot);
}
.small {
  font-size: var(--fs-12);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}
.tbs {
  flex: none;
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
}
.tb {
  flex: none;
  display: flex;
  align-items: center;
  gap: 6px;
  height: 22px;
  padding: 0 8px;
  border: 1px solid transparent; /* Zed: ghost buttons, a fill on hover */
  border-radius: 4px;
  background: transparent;
  color: var(--text);
  font-size: var(--fs-12);
  white-space: nowrap;
}
.mono-s {
  font-family: var(--mono);
  font-size: var(--fs-11-5);
  color: var(--muted);
}
.tb:hover {
  background: var(--hover);
  color: var(--text);
}
.ttl {
  flex: 1 1 0; /* shrinks to an ellipsis before the buttons wrap */
  min-width: 80px;
  font-weight: 500;
}
.views {
  flex: none;
  height: 22px;
}
.views > button {
  padding: 0 7px;
}
.empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 48px;
  text-align: center;
}
.ico {
  color: var(--faint);
  display: flex;
}
.title {
  font-size: var(--fs-14);
}
.text {
  font-size: var(--fs-12-5);
  color: var(--muted);
  max-width: 380px;
  line-height: 1.5;
}
</style>
