<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { derby, resumeCommand } from '../lib/state'
import Icon from './ui/Icon.vue'
import Terminal from './Terminal.vue'
import ChatView from './ChatView.vue'
import PRTracker from './PRTracker.vue'
import Toasts from './Toasts.vue'

// One tab per task: its dot says whether one of its steps waits for approval
// or runs, whichever step the tab shows.
const tabs = computed(() =>
  derby.tabs.map((tb, i) => {
    const t = derby.task(tb.task)
    const steps = t?.steps ?? []
    const kind = steps.some((s) => s.state === 'approval') ? 'approval' : steps.some((s) => s.state === 'running' || s.state === 'idle') ? 'live' : 'off'
    const step = steps.find((x) => x.id === tb.step)?.name ?? ''
    return { i, id: tb.task, kind, ref: t?.issue ? '#' + t.issue : 'LOCAL', title: t?.title ?? '', tip: `${t?.title ?? ''}${step ? ` (${step})` : ''}`, project: t?.projectName ?? '', on: i === derby.active && !!derby.activeTab }
  }),
)
// Task titles make the tabs wide: the bar scrolls to the open one.
const bar = ref<HTMLElement>()
const reveal = () => bar.value?.querySelector('.tab.on')?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
watch(() => [derby.active, derby.tabs.length], reveal, { flush: 'post' })
onMounted(reveal)
const step = computed(() => derby.selStep)
const t = computed(() => derby.selTask)
const agent = computed(() => (step.value ? derby.settings?.agents.find((a) => a.id === step.value!.agent) : undefined))
const model = (m: string) => (m ? m[0].toUpperCase() + m.slice(1) : '')
const chip = computed(() => {
  const s = step.value
  if (!s) return null
  if (s.paused && !s.live) return { cls: 's-idle', text: 'Paused' }
  if (s.state === 'approval') return { cls: 's-approval', text: 'Live · waiting for approval' }
  if (s.state === 'running' || s.state === 'idle') return { cls: 's-running', text: s.state === 'idle' ? 'Live · waiting for you' : 'Live' }
  if (s.state === 'failed') return { cls: 's-waiting', text: s.live ? 'Resumed · step ended early' : 'Ended early' }
  if (!s.live && derby.sessionView === 'chat') return { cls: 's-idle', text: 'Ended' } // the chat reads it without resuming it
  return { cls: 's-idle', text: s.live ? 'Resumed' : 'Resumed on click' }
})

</script>

<template>
  <section>
    <div ref="bar" class="tabs" role="tablist">
      <div v-for="tb in tabs" :key="tb.id" class="tab" :class="{ on: tb.on }">
        <button class="open" role="tab" :aria-selected="tb.on" :title="tb.tip" @click="derby.focusTab(tb.i)">
          <span :class="['dot', `k-${tb.kind}`]"></span>
          <span class="issue-ref">{{ tb.ref }}</span>
          <span class="name">{{ tb.title }}</span>
          <span class="proj">{{ tb.project }}</span>
        </button>
        <button class="x" aria-label="Close tab" @click="derby.closeTab(tb.i)"><Icon name="close" :size="14" /></button>
      </div>
      <span class="grow"></span>
    </div>

    <PRTracker v-if="step && t && step.external" :task="t" :step="step" />
    <template v-else-if="step && t">
      <div class="toolbar">
        <span v-if="chip" :class="['chip', 'badge', chip.cls]"><span class="dot"></span>{{ chip.text }}</span>
        <span v-if="derby.sessionView === 'chat' && derby.chatTitles[step.id]" class="small ttl" :title="derby.chatTitles[step.id]">{{ derby.chatTitles[step.id] }}</span>
        <span v-else class="muted small">{{ agent?.name ?? step.agent }} agent · {{ model(step.model) }}</span>
        <span class="pill views" role="radiogroup" aria-label="Session view">
          <button role="radio" :aria-checked="derby.sessionView === 'terminal'" :class="{ on: derby.sessionView === 'terminal' }" title="Show the session as Claude Code's terminal" @click="derby.setSessionView('terminal')"><Icon name="terminal" :size="13" /></button>
          <button role="radio" :aria-checked="derby.sessionView === 'chat'" :class="{ on: derby.sessionView === 'chat' }" title="Show the session as a chat" @click="derby.setSessionView('chat')"><Icon name="chat" :size="13" /></button>
        </span>
        <span class="tbs">
          <button v-if="step.live" class="tb" :title="'Stop Claude in Derby and copy ' + resumeCommand(step) + ', e.g. to continue in Zed'" @click="derby.stopAndCopy(step)">
            <Icon name="stop" :size="14" />Stop and copy <span class="mono-s">claude --resume {{ step.session.slice(0, 8) }}</span>
          </button>
          <template v-else>
            <button class="tb" :title="'Copy ' + resumeCommand(step)" @click="derby.copyResume(step)">
              <Icon name="copy" :size="14" />Copy <span class="mono-s">claude --resume {{ step.session.slice(0, 8) }}</span>
            </button>
            <button v-if="step.paused" class="tb" title="Resume this session in Derby" @click="derby.resumeStep(step)"><Icon name="play" :size="13" />Resume</button>
          </template>
        </span>
      </div>
      <ChatView v-if="derby.sessionView === 'chat'" :step="step" />
      <Terminal v-else :key="step.id + ':' + derby.termRev" :step="step" />
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
.name {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
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
