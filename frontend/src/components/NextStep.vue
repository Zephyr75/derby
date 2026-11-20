<script setup lang="ts">
import { computed } from 'vue'
import { api } from '../lib/api'
import { derby } from '../lib/state'
import Icon from './ui/Icon.vue'

const t = computed(() => derby.selTask)
const nx = computed(() => t.value?.next)
const markReady = computed(() => !!t.value && derby.prReady(t.value) === 'draft' && derby.copilotClean(t.value))
const runnable = computed(() => !!t.value && !t.value.archived && !!nx.value?.options.length)
// Nothing to act on, e.g. while an agent runs: the box hides.
const show = computed(() => {
  const task = t.value
  const n = nx.value
  if (!task || !n) return false
  return task.archived || task.projectGone || runnable.value || markReady.value || !!n.sisters.length || !!n.approval || !!n.markDone
})

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
// One click per repository the plan also needs: its sister task, or first its project.
async function createSister(repo: string) {
  const task = t.value
  if (!task) return
  const s = await derby.attempt('Could not create the sister task', () => api.CreateSister(task.id, repo))
  if (s) derby.toast('Sister task created', `${repo}: ${s.title}`)
}
const addProject = () => derby.attempt('Could not add the project', () => api.AddProject())
</script>

<template>
  <div v-if="t && nx && show" class="box">

    <div v-for="o in nx.sisters" :key="o.repo" class="sister">
      <p :title="o.scope"><b>The plan also needs {{ o.repo }}</b>: {{ o.part || o.scope }}</p>
      <button v-if="o.project" class="btn primary" :title="`Create a task in ${o.repo} for the same issue, with its own worktree, plan and PR, linked to this one under Sisters`" @click="createSister(o.repo)"><Icon name="plus" :size="13" /><span>Create sister task</span></button>
      <button v-else class="btn" :title="`${o.repo} is not a Derby project yet: pick its folder, then create the sister task`" @click="addProject"><Icon name="plus" :size="13" /><span>Add as a project</span></button>
    </div>

    <template v-if="runnable">
      <label for="guide" class="sr-only">Guidance for Claude</label>
      <p v-if="nx.warning" class="warn"><Icon name="warn" :size="14" />{{ nx.warning }}</p>
      <p v-if="nx.elsewhere.length" class="warn">
        <Icon name="warn" :size="14" />Steps also edited files outside the worktree, in {{ nx.elsewhere.join(', ') }}. Derby only commits {{ t.branch }}: check those edits yourself.
      </p>
      <textarea id="guide" v-model="derby.guidance" rows="3" :class="{ need: nx.warning && !derby.guidance.trim() }" :placeholder="nx.warning ? 'Which option? e.g. take option B, because…' : 'Guidance for Claude (optional)'"></textarea>
    </template>

    <p v-if="t.projectGone">{{ t.projectName }} is no longer in Settings &gt; Projects, so Derby cannot restore this task or touch its repository. Delete task only removes it from derby.</p>

    <!-- Each button takes half the width, or a full line when its label needs more. -->
    <div class="opts">
      <template v-if="runnable">
        <button v-for="(o, i) in nx.options" :key="o.agent + o.n" class="btn" :class="{ primary: i === 0 }" :title="o.label" @click="derby.run(o)"><Icon name="play" :size="13" /><span>{{ o.label }}</span></button>
      </template>
      <button v-if="markReady" class="btn" title="Turn the draft PR into one ready for review and add the ready label" @click="derby.markReady(t)"><Icon name="check" :size="13" /><span>Mark PR ready for review</span></button>
      <button v-if="nx.approval" class="btn allow" @click="allow(nx.approval)"><span>Allow once</span></button>
      <button v-if="nx.markDone" class="btn" @click="markDone(nx.markDone)"><span>Mark done</span></button>
      <template v-if="t.archived">
        <button v-if="!t.worktreeRemoved && !t.projectGone" class="btn" @click="restore"><span>Restore to Tasks</span></button>
        <button
          class="btn"
          :class="{ danger: derby.confirmDelete === t.id }"
          title="Forget the task in Derby, and remove its worktree, local branch and run files (the pushed branch and the PR stay on GitHub)"
          @click="deleteTask"
        >
          <span>{{ derby.confirmDelete === t.id ? 'Delete for good? This cannot be undone' : t.projectGone ? 'Delete task' : 'Delete task and worktree' }}</span>
        </button>
      </template>
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
.opts {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.opts:empty {
  display: none;
}
/* Half the width each; a label wider than that keeps its width (the automatic
   minimum), wraps to its own line, and grows to fill it. */
.opts .btn {
  flex: 1 1 calc(50% - 3px);
  max-width: 100%;
  justify-content: center;
}
.opts .btn:not(.primary):not(.danger):not(.allow) {
  border-color: var(--border);
}
.opts .btn span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}
p {
  margin: 0;
  font-size: var(--fs-12-5);
  line-height: 1.5;
  color: var(--muted);
}
.allow {
  border-color: var(--approval-dot);
  background: var(--approval-bg);
  color: var(--approval);
}
.sister {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  padding: 8px 10px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--surface);
}
.sister p {
  color: var(--text);
  overflow-wrap: anywhere;
}
.sister .btn {
  max-width: 100%;
}
.sister .btn span {
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
