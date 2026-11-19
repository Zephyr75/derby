<script setup lang="ts">
// A diff, unified (hunks with old and new line numbers) or split (the old
// file on the left, the new on the right, as Zed's split view). With
// commentable, a + button shows on the hovered line: a click comments on that
// line, a drag (or a shift-click on another line's +) on a range, marked by
// a bar down the gutter. The comments show under their last line.
import { computed, nextTick, onBeforeUnmount, ref } from 'vue'
import type { Note } from '../lib/types'

const props = defineProps<{ diff: string; notes?: Note[]; commentable?: boolean; split?: boolean }>()
const emit = defineEmits<{ add: [n: Omit<Note, 'id' | 'path'>]; update: [id: string, text: string]; remove: [id: string] }>()

interface Row {
  kind: 'hunk' | 'add' | 'del' | 'ctx' | 'note'
  old: number | null
  new: number | null
  text: string
}

const rows = computed(() => {
  const out: Row[] = []
  let o = 0
  let n = 0
  let inHunk = false
  for (const line of props.diff.split('\n')) {
    const m = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@(.*)$/.exec(line)
    if (m) {
      o = Number(m[1])
      n = Number(m[2])
      inHunk = true
      out.push({ kind: 'hunk', old: null, new: null, text: m[3].trim() })
      continue
    }
    if (!inHunk) continue // diff --git, index, ---, +++ headers
    if (line.startsWith('+')) out.push({ kind: 'add', old: null, new: n++, text: line.slice(1) })
    else if (line.startsWith('-')) out.push({ kind: 'del', old: o++, new: null, text: line.slice(1) })
    else if (line.startsWith('\\')) out.push({ kind: 'note', old: null, new: null, text: line.slice(2) })
    else if (line.startsWith('diff --git')) inHunk = false
    else out.push({ kind: 'ctx', old: o++, new: n++, text: line.slice(1) })
  }
  return out
})

// What the table shows: one row per diff line, or, split, a removed line
// beside the added line it became (-1: nothing on that side). idx: the
// diff lines of a table row, the comments under it follow.
interface Unit {
  hunk: boolean
  left: number
  right: number
  idx: number[]
}
const units = computed<Unit[]>(() => {
  const rs = rows.value
  if (!props.split) return rs.map((r, i) => ({ hunk: r.kind === 'hunk', left: i, right: i, idx: [i] }))
  const out: Unit[] = []
  let i = 0
  while (i < rs.length) {
    const r = rs[i]
    if (r.kind !== 'del' && r.kind !== 'add') {
      out.push({ hunk: r.kind === 'hunk', left: i, right: r.kind === 'note' ? -1 : i, idx: [i] })
      i++
      continue
    }
    const dels: number[] = []
    const adds: number[] = []
    while (i < rs.length && rs[i].kind === 'del') dels.push(i++)
    while (i < rs.length && rs[i].kind === 'add') adds.push(i++)
    for (let k = 0; k < Math.max(dels.length, adds.length); k++) {
      const l = dels[k] ?? -1
      const rt = adds[k] ?? -1
      out.push({ hunk: false, left: l, right: rt, idx: [l, rt].filter((x) => x >= 0) })
    }
  }
  return out
})
const cols = computed(() => (props.split ? 6 : 4))
const sides = (u: Unit) => [
  { side: 'l', i: u.left },
  { side: 'r', i: u.right },
]
// Split: the line under the mouse shows its + (a row holds two lines).
const hov = ref(-1)
function enter(i: number) {
  hov.value = i
  over(i)
}
const sign = (r: Row) => (r.kind === 'add' ? '+' : r.kind === 'del' ? '−' : ' ')

