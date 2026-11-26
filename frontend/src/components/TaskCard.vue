<script setup lang="ts">
// A task in the left panel: its reference and title, its status and its PR.
// In a sister family it also names its project, says what its part of the
// work does on hover, and the big sister (the task that started the others)
// has a flag. The Archive of every project names each task's project too.
import { jockey, taskName } from '../lib/state'
import Icon from './ui/Icon.vue'
import TaskRef from './ui/TaskRef.vue'
import StatusBadge from './ui/StatusBadge.vue'
import type { Task } from '../lib/types'

defineProps<{ task: Task; family?: boolean; big?: boolean; project?: boolean }>()

// A merged PR on its way to production: where it is, and the next branch.
const mergedTitle = (t: Task) => {
  const path = [t.baseBranch, ...t.stages]
  const at = t.release || t.baseBranch
  return `PR #${t.pr} · merged · in ${at}` + (at !== path[path.length - 1] ? `, then ${path[path.indexOf(at) + 1]}` : '')
}
const prTitle = (t: Task) => {
  if (t.prState === 'merged') return mergedTitle(t)
  const b = jockey.prBadge(t)!
  const what = { Failed: 'CI failed', Running: 'CI running', Passed: 'CI passed', 'No CI': 'No CI yet', Conflict: 'Conflicts' }[b.label] ?? b.label
  return `PR #${t.pr} · ${what}` + (b.ready === 'draft' ? ' · Draft' : b.ready === 'ready' ? ' · Ready' : '')
}

// GitHub's PR icons: draft, open (ready for review), merged, closed.
const prIcon = (t: Task) => (t.prState === 'merged' ? 'rebase' : t.prState && t.prState !== 'open' ? 'pr-closed' : t.draft ? 'pr-draft' : 'pr')
</script>

<template>
  <button class="card" :class="{ on: task.id === jockey.sel }" :title="family ? task.part : undefined" @click="jockey.pickTask(task.id)">
    <span class="row">
      <span v-if="big" class="big" title="Started the sister tasks"><Icon name="flag" :size="11" /></span>
      <TaskRef :issue="task.issue" :task="task.id" />
      <span class="title">{{ taskName(task) }}</span>
      <span v-if="family || project" class="proj">{{ task.projectName }}</span>
    </span>
    <span class="row">
      <StatusBadge :kind="task.status.kind" :label="task.status.label" />
      <span v-if="jockey.prBadge(task)" :class="['badge', 'pr', `s-${jockey.prBadge(task)!.kind}`]" :title="prTitle(task)">
        <Icon :name="prIcon(task)" :size="12" />#{{ task.pr }} · {{ jockey.prBadge(task)!.label }}
      </span>
    </span>
  </button>
</template>

<style scoped>
.card {
  display: flex;
  flex-direction: column;
  gap: 7px;
  width: 100%;
  flex: none;
  text-align: left;
  padding: 10px 12px;
  border-radius: 6px;
  border: 1px solid transparent;
  background: transparent;
  font-size: var(--fs-13);
  color: var(--text);
}
.card:hover {
  background: var(--hover);
}
.card.on {
  background: var(--sel);
}
.row {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  width: 100%;
}
.row > .badge {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pr {
  flex: none;
  gap: 5px;
  margin-left: auto;
}
.title {
  flex: 1;
  min-width: 0;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.proj {
  flex: none;
  font-size: var(--fs-11-5);
  color: var(--faint);
}
.big {
  flex: none;
  display: flex;
  color: var(--muted);
}
</style>
