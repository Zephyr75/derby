<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api } from '../lib/api'
import { derby } from '../lib/state'
import Icon from './ui/Icon.vue'
import PaneHead from './ui/PaneHead.vue'
import ChangesView from './Changes.vue'
import PRSummary from './PRSummary.vue'
import ArtifactView from './ArtifactView.vue'
import TodoList from './TodoList.vue'
import { useCompact } from '../lib/compact'
import { summary } from '../lib/markdown'
import type { Artifact, Changes } from '../lib/types'

const step = computed(() => derby.selStep)
const art = ref<Artifact | null>(null)

let seq = 0
watch(
  () => [step.value?.id, step.value?.state, derby.artifactRev] as const,
  ([id]) => {
    const mine = ++seq
    if (!id) {
      art.value = null
      return
    }
    api.ReadArtifact(id).then(
      (a) => {
        if (mine === seq) art.value = a
      },
      () => {
        if (mine === seq) art.value = null
      },
    )
  },
  { immediate: true },
)

// The step's changed files, refreshed as the task changes (debounced: hooks
// fire often while a step works).
const changes = ref<Changes | null>(null)
let cseq = 0
let ctimer: ReturnType<typeof setTimeout> | undefined
function loadChanges(id: string | undefined, delay: number) {
  clearTimeout(ctimer)
  const mine = ++cseq
  if (!id) {
    changes.value = null
    return
  }
  ctimer = setTimeout(() => {
    api.StepChanges(id).then(
      (c) => mine === cseq && (changes.value = c),
      () => mine === cseq && (changes.value = { source: '', sha: '', via: '', files: [] }),
    )
  }, delay)
}
watch(
  () => step.value?.id,
  (id) => {
    changes.value = null
    loadChanges(id, 0)
  },
  { immediate: true },
)
watch(
  () => [derby.selTask, step.value?.sha, step.value?.state] as const,
  () => loadChanges(step.value?.id, 800),
)

const file = computed(() => art.value?.file || step.value?.artifact || '')
// The report's TL;DR and ELI5 only (its first paragraph without them): the
// expand button opens the whole of it.
const parts = computed(() => (art.value?.exists ? summary(art.value.body).map((p) => ({ ...p, art: { ...art.value!, body: p.md } })) : []))
// The four tabs always show, in equal parts; Changes is off while the step
// changed no files, PR while the task has none, and Report stands in for them.
const task = computed(() => derby.selTask)
const openTodos = computed(() => task.value?.todos.filter((x) => !x.done).length ?? 0)
const hasChanges = computed(() => !!(step.value && changes.value?.files.length))
const tab = computed(() =>
  derby.rightTab === 'todos' && task.value ? 'todos' : derby.rightTab === 'pr' && task.value?.pr ? 'pr' : hasChanges.value && derby.rightTab === 'changes' ? 'changes' : 'preview',
)
const diffTitle = computed(() => {
  if (!step.value) return 'No step open'
  if (changes.value && !changes.value.files.length) return 'No changes'
  if (step.value.external) return `Branch vs ${task.value?.baseBranch}`
  return `${step.value.name}'s changes`
})

// The tabs keep only their icons when one of their names doesn't fit.
const bar = ref<HTMLDivElement>()
const compact = useCompact(bar, (el) => [...el.children] as HTMLElement[])
</script>

