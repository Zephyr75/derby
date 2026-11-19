<script setup lang="ts">
import { computed } from 'vue'
import { api } from '../lib/api'
import { derby } from '../lib/state'
import Icon from './ui/Icon.vue'
import TaskRef from './ui/TaskRef.vue'

const t = computed(() => derby.selTask)
const nx = computed(() => t.value?.next)

async function restore() {
  const task = t.value
  if (task) await derby.restoreTask(task)
}
// Deleting is for good: derby forgets the task and removes its worktree, local
// branch and run files. Two clicks.
async function deleteTask() {
  const task = t.value
  if (!task) return
  if (derby.confirmDelete !== task.id) {
    derby.confirmDelete = task.id
    return
  }
  derby.confirmDelete = ''
  const lines = await derby.attempt('Could not delete the task', () => api.DeleteTask(task.id))
  if (lines) derby.toast('Deleted ' + (task.issue ? '#' + task.issue : task.title), lines.join('\n'))
}
const allow = (id: string) => derby.attempt('Allow once failed', () => api.AllowOnce(id))
const markDone = (id: string) => derby.attempt('Mark done failed', () => api.MarkDone(id))
</script>

<template>
  <div v-if="t && nx" class="box">
    <div class="top">
      <span class="muted small">Next step</span>
      <span class="grow"></span>
      <TaskRef :issue="t.issue" :task="t.id" class="tref" />
      <span class="ref">· {{ t.projectName }}</span>
    </div>

    <template v-if="nx.options.length && !t.archived">
      <label for="guide" class="sr-only">Guidance for Claude</label>
      <p v-if="nx.warning" class="warn"><Icon name="warn" :size="14" />{{ nx.warning }}</p>
      <p v-if="nx.elsewhere.length" class="warn">
        <Icon name="warn" :size="14" />Steps also edited files outside the worktree, in {{ nx.elsewhere.join(', ') }}. Derby only commits {{ t.branch }}: check those edits yourself.
      </p>
      <textarea id="guide" v-model="derby.guidance" rows="3" :class="{ need: nx.warning && !derby.guidance.trim() }" :placeholder="nx.warning ? 'Which option? e.g. take option B, because…' : 'Guidance for Claude (optional)'"></textarea>
      <div class="opts">
        <button v-for="(o, i) in nx.options" :key="o.agent + o.n" class="btn" :class="{ primary: i === 0 }" @click="derby.run(o)"><Icon name="play" :size="13" />{{ o.label }}</button>
      </div>
    </template>

    <p v-if="nx.text">{{ nx.text }}</p>
    <button v-if="derby.prReady(t) === 'draft' && derby.copilotClean(t)" class="btn" title="Turn the draft PR into one ready for review and add the ready label" @click="derby.markReady(t)"><Icon name="check" :size="13" />Mark PR #{{ t.pr }} ready for review</button>
    <p v-else-if="derby.prReady(t) === 'ready'" class="muted small">PR #{{ t.pr }} is ready for review.</p>

    <button v-if="nx.approval" class="btn allow" @click="allow(nx.approval)">Allow once</button>
    <button v-if="nx.markDone" class="btn ghost-border" @click="markDone(nx.markDone)">Mark done</button>

    <p v-if="t.projectGone">{{ t.projectName }} is no longer in Settings &gt; Projects, so Derby cannot restore this task or touch its repository. Delete task only removes it from derby.</p>
    <div v-if="t.archived" class="row">
      <button v-if="!t.worktreeRemoved && !t.projectGone" class="btn" @click="restore">Restore to Tasks</button>
      <button
        class="btn"
        :class="derby.confirmDelete === t.id ? 'danger' : 'transparent'"
        title="Forget the task in Derby, and remove its worktree, local branch and run files (the pushed branch and the PR stay on GitHub)"
        @click="deleteTask"
      >
        {{ derby.confirmDelete === t.id ? 'Delete for good? This cannot be undone' : t.projectGone ? 'Delete task' : 'Delete task and worktree' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.box {
  flex: none;
  border-top: 1px solid var(--border);
  padding: 10px 12px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.top {
  display: flex;
  align-items: center;
  gap: 8px;
}
.small {
  font-size: var(--fs-12);
}
.ref,
.tref {
  font-family: var(--mono);
  font-size: var(--fs-11);
  color: var(--muted);
}
.grow {
  flex: 1;
}
textarea {
  resize: none;
  background: var(--surface);
  color: var(--text);
  border: 1px solid var(--border);
  border-radius: 6px;
  padding: 7px 9px;
  font-size: var(--fs-13);
  line-height: 1.45;
}
.warn {
  display: flex;
  gap: 7px;
  align-items: flex-start;
  margin: 0;
  padding: 7px 9px;
  border-radius: 6px;
  background: var(--waiting-bg);
  color: var(--waiting);
  font-size: var(--fs-12-5);
  line-height: 1.45;
}
.warn :deep(svg) {
  flex: none;
  margin-top: 2px;
  color: var(--waiting-dot);
}
textarea.need {
  border-color: var(--waiting-dot);
}
textarea:focus {
  outline: none;
}
.opts,
.row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}
p {
  margin: 0;
  font-size: var(--fs-12-5);
  line-height: 1.5;
  color: var(--muted);
}
.allow {
  align-self: flex-start;
  border-color: var(--approval-dot);
  background: var(--approval-bg);
  color: var(--approval);
}
.ghost-border {
  align-self: flex-start;
  background: transparent;
}
.transparent {
  background: transparent;
}
</style>
