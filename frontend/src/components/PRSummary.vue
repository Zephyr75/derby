<script setup lang="ts">
// The PR tab: Copilot's review and the CI checks a line each, the failing
// checks under them. The full view (the Copilot step, in the center) has the
// comments to send to a fix and the checks to rerun.
import { computed, reactive } from 'vue'
import { usePR } from '../lib/pr'
import type { Task } from '../lib/types'
import Icon from './ui/Icon.vue'
import PaneHead from './ui/PaneHead.vue'

const props = defineProps<{ task: Task }>()
const pr = reactive(usePR(props))

const state = computed(() =>
  props.task.prState === 'merged' ? 'Merged' : props.task.prState === 'closed' ? 'Closed' : pr.ready === 'draft' ? 'Draft' : pr.ready === 'ready' ? 'Ready' : '',
)

const plural = (n: number, w: string) => `${n} ${w}${n === 1 ? '' : 's'}`
const review = computed(() => {
  const open = pr.openFb.length
  const done = pr.resolvedFb.length
  const resolved = done ? `${done} resolved` : ''
  if (pr.waiting) return { dot: 'd-run', text: ['waiting for Copilot', open && `${open} open`].filter(Boolean).join(' · ') }
  if (open) return { dot: 'd-wait', text: [`${open} open`, resolved].filter(Boolean).join(' · ') }
  return { dot: done ? 'd-pass' : 'd-skip', text: resolved || 'no comments' }
})
const checks = computed(() => {
  const n = pr.counts
  const passed = `${n.pass}/${n.total} passed`
  if (!n.total) return { dot: 'd-skip', text: props.task.conflicting ? 'none until the conflicts are resolved' : 'none yet' }
  if (n.fail) return { dot: 'd-fail', text: `${n.fail} failed · ${passed}` }
  if (n.pending) return { dot: 'd-run', text: `${n.pending} running · ${passed}` }
  return { dot: 'd-pass', text: passed }
})
const FAILING = 3
const failing = computed(() => pr.failing.slice(0, FAILING))

// A line opens the full view, or the PR on GitHub without a Copilot step.
const full = () => (pr.cop ? pr.openStep(pr.cop.id) : pr.open(props.task.prUrl))
const fullTip = computed(() => (pr.cop ? 'Open full PR view' : 'Open on GitHub'))
</script>

<template>
  <PaneHead :title="`#${task.pr}`" :meta="state" :tip="`${task.branch} → ${task.baseBranch}`">
    <button v-if="pr.ready === 'draft'" title="Mark ready for review" aria-label="Mark ready" @click="pr.markReady"><Icon name="check" :size="13" /></button>
    <button v-if="pr.live" :disabled="pr.refreshing" title="Refresh now" aria-label="Refresh" @click="pr.refresh"><Icon name="rerun" :size="13" /></button>
    <button v-if="pr.cop" title="Open full PR view" aria-label="Open the full PR view" @click="pr.openStep(pr.cop.id)"><Icon name="maximize" :size="13" /></button>
    <button v-if="task.prUrl" title="Open on GitHub" aria-label="Open on GitHub" @click="pr.open(task.prUrl)"><Icon name="external" :size="13" /></button>
  </PaneHead>
  <div class="rows">
    <button class="row" :title="fullTip" @click="full">
      <span :class="['sd', review.dot]"></span><span class="nm">Review</span><span class="val">{{ review.text }}</span>
    </button>
    <button class="row" :title="fullTip" @click="full">
      <span :class="['sd', checks.dot]"></span><span class="nm">Checks</span><span class="val">{{ checks.text }}</span>
    </button>
    <button v-for="c in failing" :key="c.workflow + c.name" class="row sub" :title="c.link ? 'Open on GitHub' : ''" @click="pr.open(c.link)">
      <span class="sd d-fail"></span><span class="nm">{{ c.workflow ? `${c.workflow} / ${c.name}` : c.name }}</span>
    </button>
    <button v-if="pr.failing.length > FAILING" class="row sub more" :title="fullTip" @click="full">
      <span class="nm">{{ plural(pr.failing.length - FAILING, 'more failed check') }}</span>
    </button>
    <button v-if="pr.live && task.conflicting" class="row" :title="fullTip" @click="full">
      <span class="sd d-fail"></span><span class="nm">Conflicts</span><span class="val">with {{ task.baseBranch }}</span>
    </button>
  </div>
</template>

<style scoped>
  .rows {
    display: flex;
    flex-direction: column;
    margin: 0 -6px;
  }
  /* a line, like a changed file */
  .row {
    display: flex;
    align-items: center;
    gap: 9px;
    height: 26px;
    padding: 0 6px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--text);
    text-align: left;
    white-space: nowrap;
    font-size: var(--fs-12-5);
  }
  .row:hover {
    background: var(--hover);
  }
  .nm {
    flex: none;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .val {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    text-align: right;
    color: var(--muted);
    font-size: var(--fs-12);
  }
  .sd {
    flex: none;
    width: 7px;
    height: 7px;
    margin: 0 4px;
    border-radius: 50%;
  }
  .d-fail { background: var(--approval-dot); }
  .d-wait { background: var(--waiting-dot); }
  .d-run { background: var(--running-dot); }
  .d-pass { background: var(--done-dot); }
  .d-skip { background: var(--faint); }
  .row.sub {
    height: 24px;
    padding-left: 24px;
    color: var(--muted);
    font-size: var(--fs-12);
  }
  .row.sub .nm {
    flex: 1;
  }
  .row.sub .sd {
    width: 5px;
    height: 5px;
    margin: 0 5px;
  }
  .row.more {
    padding-left: 45px;
    color: var(--faint);
  }
</style>
