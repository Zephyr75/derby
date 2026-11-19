<script setup lang="ts">
// A task with sister tasks, or whose plan asks for some: what each part of the
// work does, in a line, and where it stands. The lines come from the plan's
// "Parts overview". The repository name opens that sister task, or creates it
// (or first adds its project) when the plan asks for it and it does not exist
// yet. Collapses to its title; herd remembers it.
import { computed, ref } from 'vue'
import { api } from '../lib/api'
import { herd } from '../lib/state'
import Icon from './ui/Icon.vue'
import StatusBadge from './ui/StatusBadge.vue'
import type { SisterOffer, Task } from '../lib/types'

const props = defineProps<{ task: Task }>()

const KEY = 'herd.sisterOverview.collapsed'
const collapsed = ref(false)
try {
  collapsed.value = localStorage.getItem(KEY) === '1'
} catch {
  /* private window: stays open */
}
function toggle() {
  collapsed.value = !collapsed.value
  try {
    localStorage.setItem(KEY, collapsed.value ? '1' : '0')
  } catch {
    /* not remembered */
  }
}

interface Row {
  key: string
  name: string
  part: string
  here?: boolean // this task
  id?: string // an existing sister task
  offer?: SisterOffer // a repository the plan asks for, with no task yet
  status?: Task['status']
}
const rows = computed<Row[]>(() => [
  { key: props.task.id, name: props.task.projectName, part: props.task.part, here: true, status: herd.task(props.task.id)?.status },
  ...props.task.sisters.map((x) => ({ key: x.id, name: x.project, part: x.part, id: x.id, status: herd.task(x.id)?.status })),
  ...props.task.next.sisters.map((o) => ({ key: 'new:' + o.repo, name: o.repo, part: o.part, offer: o })),
])

// One click per repository the plan also needs: its sister task, or first its project.
async function create(repo: string) {
  const s = await herd.attempt('Could not create the sister task', () => api.CreateSister(props.task.id, repo))
  if (s) herd.toast('Sister task created', `${repo}: ${s.title}`)
}
const addProject = () => herd.attempt('Could not add the project', () => api.AddProject())
</script>

<template>
  <div class="overview">
    <button class="head" :aria-expanded="!collapsed" @click="toggle">
      <Icon :name="collapsed ? 'chevron-right' : 'chevron'" :size="13" />
      <span class="title">Sister tasks</span>
      <span class="muted small">{{ rows.length }} parts of the same work</span>
    </button>
    <ul v-if="!collapsed">
      <li v-for="r in rows" :key="r.key" :class="{ here: r.here }">
        <span v-if="r.here" class="name here-name" title="This task">{{ r.name }}</span>
        <button v-else-if="r.id" class="name link" title="Open this sister task" @click="herd.pickTask(r.id)">{{ r.name }}<Icon name="chevron-right" :size="11" /></button>
        <button v-else-if="r.offer?.project" class="name link new" title="Create a task in that repository for the same issue, with its own worktree, plan and PR, linked to this one" @click="create(r.offer.repo)"><Icon name="plus" :size="11" />{{ r.name }}</button>
        <button v-else class="name link new" :title="`${r.name} is not a Herd project yet: pick its folder, then create the sister task`" @click="addProject"><Icon name="plus" :size="11" />{{ r.name }}</button>
        <span class="part" :title="r.part">{{ r.part }}</span>
        <StatusBadge v-if="r.status" :kind="r.status.kind" :label="r.status.short || r.status.label" />
        <span v-else-if="!r.here" class="muted small nostat">{{ r.offer?.project ? 'to create' : 'add project' }}</span>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.overview {
  flex: none;
  border-bottom: 1px solid var(--border);
  background: var(--surface);
}
.head {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  height: 28px;
  padding: 0 12px;
  border: 0;
  background: transparent;
  color: var(--muted);
  text-align: left;
}
.head:hover {
  color: var(--text);
}
.title {
  font-size: var(--fs-12);
  font-weight: 600;
  color: var(--text);
}
.small {
  font-size: var(--fs-12);
}
ul {
  margin: 0;
  padding: 0 6px 6px;
  list-style: none;
  max-height: 180px;
  overflow: auto;
}
li {
  flex: none;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 3px 6px;
  border-radius: 4px;
  font-size: var(--fs-12-5);
  line-height: 18px;
}
li.here {
  background: var(--sel);
}
.name {
  flex: none;
  width: 140px;
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 18px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--text);
  font-family: var(--mono);
  font-size: var(--fs-12);
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.link {
  cursor: pointer;
}
.link:hover {
  text-decoration: underline;
}
.link.new {
  color: var(--muted);
}
.part {
  flex: 1;
  min-width: 0;
  color: var(--muted);
  overflow-wrap: anywhere; /* shown in full: wraps instead of cutting the end */
}
li > :deep(.badge),
.nostat {
  flex: none;
}
</style>
