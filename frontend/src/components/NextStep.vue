<script setup lang="ts">
import { computed, onUpdated, ref, watch } from 'vue'
import { api } from '../lib/api'
import { jockey, taskName } from '../lib/state'
import Icon from './ui/Icon.vue'

const t = computed(() => jockey.selTask)
const nx = computed(() => t.value?.next)
const runnable = computed(() => !!t.value && !t.value.archived && !!nx.value?.options.length)
// Nothing to act on, e.g. while an agent runs: the box hides.
const show = computed(() => {
  const task = t.value
  const n = nx.value
  if (!task || !n) return false
  return task.archived || task.projectGone || runnable.value || !!n.sisters.length || !!n.markDone
})

async function restore() {
  const task = t.value
  if (task) await jockey.restoreTask(task)
}
// Deleting is for good: jockey forgets the task and removes its worktree, local
// branch and run files. Two clicks.
async function deleteTask() {
  const task = t.value
  if (!task) return
  if (jockey.confirmDelete !== task.id) {
    jockey.confirmDelete = task.id
    return
  }
  jockey.confirmDelete = ''
  const lines = await jockey.attempt('Could not delete the task', () => api.DeleteTask(task.id))
  if (lines) jockey.toast('Deleted ' + (task.issue ? '#' + task.issue : taskName(task)), lines.join('\n'))
}
const markDone = (id: string) => jockey.attempt('Mark done failed', () => api.MarkDone(id))
// One click per repository the plan also needs: its sister task, or first its project.
async function createSister(repo: string) {
  const task = t.value
  if (!task) return
  const s = await jockey.attempt('Could not create the sister task', () => api.CreateSister(task.id, repo))
  if (s) jockey.toast('Sister task created', `${repo}: ${s.title}`)
}
const addProject = () => jockey.attempt('Could not add the project', () => api.AddProject())

// Buttons pair up two per line; with an odd count the first, the main one,
// takes the first line alone. If a paired label does not fit in half the
// width, every button gets its own line instead.
const opts = ref<HTMLElement>()
const stack = ref(false)
function needed(b: Element) {
  const cs = getComputedStyle(b)
  const px = (v: string) => parseFloat(v) || 0
  let w = px(cs.paddingLeft) + px(cs.paddingRight) + px(cs.borderLeftWidth) + px(cs.borderRightWidth)
  for (const k of b.children) w += k.tagName === 'SPAN' ? k.scrollWidth : k.getBoundingClientRect().width
  return w + px(cs.columnGap) * Math.max(0, b.children.length - 1)
}
function fit() {
  const el = opts.value
  if (!el) return
  const btns = [...el.children]
  const paired = btns.length % 2 ? btns.slice(1) : btns
  const half = (el.clientWidth - (parseFloat(getComputedStyle(el).columnGap) || 0)) / 2
  stack.value = paired.some((b) => needed(b) > half + 0.5)
}
watch(opts, (el, _, onCleanup) => {
  if (!el) return
  let raf = 0
  const ro = new ResizeObserver(() => {
    cancelAnimationFrame(raf)
    raf = requestAnimationFrame(fit)
  })
  ro.observe(el)
  onCleanup(() => {
    ro.disconnect()
    cancelAnimationFrame(raf)
  })
})
onUpdated(fit)
</script>

<template>
  <div v-if="t && nx && show" class="box">

    <div v-for="o in nx.sisters" :key="o.repo" class="sister">
      <p :title="o.scope"><b>The plan also needs {{ o.repo }}</b>: {{ o.part || o.scope }}</p>
      <button v-if="o.project" class="btn primary" :title="`Create a sister task in ${o.repo}`" @click="createSister(o.repo)"><Icon name="plus" :size="13" /><span>Create sister task</span></button>
      <button v-else class="btn" :title="`Add ${o.repo} to Jockey first`" @click="addProject"><Icon name="plus" :size="13" /><span>Add as a project</span></button>
    </div>

    <template v-if="runnable">
      <label for="guide" class="sr-only">Guidance for Claude</label>
      <p v-if="nx.warning" class="warn"><Icon name="warn" :size="14" />{{ nx.warning }}</p>
      <p v-if="nx.elsewhere.length" class="warn">
        <Icon name="warn" :size="14" />Steps also edited files outside the worktree, in {{ nx.elsewhere.join(', ') }}. Jockey only commits {{ t.branch }}: check those edits yourself.
      </p>
      <textarea id="guide" v-model="jockey.guidance" rows="3" :class="{ need: nx.warning && !jockey.guidance.trim() }" :placeholder="nx.warning ? 'Which option? e.g. take option B, because…' : 'Guidance for Claude (optional)'"></textarea>
    </template>

    <p v-if="t.projectGone">{{ t.projectName }} is no longer in Settings &gt; Projects, so Jockey cannot restore this task or touch its repository. Delete task only removes it from jockey.</p>

    <div ref="opts" class="opts" :class="{ stack }">
      <template v-if="runnable">
        <button v-for="(o, i) in nx.options" :key="o.agent + o.n" class="btn" :class="{ primary: i === 0 }" :title="o.label" @click="jockey.run(o)"><Icon name="play" :size="13" /><span>{{ o.label }}</span></button>
      </template>
      <button v-if="nx.markDone" class="btn" @click="markDone(nx.markDone)"><span>Mark done</span></button>
      <template v-if="t.archived">
        <button v-if="!t.worktreeRemoved && !t.projectGone" class="btn" @click="restore"><span>Restore to Tasks</span></button>
        <button
          class="btn"
          :class="{ danger: jockey.confirmDelete === t.id }"
          title="The PR stays on GitHub"
          @click="deleteTask"
        >
          <span>{{ jockey.confirmDelete === t.id ? 'Delete for good? This cannot be undone' : t.projectGone ? 'Delete task' : 'Delete task and worktree' }}</span>
        </button>
      </template>
    </div>
  </div>
</template>

<style scoped>
.box {
  flex: none;
  min-height: var(--dock-h); /* as the chat's message editor beside it: their top lines meet */
  box-sizing: border-box;
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
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
}
.opts:empty {
  display: none;
}
/* an odd count: the first button has the first line to itself */
.opts > :first-child:nth-last-child(odd),
.opts.stack > * {
  grid-column: 1 / -1;
}
.opts .btn {
  min-width: 0;
  justify-content: center;
}
.opts .btn:not(.primary):not(.danger) {
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