<template>
  <div class="wrap">
    <div ref="bar" class="bar" :class="{ compact }" role="tablist">
      <button class="tab" role="tab" :aria-selected="tab === 'preview'" :class="{ on: tab === 'preview' }" :title="file || 'Report'" aria-label="Report" @click="derby.rightTab = 'preview'">
        <span class="in"><Icon name="file" /><span class="label">Report</span></span>
      </button>
      <button class="tab" role="tab" :aria-selected="tab === 'changes'" :aria-disabled="!hasChanges" :class="{ on: tab === 'changes', off: !hasChanges }" :title="diffTitle" aria-label="Changes" @click="hasChanges && (derby.rightTab = 'changes')">
        <span class="in"><Icon name="diff" /><span class="label">Changes</span><span v-if="hasChanges" class="count">{{ changes!.files.length }}</span></span>
      </button>
      <button class="tab" role="tab" :aria-selected="tab === 'pr'" :aria-disabled="!task?.pr" :class="{ on: tab === 'pr', off: !task?.pr }" :title="task?.pr ? `PR #${task.pr}` : 'No PR yet'" aria-label="PR" @click="task?.pr && (derby.rightTab = 'pr')">
        <span class="in"><Icon name="pr" /><span class="label">PR<span v-if="task?.pr" class="count">#{{ task.pr }}</span></span></span>
      </button>
      <button class="tab" role="tab" :aria-selected="tab === 'todos'" :aria-disabled="!task" :class="{ on: tab === 'todos', off: !task }" :title="openTodos ? `${openTodos} open` : 'Todos'" aria-label="TODOs" @click="task && (derby.rightTab = 'todos')">
        <span class="in"><Icon name="todo" /><span class="label">TODOs</span><span v-if="openTodos" class="count">{{ openTodos }}</span></span>
      </button>
    </div>
    <div class="content">
      <template v-if="tab === 'todos' && task">
        <PaneHead :title="`${openTodos} open`" :meta="`${task.todos.length - openTodos} done`" tip="The task's todos">
          <button v-if="task.todos.length > openTodos" :title="derby.hideDone ? 'Show done' : 'Hide done'" @click="derby.hideDone = !derby.hideDone">{{ derby.hideDone ? 'Show done' : 'Hide done' }}</button>
        </PaneHead>
        <TodoList :todos="task.todos" :hide-done="derby.hideDone" @change="derby.setTodos(task, $event)" :run="derby.todoAgent(task)?.name ?? 'Implement'" :blocked="derby.todoBlocked(task)" @run="derby.runTodo(task, $event)" />
      </template>
      <PRSummary v-else-if="tab === 'pr' && task" :task="task" />
      <ChangesView v-else-if="tab === 'changes'" :changes="changes" />
      <p v-else-if="!step" class="empty">Open a step to preview the artifact it wrote.</p>
      <template v-else>
        <PaneHead :title="file || 'Report'" :tip="art?.path">
          <button v-if="art?.exists" title="Expand" aria-label="Open the whole report" @click="derby.openArtifactPopup(step.id)"><Icon name="maximize" :size="13" /></button>
        </PaneHead>
        <p v-if="!art || !art.exists" class="empty">{{ file || 'The artifact' }} has not been written yet. It appears here as soon as {{ step.external ? "Copilot's review arrives" : 'Claude writes it' }}.</p>
        <p v-else-if="!parts.length" class="empty">The report is empty.</p>
        <section v-for="p in parts" :key="p.label" class="part">
          <div v-if="p.label" class="plabel">{{ p.label }}</div>
          <ArtifactView :art="p.art" :task="step.taskId" />
        </section>
      </template>
    </div>
  </div>
</template>

<style scoped>

  .wrap {
    flex: 1;
    display: flex;
    flex-direction: column;
    background: var(--bg);
    border-top: 1px solid var(--border);
    min-height: 0;
  }
  /* Report, Changes, PR, TODOs in four equal parts, like the left panel's tabs: on
     the panel's colour, the open one on its content's with no line between them */
  .bar {
    display: flex;
    flex: none;
    height: 32px; /* Zed's tab bar */
  }
  .tab {
    flex: 1 1 0;
    min-width: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0 4px;
    overflow: hidden;
    border: 0;
    border-bottom: 1px solid var(--border);
    background: var(--panel);
    color: var(--muted);
    white-space: nowrap;
    font-size: var(--fs-12-5);
    cursor: default;
  }
  .tab + .tab {
    border-left: 1px solid var(--border);
  }
  .tab:hover:not(.off) {
    color: var(--text);
  }
  .tab.on {
    border-bottom-color: transparent;
    background: var(--bg);
    color: var(--text);
    font-weight: 500;
  }
  .in {
    flex: none;
    display: flex;
    align-items: center;
  }
  .label {
    margin-left: 6px; /* LABEL_GAP in lib/compact.ts */
    overflow: hidden;
  }
  .compact .label {
    max-width: 0;
    margin-left: 0;
  }
  .count {
    margin-left: 6px;
    color: var(--faint);
    font-weight: 400;
  }
  .on .count {
    color: var(--muted);
  }
  /* a summary part's label, like the PR view's section names */
  .plabel {
    margin-bottom: 4px;
    color: var(--muted);
    font-size: var(--fs-12);
    font-weight: 600;
  }
  .part :deep(.md > :last-child) {
    margin-bottom: 0;
  }
  .content {
    flex: 1;
    min-height: 0;
    padding: 16px 20px 28px;
    overflow: auto;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .empty {
    margin: 0;
    color: var(--muted);
    font-size: var(--fs-12-5);
    line-height: 1.5;
  }
</style>
