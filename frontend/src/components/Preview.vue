<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { api } from '../lib/api'
import { derby } from '../lib/state'
import Icon from './ui/Icon.vue'
import ChangesView from './Changes.vue'
import PRTracker from './PRTracker.vue'
import ArtifactView from './ArtifactView.vue'
import TodoList from './TodoList.vue'
import { useCompact } from '../lib/compact'
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
// The four tabs always show, in equal parts; Changes is off while the step
// changed no files, PR while the task has none, and Report stands in for them.
const task = computed(() => derby.selTask)
const openTodos = computed(() => task.value?.todos.filter((x) => !x.done).length ?? 0)
const hasChanges = computed(() => !!(step.value && changes.value?.files.length))
const tab = computed(() =>
  derby.rightTab === 'todos' && task.value ? 'todos' : derby.rightTab === 'pr' && task.value?.pr ? 'pr' : hasChanges.value && derby.rightTab === 'changes' ? 'changes' : 'preview',
)
const diffTitle = computed(() => {
  if (!step.value) return 'Open a step to see the files it changed'
  if (changes.value && !changes.value.files.length) return `${step.value.name} changed no files`
  if (step.value.external) return `Every change of the branch against ${task.value?.baseBranch}, as the PR shows it`
  return `The files ${step.value.name} changed`
})

// The tabs keep only their icons when one of their names doesn't fit.
const bar = ref<HTMLDivElement>()
const compact = useCompact(bar, (el) => [...el.children] as HTMLElement[])
</script>

<template>
  <div class="wrap">
    <div ref="bar" class="bar" :class="{ compact }" role="tablist">
      <!-- a div, not a button: it holds the button that opens the report in a pop-up -->
      <div class="tab" role="tab" tabindex="0" :aria-selected="tab === 'preview'" :class="{ on: tab === 'preview' }" :title="file ? `${file}, the report this step wrote` : 'The report a step writes'" aria-label="Report" @click="derby.rightTab = 'preview'" @keydown.enter.space.prevent="derby.rightTab = 'preview'">
        <span class="in"
          ><Icon name="file" /><span class="label">Report</span
          ><button v-if="step && art?.exists" class="expand" aria-label="Open the report in a large pop-up" title="Open the report in a large pop-up" @click.stop="derby.openArtifactPopup(step.id)" @keydown.stop><Icon name="maximize" :size="12" /></button
        ></span>
      </div>
      <button class="tab" role="tab" :aria-selected="tab === 'changes'" :aria-disabled="!hasChanges" :class="{ on: tab === 'changes', off: !hasChanges }" :title="diffTitle" aria-label="Changes" @click="hasChanges && (derby.rightTab = 'changes')">
        <span class="in"><Icon name="diff" /><span class="label">Changes</span><span v-if="hasChanges" class="count">{{ changes!.files.length }}</span></span>
      </button>
      <button class="tab" role="tab" :aria-selected="tab === 'pr'" :aria-disabled="!task?.pr" :class="{ on: tab === 'pr', off: !task?.pr }" :title="task?.pr ? `PR #${task.pr}: its review comments and CI checks` : 'No PR yet: Commit and push opens one'" aria-label="PR" @click="task?.pr && (derby.rightTab = 'pr')">
        <span class="in"><Icon name="pr" /><span class="label">PR</span><span v-if="task?.pr" class="count">#{{ task.pr }}</span></span>
      </button>
      <button class="tab" role="tab" :aria-selected="tab === 'todos'" :aria-disabled="!task" :class="{ on: tab === 'todos', off: !task }" :title="`The task's todos${openTodos ? `, ${openTodos} open` : ''}`" aria-label="TODOs" @click="task && (derby.rightTab = 'todos')">
        <span class="in"><Icon name="todo" /><span class="label">TODOs</span><span v-if="openTodos" class="count">{{ openTodos }}</span></span>
      </button>
    </div>
    <div class="content">
      <template v-if="tab === 'todos' && task">
        <div class="todo-head">
          <span class="todo-title">Todos for this task</span>
          <span class="muted small">{{ openTodos }} open · {{ task.todos.length - openTodos }} done</span>
          <span class="grow"></span>
          <label class="hide-done"><input v-model="derby.hideDone" type="checkbox" />Hide done</label>
        </div>
        <TodoList :todos="task.todos" :hide-done="derby.hideDone" @change="derby.setTodos(task, $event)" :run="derby.todoAgent(task)?.name ?? 'Implement'" :blocked="derby.todoBlocked(task)" @run="derby.runTodo(task, $event)" />
      </template>
      <PRTracker v-else-if="tab === 'pr' && task" :task="task" />
      <ChangesView v-else-if="tab === 'changes'" :changes="changes" />
      <p v-else-if="!step" class="empty">Open a step to preview the artifact it wrote.</p>
      <p v-else-if="!art || !art.exists" class="empty">{{ file || 'The artifact' }} has not been written yet. It appears here as soon as {{ step.external ? "Copilot's review arrives" : 'Claude writes it' }}.</p>
      <template v-else>
        <div class="md-file">{{ file }}</div>
        <ArtifactView :art="art" :task="step.taskId" />
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
  /* Report, Changes, PR, TODOs in four equal parts, like the left panel's tabs: the
     open one has no line between it and its content */
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
    background: var(--chrome);
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
  .tab.off {
    color: var(--faint);
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
  /* opens the report in a pop-up; sized and coloured like a count */
  .expand {
    display: flex;
    margin: 0 -2px 0 4px;
    padding: 2px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--faint);
  }
  .on .expand {
    color: var(--muted);
  }
  .tab .expand:hover {
    background: var(--hover);
    color: var(--text);
  }
  .grow {
    flex: 1;
  }
  /* the artifact's file name */
  .md-file {
    margin-top: -2px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--muted);
    font-size: var(--fs-12);
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
  .todo-head {
    display: flex;
    align-items: baseline;
    gap: 10px;
    padding: 0 6px 8px;
    border-bottom: 1px solid var(--border);
  }
  .todo-title {
    font-weight: 600;
    font-size: var(--fs-14);
  }
  .small {
    font-size: var(--fs-12);
  }
  .hide-done {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: var(--fs-12);
    color: var(--muted);
  }
  .empty {
    margin: 0;
    color: var(--muted);
    font-size: var(--fs-12-5);
    line-height: 1.5;
  }
</style>
