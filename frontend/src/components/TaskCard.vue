<script setup lang="ts">
// A task in the left panel: its reference and title, its status and its PR.
// In a sister family it also names its project, and the big sister (the task
// that started the others) wears a crown.
import { herd } from '../lib/state'
import Icon from './ui/Icon.vue'
import TaskRef from './ui/TaskRef.vue'
import StatusBadge from './ui/StatusBadge.vue'
import type { Task } from '../lib/types'

defineProps<{ task: Task; family?: boolean; big?: boolean }>()

const prTitle = (t: Task) => {
  const b = herd.prBadge(t)!
  const what = { Failed: 'CI failed', Running: 'CI running', Passed: 'CI passed', 'No CI': 'no CI checks yet', Conflict: 'conflicts with its base' }[b.label] ?? b.label.toLowerCase()
  return `PR #${t.pr}: ${what}` + (b.ready === 'draft' ? ', a draft' : b.ready === 'ready' ? ', ready for review' : '')
}

// GitHub's PR icons: draft, open (ready for review), merged, closed.
const prIcon = (t: Task) => (t.prState === 'merged' ? 'rebase' : t.prState && t.prState !== 'open' ? 'pr-closed' : t.draft ? 'pr-draft' : 'pr')
</script>

<template>
  <button class="card" :class="{ on: task.id === herd.sel }" @click="herd.pickTask(task.id)">
    <span class="row">
      <span v-if="big" class="crown" title="The big sister: it started the other tasks"><Icon name="crown" :size="12" /></span>
      <TaskRef :issue="task.issue" :task="task.id" />
      <span class="title">{{ task.title }}</span>
      <span v-if="family" class="proj">{{ task.projectName }}</span>
    </span>
    <span class="row">
      <StatusBadge :kind="task.status.kind" :label="task.status.label" />
      <span v-if="herd.prBadge(task)" :class="['badge', 'pr', `s-${herd.prBadge(task)!.kind}`]" :title="prTitle(task)">
        <Icon :name="prIcon(task)" :size="12" />#{{ task.pr }} · {{ herd.prBadge(task)!.label }}
      </span>
    </span>
  </button>
</template>

<style scoped>
.card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  flex: none;
  text-align: left;
  padding: 8px 10px;
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
.crown {
  flex: none;
  display: flex;
  color: var(--waiting);
}
</style>
