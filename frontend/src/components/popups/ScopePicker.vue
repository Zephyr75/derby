<script setup lang="ts">
// The top bar's lists: a click on the project, worktree or branch (or ⌘O for
// projects) lists every project, or the task's project's worktrees or
// branches; picking one opens it in Zed.
import { computed, onMounted, ref, watch } from 'vue'
import { api } from '../../lib/api'
import { derby, type ScopeMode } from '../../lib/state'
import type { Branch, Worktree } from '../../lib/types'
import Icon from '../ui/Icon.vue'

interface Row {
  key: string
  label: string
  detail: string
  current: boolean
  open: () => Promise<void>
}

const t = computed(() => derby.selTask)
const project = computed(() => t.value?.project ?? derby.projects[0]?.id ?? '')
const pname = computed(() => derby.project(project.value)?.name ?? project.value)
const worktrees = ref<Worktree[]>()
const branches = ref<Branch[]>()
const filter = ref('')
const hi = ref(0)
const input = ref<HTMLInputElement>()
const at = ref<number>()

onMounted(() => {
  input.value?.focus()
  // Under the clicked crumb.
  const crumb = document.querySelector('header .crumb.on')
  if (crumb) at.value = crumb.getBoundingClientRect().left
})

async function load(mode: ScopeMode) {
  const id = project.value
  if (!id) return
  if (mode === 'worktree' && !worktrees.value) worktrees.value = (await derby.attempt('Could not list the worktrees', () => api.ProjectWorktrees(id))) ?? []
  if (mode === 'branch' && !branches.value) branches.value = (await derby.attempt('Could not list the branches', () => api.ProjectBranches(id))) ?? []
}
watch(() => derby.scope, load, { immediate: true })

// A path inside the project, relative to it; else the folder's name.
function short(path: string) {
  const root = derby.project(project.value)?.path ?? ''
  if (root && path.startsWith(root + '/')) return path.slice(root.length + 1)
  return path.split('/').pop() || path
}

const all = computed<Row[] | undefined>(() => {
  const id = project.value
  if (derby.scope === 'project')
    return derby.projects.map((p) => ({ key: p.id, label: p.name, detail: p.path, current: p.id === id, open: () => api.OpenProjectInZed(p.id) }))
  if (derby.scope === 'worktree')
    return worktrees.value?.map((w) => ({
      key: w.path,
      label: w.main ? pname.value : short(w.path),
      detail: w.branch || 'detached',
      current: w.path === t.value?.worktree,
      open: () => api.OpenWorktreeInZed(id, w.path),
    }))
  return branches.value?.map((b) => ({
    key: b.name,
    label: b.name,
    detail: b.path ? short(b.path) : '',
    current: b.name === t.value?.branch,
    open: () => api.OpenBranchInZed(id, b.name),
  }))
})
const rows = computed(() => {
  const q = filter.value.toLowerCase().trim()
  return (all.value ?? []).filter((r) => !q || `${r.label} ${r.detail}`.toLowerCase().includes(q))
})
watch(rows, () => (hi.value = Math.max(0, rows.value.findIndex((r) => r.current))), { immediate: true })

const placeholder = computed(() => (derby.scope === 'project' ? 'Open a project in Zed…' : derby.scope === 'worktree' ? 'Open a worktree in Zed…' : 'Open a branch in Zed…'))
const empty = computed(() => (all.value ? 'Nothing found' : 'Loading…'))

function open(r: Row) {
  derby.overlay = 'none'
  void derby.attempt('Could not open Zed', r.open)
}

const modes: ScopeMode[] = ['project', 'worktree', 'branch']
function key(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') hi.value = Math.min(rows.value.length - 1, hi.value + 1)
  else if (e.key === 'ArrowUp') hi.value = Math.max(0, hi.value - 1)
  else if (e.key === 'Tab' && t.value) {
    derby.scope = modes[(modes.indexOf(derby.scope) + (e.shiftKey ? 2 : 1)) % 3]
    filter.value = ''
  } else if (e.key === 'Enter' && rows.value[hi.value]) open(rows.value[hi.value])
  else return
  e.preventDefault()
}
</script>

<template>
  <button class="scrim" aria-label="Close" @click="derby.overlay = 'none'"></button>
  <div class="box popup" :class="{ mac: derby.os === 'darwin' }" :style="at !== undefined ? { left: at + 'px' } : undefined" role="dialog" :aria-label="placeholder">
    <label for="sq" class="sr-only">{{ placeholder }}</label>
    <input id="sq" ref="input" v-model="filter" :placeholder="placeholder" autocomplete="off" spellcheck="false" @keydown="key" />
    <div class="rows" role="listbox">
      <button v-for="(r, i) in rows" :key="r.key" class="row" role="option" :aria-selected="i === hi" :class="{ hi: i === hi }" :title="r.detail" @click="open(r)" @mousemove="hi = i">
        <span class="label" :class="{ mono: derby.scope !== 'project' }">{{ r.label }}</span>
        <span class="detail">{{ derby.scope === 'project' ? '' : r.detail }}</span>
        <Icon v-if="r.current" name="check" :size="13" class="cur" />
      </button>
      <span v-if="!rows.length" class="none">{{ empty }}</span>
    </div>
  </div>
</template>

<style scoped>
.box {
  position: absolute;
  z-index: 21;
  top: 38px;
  left: 110px;
  width: 340px;
  max-width: calc(100% - 32px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.box.mac {
  left: 184px;
}
input {
  height: 36px;
  padding: 0 14px;
  border: 0;
  border-bottom: 1px solid var(--border);
  outline: none;
  background: transparent;
  color: var(--text);
  font-size: var(--fs-13);
}
.rows {
  display: flex;
  flex-direction: column;
  padding: 6px;
  max-height: 340px;
  overflow: auto;
}
.row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  flex: none;
  height: 30px;
  padding: 0 10px;
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
.label {
  flex: none;
  max-width: 70%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.label.mono {
  font-family: var(--mono);
  font-size: var(--fs-12-5);
}
.detail {
  flex: 1;
  min-width: 0;
  font-size: var(--fs-12);
  color: var(--muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cur {
  flex: none;
  color: var(--muted);
}
.none {
  padding: 8px;
  color: var(--muted);
  font-size: var(--fs-12-5);
}
</style>
