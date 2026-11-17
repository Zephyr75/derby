<script setup lang="ts">
// A task with sister tasks: what each part of the work does, in a line, and
// where it stands. The line comes from the plan's "Parts overview". Click a
// row to open that task. Collapses to its title; herd remembers it.
import { computed, ref } from 'vue'
import { herd } from '../lib/state'
import { groupOf } from '../lib/sisters'
import Icon from './ui/Icon.vue'
import StatusBadge from './ui/StatusBadge.vue'
import type { Task } from '../lib/types'

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

const color = computed(() => groupOf.value.get(props.task.id)?.color)
const rows = computed(() =>
  [{ id: props.task.id, project: props.task.projectName, part: props.task.part }, ...props.task.sisters].map((r) => ({ ...r, here: r.id === props.task.id, status: herd.task(r.id)?.status })),
)
</script>

<template>
  <div class="overview" :class="color !== undefined ? 'sis-' + color : ''">
    <button class="head" :aria-expanded="!collapsed" @click="toggle">
      <Icon :name="collapsed ? 'chevron-right' : 'chevron'" :size="13" />
      <span class="title">Sister tasks</span>
      <span class="muted small">{{ rows.length }} parts of the same work</span>
    </button>
    <ul v-if="!collapsed">
      <li v-for="r in rows" :key="r.id" :class="{ here: r.here }">
        <button class="row" :aria-current="r.here ? 'true' : undefined" :title="r.here ? 'This task' : 'Open this sister task'" @click="!r.here && herd.pickTask(r.id)">
          <span class="proj">{{ r.project }}</span>
          <span class="part">{{ r.part }}</span>
          <StatusBadge v-if="r.status" :kind="r.status.kind" :label="r.status.short || r.status.label" />
        </button>
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
  font-size: 12px;
  font-weight: 600;
  color: var(--text);
}
.small {
  font-size: 12px;
}
ul {
  margin: 0;
  padding: 0 6px 6px;
  list-style: none;
  max-height: 140px;
  overflow: auto;
}
li {
  flex: none;
}
.row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 26px;
  padding: 0 6px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--text);
  text-align: left;
  font-size: 12.5px;
}
.row:hover {
  background: var(--hover);
}
.here .row {
  background: var(--sel);
  cursor: default;
}
.proj {
  flex: none;
  min-width: 110px;
  font-family: var(--mono);
  font-size: 12px;
}
.part {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--muted);
}
.row :deep(.badge) {
  flex: none;
}
</style>