// Selection: the rows a new comment is on.
const anchor = ref(-1)
const end = ref(-1)
const editing = ref('') // a note's id, or 'new'
const draft = ref('')
const box = ref<HTMLTextAreaElement[]>()
const lo = computed(() => Math.min(anchor.value, end.value))
const hi = computed(() => Math.max(anchor.value, end.value))
const selected = (i: number) => anchor.value >= 0 && i >= lo.value && i <= hi.value
const lineRow = (r: Row) => r.kind === 'add' || r.kind === 'del' || r.kind === 'ctx'
// A line's selection classes: in it, its first, its last.
const marks = (i: number) => (selected(i) ? { sel: true, 'sel-top': i === lo.value, 'sel-end': i === hi.value } : {})

function focus() {
  void nextTick(() => box.value?.[0]?.focus())
}
// The + button: press starts a selection, dragging over lines extends it,
// releasing opens the editor under its last line.
const dragging = ref(false)
function press(i: number, e: MouseEvent) {
  if (!props.commentable || !lineRow(rows.value[i]) || e.button !== 0) return
  e.preventDefault() // no text selection while dragging
  if (e.shiftKey && anchor.value >= 0 && editing.value === 'new') end.value = i
  else (anchor.value = i), (end.value = i)
  editing.value = ''
  dragging.value = true
  window.addEventListener('mouseup', release, { once: true })
}
function over(i: number) {
  if (dragging.value && lineRow(rows.value[i])) end.value = i
}
function release() {
  if (!dragging.value) return
  dragging.value = false
  editing.value = 'new'
  focus()
}
onBeforeUnmount(() => window.removeEventListener('mouseup', release))
function cancel() {
  anchor.value = end.value = -1
  editing.value = ''
  draft.value = ''
}
function save() {
  const text = draft.value.trim()
  if (editing.value === 'new' && text) {
    const sel = rows.value.slice(lo.value, hi.value + 1).filter(lineRow)
    // New line numbers unless every line is a removed one.
    const side = sel.some((r) => r.new !== null) ? 'new' : 'old'
    const nums = sel.map((r) => (side === 'new' ? r.new : r.old)).filter((x): x is number => x !== null)
    const code = sel.map((r) => (r.kind === 'add' ? '+' : r.kind === 'del' ? '-' : ' ') + r.text).join('\n')
    emit('add', { side, from: Math.min(...nums), to: Math.max(...nums), code, text })
  } else if (editing.value && editing.value !== 'new') {
    if (text) emit('update', editing.value, text)
    else emit('remove', editing.value)
  }
  cancel()
}
function edit(n: Note) {
  anchor.value = end.value = -1
  editing.value = n.id
  draft.value = n.text
  focus()
}
function key(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    cancel()
  }
}

// Each note under the row of its last line; the ones whose lines left the diff at the end.
const placed = computed(() => {
  const at = new Map<number, Note[]>()
  const lost: Note[] = []
  for (const n of props.notes ?? []) {
    let idx = -1
    rows.value.forEach((r, i) => {
      if ((n.side === 'new' ? r.new : r.old) === n.to && (n.side === 'new' || r.kind === 'del')) idx = i
    })
    if (idx < 0) lost.push(n)
    else at.set(idx, [...(at.get(idx) ?? []), n])
  }
  return { at, lost }
})
const span = (n: Note) => (n.from === n.to ? `line ${n.to}` : `lines ${n.from}–${n.to}`) + (n.side === 'old' ? ' (removed)' : '')
</script>

