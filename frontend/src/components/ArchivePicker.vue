<script setup lang="ts">
// The Archive's picker: the Archive lists one project's archived tasks at a
// time, the sister families (several projects' work, so a pick of their own),
// or everything. A Zed picker, as the top bar's lists: a ghost button, then a
// filter over the picks.
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { derby, SISTERS } from '../lib/state'
import Icon from './ui/Icon.vue'

interface Row {
  id: string
  label: string
  count: number
}

const open = ref(false)
const filter = ref('')
const hi = ref(0)
const root = ref<HTMLElement>()
const input = ref<HTMLInputElement>()

// Counts are archived tasks: a project's own, and the families' together.
const all = computed<Row[]>(() => {
  const archived = derby.tasks.filter((t) => t.archived)
  const families = derby.sisterGroups
  const single = archived.filter((t) => !families.has(t.id))
  const projects = derby.groups
    .map((g) => ({ id: g.id, label: g.name, count: single.filter((t) => t.project === g.id).length, gone: g.gone }))
    .filter((r) => !r.gone || r.count)
  return [{ id: '', label: 'All projects', count: archived.length }, { id: SISTERS, label: 'Sisters', count: archived.length - single.length }, ...projects]
})
const label = computed(() => all.value.find((r) => r.id === derby.archiveScope)?.label ?? 'All projects')
const rows = computed(() => {
  const q = filter.value.toLowerCase().trim()
  return all.value.filter((r) => !q || r.label.toLowerCase().includes(q))
})
const current = () => Math.max(0, rows.value.findIndex((r) => r.id === derby.archiveScope))
watch(rows, () => (hi.value = current()))

async function toggle() {
  open.value = !open.value
  if (!open.value) return
  filter.value = ''
  hi.value = current()
  await nextTick()
  input.value?.focus()
}

function pick(r: Row) {
  open.value = false
  derby.pickArchiveProject(r.id)
}

function key(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') hi.value = Math.min(rows.value.length - 1, hi.value + 1)
  else if (e.key === 'ArrowUp') hi.value = Math.max(0, hi.value - 1)
  else if (e.key === 'Enter' && rows.value[hi.value]) pick(rows.value[hi.value])
  else if (e.key === 'Escape') open.value = false
  else return
  e.preventDefault()
}

// Closes on a click anywhere else.
const outside = (e: PointerEvent) => root.value && !root.value.contains(e.target as Node) && (open.value = false)
watch(open, (on) => (on ? window.addEventListener('pointerdown', outside, true) : window.removeEventListener('pointerdown', outside, true)))
onBeforeUnmount(() => window.removeEventListener('pointerdown', outside, true))
</script>

<template>
  <div ref="root" class="picker">
    <button type="button" class="trigger" :class="{ on: open }" title="Pick what to list" aria-haspopup="listbox" :aria-expanded="open" @click="toggle">
      <Icon :name="derby.archiveScope === SISTERS ? 'sisters' : 'folder-open'" :size="14" class="fi" />
      <span class="name">{{ label }}</span>
      <Icon name="chevron" :size="12" class="chev" />
    </button>
    <div v-if="open" class="box popup" role="dialog" aria-label="Projects">
      <label for="aq" class="sr-only">Filter projects</label>
      <input id="aq" ref="input" v-model="filter" placeholder="Filter projects…" autocomplete="off" spellcheck="false" @keydown="key" />
      <div class="rows" role="listbox">
        <button v-for="(r, i) in rows" :key="r.id" type="button" class="row" role="option" :aria-selected="r.id === derby.archiveScope" :class="{ hi: i === hi }" @click="pick(r)" @mousemove="hi = i">
          <span class="label">{{ r.label }}</span>
          <span class="count">{{ r.count }}</span>
          <span class="cur"><Icon v-if="r.id === derby.archiveScope" name="check" :size="13" /></span>
        </button>
        <span v-if="!rows.length" class="none">Nothing found</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.picker {
  position: relative;
  flex: none;
  display: flex;
  padding: 8px 8px 0;
}
/* Zed's ghost button with a chevron, as the top bar's project */
.trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
  height: 28px;
  padding: 0 8px 0 12px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  text-align: left;
}
.trigger:hover,
.trigger.on {
  background: var(--hover);
}
.fi,
.chev {
  flex: none;
}
.name {
  min-width: 0;
  color: var(--text);
  font-weight: 600;
  font-size: var(--fs-12-5);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.box {
  position: absolute;
  z-index: 30;
  top: calc(100% + 4px);
  left: 8px;
  right: 8px;
  max-width: 320px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
input {
  height: 32px;
  padding: 0 12px;
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
  padding: 4px;
  max-height: 320px;
  overflow: auto;
}
.row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  flex: none;
  height: 28px;
  padding: 0 8px;
  border-radius: 5px;
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
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.count {
  flex: none;
  font-size: var(--fs-12);
  color: var(--faint);
}
.cur {
  flex: none;
  display: flex;
  width: 13px;
  color: var(--muted);
}
.none {
  padding: 8px;
  color: var(--muted);
  font-size: var(--fs-12-5);
}
</style>
