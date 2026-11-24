<script setup lang="ts">
// Go to task (⌘P): every word must match the title, issue number, project,
// description or LOCAL. Open tasks first, archived last.
import { computed, onMounted, ref, watch } from 'vue'
import { derby, taskName } from '../../lib/state'
import { isMac } from '../../lib/keymap'
import Icon from '../ui/Icon.vue'
import TaskRef from '../ui/TaskRef.vue'

const query = ref(derby.previewQuery)
const hi = ref(0)
const list = ref<HTMLDivElement>()
const input = ref<HTMLInputElement>()
onMounted(() => input.value?.focus())

const matches = computed(() => {
  const words = query.value.toLowerCase().trim().split(/\s+/).filter(Boolean)
  const out = derby.tasks.filter((t) => {
    const hay = `${t.name} ${t.title} ${t.issue} ${t.projectName} ${t.issue ? '' : 'local'} ${t.description} ${t.slug}`.toLowerCase()
    return words.every((w) => hay.includes(w))
  })
  return out.sort((a, b) => Number(a.archived) - Number(b.archived))
})
watch(query, () => (hi.value = 0))

function key(e: KeyboardEvent) {
  const mod = isMac() ? e.metaKey : e.ctrlKey
  const rows = matches.value
  if (e.key === 'ArrowDown') hi.value = Math.min(rows.length - 1, hi.value + 1)
  else if (e.key === 'ArrowUp') hi.value = Math.max(0, hi.value - 1)
  else if (e.key === 'Enter' && rows[hi.value]) {
    derby.pickTask(rows[hi.value].id)
    if (mod) void derby.openWorktree()
  } else return
  e.preventDefault()
  list.value?.querySelectorAll('button')[hi.value]?.scrollIntoView({ block: 'nearest' })
}
</script>

<template>
  <button class="scrim" aria-label="Close search" @click="derby.overlay = 'none'"></button>
  <div class="box popup pop pop-small" role="dialog" aria-label="Go to task">
    <div class="search">
      <span class="muted ico"><Icon name="search" :size="14" /></span>
      <label for="pq" class="sr-only">Search tasks</label>
      <input id="pq" ref="input" v-model="query" placeholder="Search tasks in all projects…" autocomplete="off" spellcheck="false" @keydown="key" />
    </div>
    <div ref="list" class="rows">
      <button v-for="(t, i) in matches" :key="t.id" class="row" :class="{ hi: i === hi }" @click="derby.pickTask(t.id)" @mousemove="hi = i">
        <TaskRef :issue="t.issue" width="46px" :task="t.id" />
        <span class="title">{{ taskName(t) }}</span>
        <span class="proj">{{ t.projectName }}</span>
        <span :class="['badge', `s-${t.status.kind}`]"><span class="dot"></span>{{ t.status.short }}</span>
      </button>
      <span v-if="!matches.length" class="none">No tasks match “{{ query }}”.</span>
    </div>
  </div>
</template>

<style scoped>
  .box {
    overflow: hidden;
  }
  .search {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 0 16px;
    height: 46px;
    border-bottom: 1px solid var(--border);
  }
  .ico {
    display: flex;
  }
  input {
    flex: 1;
    border: 0;
    outline: none;
    background: transparent;
    color: var(--text);
    font-size: var(--fs-14);
  }
  .rows {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    padding: 6px;
    overflow: auto;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    flex: none;
    height: 36px;
    padding: 0 12px;
    border-radius: 6px;
    border: 0;
    text-align: left;
    font-size: var(--fs-13);
    color: var(--text);
    background: transparent;
  }
  .row.hi {
    background: var(--sel);
  }
  .title {
    flex: 1;
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .proj {
    font-size: var(--fs-12);
    color: var(--muted);
  }
  .badge {
    align-self: center;
    flex: none;
  }
  .badge .dot {
    width: 6px;
    height: 6px;
  }
  .none {
    padding: 16px 12px;
    color: var(--muted);
  }
</style>
