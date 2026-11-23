<script setup lang="ts">
// The messages the user sent in this thread, as a Zed picker (the Archive's
// project picker): a ghost button in the thread's corner opens a filterable
// list, a pick goes back to that message. The message being read has a dot.
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { derby } from '../lib/state'
import { label } from '../lib/keymap'
import Icon from './ui/Icon.vue'

export interface Mark {
  turn: number
  text: string
  command: string // a /command's name, shown before its arguments
  at: string // RFC 3339, empty when unknown
}
const props = defineProps<{ thread: HTMLElement | undefined; marks: Mark[] }>()
const emit = defineEmits<{ go: [turn: number] }>()

const filter = ref('')
const hi = ref(0)
const root = ref<HTMLElement>()
const input = ref<HTMLInputElement>()
const list = ref<HTMLElement>()
const shown = computed(() => props.marks.length > 1)
const open = computed(() => shown.value && derby.messages)

const firstLine = (s: string) => s.split('\n').find((l) => l.trim()) ?? ''
const rows = computed(() => {
  const q = filter.value.toLowerCase().trim()
  return props.marks.filter((m) => !q || `${m.command} ${m.text}`.toLowerCase().includes(q))
})

// The turn being read: the last message above the top of the thread, the
// last one at the bottom.
const cur = ref(0)
let raf = 0
function measure() {
  raf = 0
  const el = props.thread
  if (!el) return
  if (el.scrollHeight - el.scrollTop - el.clientHeight < 4) {
    cur.value = props.marks[props.marks.length - 1]?.turn ?? 0
    return
  }
  const top = el.getBoundingClientRect().top + 24
  let turn = 0
  el.querySelectorAll<HTMLElement>('[data-turn]').forEach((m) => {
    if (m.getBoundingClientRect().top <= top) turn = Number(m.dataset.turn)
  })
  cur.value = turn || props.marks[0]?.turn || 0
}
const queue = () => (raf ||= requestAnimationFrame(measure))
watch(
  () => props.thread,
  (el, old) => {
    old?.removeEventListener('scroll', queue)
    el?.addEventListener('scroll', queue, { passive: true })
    queue()
  },
  { immediate: true },
)
watch(() => props.marks.length, () => void nextTick(queue))

const current = () => Math.max(0, rows.value.findIndex((m) => m.turn === cur.value))
watch(rows, () => (hi.value = current()))
watch(open, async (on) => {
  if (on) {
    queue()
    filter.value = ''
    await nextTick()
    measure()
    hi.value = current()
    await nextTick()
    input.value?.focus()
    scrollTo()
    window.addEventListener('pointerdown', outside, true)
  } else window.removeEventListener('pointerdown', outside, true)
})

const close = () => (derby.messages = false)
function pick(m: Mark) {
  close()
  emit('go', m.turn)
}
function scrollTo() {
  list.value?.querySelector<HTMLElement>('.hi')?.scrollIntoView({ block: 'nearest' })
}
function key(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') hi.value = Math.min(rows.value.length - 1, hi.value + 1)
  else if (e.key === 'ArrowUp') hi.value = Math.max(0, hi.value - 1)
  else if (e.key === 'Enter' && rows.value[hi.value]) pick(rows.value[hi.value])
  else if (e.key === 'Escape') close()
  else return
  e.preventDefault()
  void nextTick(scrollTo)
}
// Closes on a click anywhere else.
const outside = (e: PointerEvent) => root.value && !root.value.contains(e.target as Node) && close()
onBeforeUnmount(() => {
  window.removeEventListener('pointerdown', outside, true)
  props.thread?.removeEventListener('scroll', queue)
  cancelAnimationFrame(raf)
  derby.messages = false
})

// "Today", "Yesterday", else "Nov 18".
function when(at: string) {
  const d = new Date(at)
  if (!at || isNaN(+d)) return ''
  const days = Math.round((new Date().setHours(0, 0, 0, 0) - new Date(+d).setHours(0, 0, 0, 0)) / 864e5)
  return days === 0 ? 'Today' : days === 1 ? 'Yesterday' : d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}
const tip = computed(() => `Your messages (${label(derby.settings?.keymap.messages?.[0])})`)
</script>

<template>
  <div v-if="shown" ref="root" class="picker">
    <button type="button" class="trigger" :class="{ on: open }" :title="tip" aria-haspopup="listbox" :aria-expanded="open" @click="derby.messages = !derby.messages">
      <span>Your messages</span>
      <b>{{ marks.length }}</b>
      <Icon name="chevron" :size="12" />
    </button>
    <div v-if="open" class="box popup" role="dialog" aria-label="Your messages">
      <label for="mq" class="sr-only">Filter messages</label>
      <input id="mq" ref="input" v-model="filter" placeholder="Filter messages…" autocomplete="off" spellcheck="false" @keydown="key" />
      <div ref="list" class="rows" role="listbox">
        <button v-for="(m, i) in rows" :key="m.turn" type="button" class="row" role="option" :aria-selected="m.turn === cur" :class="{ hi: i === hi }" @click="pick(m)" @mousemove="hi = i">
          <span class="dot"><i v-if="m.turn === cur"></i></span>
          <span class="label"><code v-if="m.command" class="cmd">{{ m.command }}</code> {{ firstLine(m.text) }}</span>
          <span class="when">{{ when(m.at) }}</span>
        </button>
        <span v-if="!rows.length" class="none">Nothing found</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.picker {
  position: absolute;
  z-index: 3;
  top: 8px;
  right: 16px; /* clear of the thread's scrollbar */
  left: 16px;
  display: flex;
  justify-content: flex-end;
  pointer-events: none;
}
.picker > * {
  pointer-events: auto;
}
/* Zed's ghost button with a chevron, as the Archive's picker */
.trigger {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 26px;
  padding: 0 8px 0 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
  font-size: var(--fs-12);
}
.trigger b {
  color: var(--text);
  font-weight: 600;
}
.trigger:hover,
.trigger.on {
  background: var(--hover);
}
.box {
  position: absolute;
  z-index: 30;
  top: calc(100% + 4px);
  right: 0;
  width: min(340px, 100%);
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
.dot {
  flex: none;
  display: flex;
  width: 8px;
}
.dot i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent);
}
.label {
  flex: 1;
  min-width: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cmd {
  font-family: var(--mono);
  font-size: var(--fs-12);
  color: var(--muted);
}
.when {
  flex: none;
  font-size: var(--fs-12);
  color: var(--faint);
}
.none {
  padding: 8px;
  color: var(--muted);
  font-size: var(--fs-12-5);
}
</style>