<template>
  <div class="diff" :class="{ commentable, split }">
    <table @mouseleave="hov = -1">
      <colgroup v-if="split">
        <col class="c-ln" /><col class="c-gut" /><col /><col class="c-ln" /><col class="c-gut" /><col />
      </colgroup>
      <tbody>
        <template v-for="(u, j) in units" :key="j">
          <tr v-if="u.hunk" class="hunk">
            <td class="ln" :colspan="split ? 1 : 3">⋯</td>
            <td class="code" :colspan="split ? 5 : 1">{{ rows[u.left].text }}</td>
          </tr>
          <!-- unified: old and new line numbers, then the line -->
          <tr v-else-if="!split" :class="[rows[u.left].kind, marks(u.left)]" @mouseenter="over(u.left)">
            <td class="ln">{{ rows[u.left].old ?? '' }}</td>
            <td class="ln">{{ rows[u.left].new ?? '' }}</td>
            <td class="gut">
              <button v-if="commentable && lineRow(rows[u.left])" class="add-note" aria-label="Comment on this line" title="Comment on this line (drag, or shift-click another line's +, for several)" @mousedown="press(u.left, $event)">+</button>
            </td>
            <td class="code"><span class="sign">{{ sign(rows[u.left]) }}</span>{{ rows[u.left].text }}</td>
          </tr>
          <!-- split: the old file on the left, the new on the right -->
          <tr v-else class="srow">
            <template v-for="c in sides(u)" :key="c.side">
              <template v-if="c.i >= 0">
                <td class="ln" :class="[c.side, rows[c.i].kind, marks(c.i)]" @mouseenter="enter(c.i)">{{ (c.side === 'l' ? rows[c.i].old : rows[c.i].new) ?? '' }}</td>
                <td class="gut" :class="[rows[c.i].kind, marks(c.i), { hov: hov === c.i }]" @mouseenter="enter(c.i)">
                  <button v-if="commentable" class="add-note" aria-label="Comment on this line" title="Comment on this line (drag, or shift-click another line's +, for several)" @mousedown="press(c.i, $event)">+</button>
                </td>
                <td class="code" :class="[rows[c.i].kind, marks(c.i)]" @mouseenter="enter(c.i)"><span class="sign">{{ sign(rows[c.i]) }}</span>{{ rows[c.i].text }}</td>
              </template>
              <td v-else class="blank" :class="c.side" colspan="3"></td>
            </template>
          </tr>
          <template v-for="i in u.idx" :key="'n' + i">
            <tr v-for="n in placed.at.get(i) ?? []" :key="n.id" class="note-row">
              <td :colspan="cols">
                <div v-if="editing === n.id" class="editor">
                  <textarea ref="box" v-model="draft" rows="3" class="area" data-own-escape @keydown="key"></textarea>
                  <div class="bar"><span class="hint">Empty removes it</span><button class="b" @click="cancel">Cancel</button><button class="b on" @click="save">Save</button></div>
                </div>
                <div v-else class="note" title="Click to edit" @click="edit(n)">
                  <span class="where">{{ span(n) }}</span>
                  <span class="text">{{ n.text }}</span>
                  <button class="x" aria-label="Remove the comment" title="Remove the comment" @click.stop="emit('remove', n.id)">×</button>
                </div>
              </td>
            </tr>
          </template>
          <tr v-if="editing === 'new' && u.idx.includes(hi)" class="note-row">
            <td :colspan="cols">
              <div class="editor">
                <textarea ref="box" v-model="draft" rows="3" class="area" data-own-escape placeholder="What should change here?" @keydown="key"></textarea>
                <div class="bar"><span class="hint">Drag the + for several lines</span><button class="b" @click="cancel">Cancel</button><button class="b on" :disabled="!draft.trim()" @click="save">Comment</button></div>
              </div>
            </td>
          </tr>
        </template>
        <tr v-for="n in placed.lost" :key="n.id" class="note-row">
          <td :colspan="cols">
            <div class="note" title="Its lines are no longer in this diff" @click="edit(n)">
              <span class="where">{{ span(n) }}, no longer in the diff</span>
              <span class="text">{{ n.text }}</span>
              <button class="x" aria-label="Remove the comment" title="Remove the comment" @click.stop="emit('remove', n.id)">×</button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.diff {
  width: max-content; /* the scrolling parent scrolls long lines sideways */
  min-width: 100%;
  font-family: var(--mono);
  font-size: var(--fs-12);
  line-height: 1.55;
  tab-size: 4;
}
table {
  border-collapse: collapse;
  min-width: 100%;
}
td {
  padding: 0;
  vertical-align: top;
}
.ln {
  width: 1%;
  padding: 0 6px;
  text-align: right;
  color: var(--faint);
  user-select: none;
  white-space: nowrap;
}
.code {
  padding: 0 12px 0 4px;
  white-space: pre;
}
.sign {
  display: inline-block;
  width: 14px;
  color: var(--muted);
  user-select: none;
}
.add {
  background: var(--add-bg);
}
.add .sign {
  color: var(--add);
}
.del {
  background: var(--del-bg);
}
.del .sign {
  color: var(--del);
}
.hunk td {
  padding-top: 4px;
  padding-bottom: 4px;
  background: var(--surface);
  color: var(--muted);
}
.note .code {
  color: var(--muted);
  font-style: italic;
}
.gut {
  width: 18px;
  min-width: 18px;
}
.add-note {
  visibility: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  margin-top: 1px;
  padding: 0;
  border: 0;
  border-radius: 3px;
  background: var(--accent);
  color: #fff;
  font-size: var(--fs-13);
  font-weight: 600;
  line-height: 1;
  cursor: pointer;
}
tr:hover .add-note,
.split .gut.hov .add-note {
  visibility: visible;
}
.split tr:hover .gut:not(.hov) .add-note {
  visibility: hidden;
}
.split td.sel,
.sel td {
  box-shadow: inset 0 0 0 9999px color-mix(in srgb, var(--accent) 10%, transparent);
}
/* the selected lines: one continuous bar down the gutter */
.gut {
  position: relative;
}
.add-note {
  position: relative;
  z-index: 1;
}
.sel .gut::before,
.gut.sel::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  width: 16px; /* as wide as the + button */
  background: var(--accent);
}
.sel-top .gut::before,
.gut.sel-top::before {
  top: 1px;
  border-radius: 3px 3px 0 0;
}
.sel-end .gut::before,
.gut.sel-end::before {
  bottom: 1px;
  border-radius: 0 0 3px 3px;
}
.sel-top.sel-end .gut::before,
.gut.sel-top.sel-end::before {
  border-radius: 3px;
}
.note-row td {
  padding: 4px 12px 6px 60px;
  font-family: var(--ui-font);
  white-space: normal;
}
.note {
  display: flex;
  align-items: baseline;
  gap: 8px;
  max-width: 720px;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-left: 3px solid var(--accent);
  border-radius: 4px;
  background: var(--surface);
  font-size: var(--fs-12-5);
  cursor: text;
}
.where {
  flex: none;
  color: var(--muted);
  font-size: var(--fs-11-5);
}
.text {
  flex: 1;
  white-space: pre-wrap;
}
.x {
  border: 0;
  background: transparent;
  color: var(--muted);
  font-size: var(--fs-14);
  line-height: 1;
}
.x:hover {
  color: var(--text);
}
.editor {
  max-width: 720px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.editor .area {
  width: 100%;
  font-family: var(--ui-font);
  font-size: var(--fs-12-5);
}
.bar {
  display: flex;
  align-items: center;
  gap: 6px;
}
.hint {
  flex: 1;
  color: var(--muted);
  font-size: var(--fs-11-5);
}
.b {
  height: 24px;
  padding: 0 10px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--surface);
  color: var(--text);
  font-size: var(--fs-12);
}
.b.on {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
}
.b:disabled {
  opacity: 0.5;
}
/* split: two halves of the width, long lines wrapped, a rule between them */
.diff.split {
  width: 100%;
}
.split table {
  width: 100%;
  table-layout: fixed;
}
.split .c-ln {
  width: 44px;
}
.split .c-gut {
  width: 18px;
}
.split .code {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.split td.ln.r,
.split td.blank.r {
  border-left: 1px solid var(--border);
}
/* Zed's diff strip: a bar at the left edge of each changed line, 0.275 of
   the line height wide, in the full version-control colour */
.diff tr.add > td:first-child,
.diff tr.del > td:first-child,
.split td.ln.add,
.split td.ln.del {
  position: relative;
}
.diff tr.add > td:first-child::after,
.diff tr.del > td:first-child::after,
.split td.ln.add::after,
.split td.ln.del::after {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 5px;
  background: var(--vc-add);
}
.diff tr.del > td:first-child::after,
.split td.ln.del::after {
  background: var(--vc-del);
}
</style>
