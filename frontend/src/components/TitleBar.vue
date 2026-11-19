<script setup lang="ts">
// herd's top bar is the window's titlebar: drag it to move the window,
// double-click to maximise. On Linux and Windows the window is frameless and
// the controls are drawn here; macOS keeps its traffic lights.
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { herd } from '../lib/state'
import { label } from '../lib/keymap'
import { isFullscreen, isMaximised, minimise, quit, toggleMaximise } from '../lib/window'
import Icon from './ui/Icon.vue'

const t = computed(() => herd.selTask)
const scopeOpen = (m: string) => herd.overlay === 'scope' && herd.scope === m
const controls = computed(() => herd.os !== 'darwin')
const openTodos = computed(() => herd.todos.filter((x) => !x.done).length + herd.tasks.reduce((n, x) => n + x.todos.filter((y) => !y.done).length, 0))
const todosKey = computed(() => label(herd.settings?.keymap.todos?.[0]))

const maximised = ref(false)
const fullscreen = ref(false)
const sync = async () => {
  maximised.value = await isMaximised()
  fullscreen.value = await isFullscreen()
}
onMounted(() => {
  void sync()
  window.addEventListener('resize', sync)
})
onBeforeUnmount(() => window.removeEventListener('resize', sync))

// Double-click on the bar itself (not on a button) maximises, like a native titlebar.
function dblclick(e: MouseEvent) {
  if (!controls.value) return
  if ((e.target as HTMLElement).closest('button, input')) return
  toggleMaximise()
}
</script>

<template>
  <header :class="{ mac: herd.os === 'darwin', full: fullscreen }" @dblclick="dblclick">
    <button class="menu" :class="{ on: herd.view === 'settings' }" aria-label="Settings" title="Settings" @click="herd.command('settings')">
      <Icon name="menu" :size="15" />
    </button>
    <template v-if="herd.view === 'board' && t">
      <button class="crumb project" :class="{ on: scopeOpen('project') }" @click="herd.openOverlay('scope', 'project')">
        {{ t.projectName }}<Icon name="chevron" :size="13" />
      </button>
      <span class="sep" aria-hidden="true">/</span>
      <button class="crumb" :class="{ on: scopeOpen('worktree') }" @click="herd.openOverlay('scope', 'worktree')">
        <Icon name="worktree" /><span class="ct">{{ t.worktreeRel }}</span>
      </button>
      <span class="sep" aria-hidden="true">/</span>
      <button class="crumb" :class="{ on: scopeOpen('branch') }" @click="herd.openOverlay('scope', 'branch')">
        <Icon name="branch" /><span class="ct">{{ t.branch }}</span>
      </button>
    </template>
    <span v-else-if="herd.view === 'settings'" class="muted label">Settings</span>
    <span class="grow"></span>
    <button class="todos" :class="{ on: herd.overlay === 'todos' }" :title="`Todo lists${todosKey ? ' (' + todosKey + ')' : ''}`" @click="herd.command('todos')">
      <Icon name="todo" :size="15" /><span v-if="openTodos" class="tcount">{{ openTodos }}</span>
    </button>
    <button class="search" @click="herd.openOverlay('palette')">
      <Icon name="search" /><span class="grow stext">Search tasks</span>
    </button>
    <button class="new" @click="herd.openOverlay('new')">
      <Icon name="plus" :size="14" />New task
    </button>
    <div v-if="controls" class="win" aria-label="Window">
      <!-- Zed's Linux window controls: 16px icons with an 8px glyph -->
      <button class="wc" aria-label="Minimise" title="Minimise" @click="minimise()">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M4 8h8" /></svg>
      </button>
      <button class="wc" :aria-label="maximised ? 'Restore' : 'Maximise'" :title="maximised ? 'Restore' : 'Maximise'" @click="toggleMaximise()">
        <svg v-if="maximised" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M3.5 6.5h6v6h-6z M7.5 6V3.5h5v5H10" /></svg>
        <svg v-else width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M4.5 4.5h7v7h-7z" /></svg>
      </button>
      <button class="wc close" aria-label="Close" title="Close" @click="quit()">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="square"><path d="M11.5 4.5l-7 7M4.5 4.5l7 7" /></svg>
      </button>
    </div>
  </header>
</template>

<style scoped>
header {
  --wails-draggable: drag;
  display: flex;
  align-items: center;
  gap: 2px;
  flex: none;
  height: 34px;
  padding: 0 6px;
  background: var(--chrome);
  border-bottom: 1px solid var(--border);
  user-select: none;
}
header.mac:not(.full) {
  padding-left: 78px; /* the traffic lights */
}
header:not(.mac) {
  padding-right: 4px;
}
button {
  --wails-draggable: no-drag;
}
.crumb {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 24px;
  padding: 0 8px;
  border-radius: 6px;
  border: 0;
  background: transparent;
  color: var(--muted);
  font-size: 13px;
  white-space: nowrap;
  min-width: 0; /* long worktree and branch names shrink first */
  overflow: hidden;
  text-overflow: ellipsis;
}
.crumb > :deep(svg) {
  flex: none;
}
.ct {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}
.crumb.project {
  flex: none;
  color: var(--text);
  font-weight: 500;
}
.crumb:hover,
.crumb.on {
  background: var(--hover);
  color: var(--text);
}
.sep {
  /* Zed: a small muted slash at a quarter of its opacity */
  color: var(--muted);
  opacity: 0.25;
  font-size: 12px;
}
.stext {
  overflow: hidden;
  text-overflow: ellipsis;
}
.search > :deep(svg) {
  flex: none;
}
.label {
  padding: 0 6px;
}
.grow {
  flex: 1;
}
.todos {
  flex: none;
  display: flex;
  align-items: center;
  gap: 5px;
  height: 24px;
  padding: 0 8px;
  margin-right: 6px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--muted);
}
.todos:hover,
.todos.on {
  background: var(--hover);
  color: var(--text);
}
.tcount {
  font-family: var(--mono);
  font-size: 11px;
}
.search {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 24px;
  flex: 0 1 130px;
  min-width: 34px; /* the icon alone when space is short */
  overflow: hidden;
  white-space: nowrap;
  padding: 0 10px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--bg);
  color: var(--muted);
  text-align: left;
}
.new {
  flex: none;
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 6px;
  height: 24px;
  padding: 0 13px 0 10px; /* text ends flush, so the right side needs more to look even */
  margin-left: 6px;
  border-radius: 6px;
  border: 1px solid var(--border);
  background: var(--surface);
  color: var(--text);
}
.menu {
  width: 28px;
  height: 24px;
  margin-right: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  border: 0;
  background: transparent;
  color: var(--muted);
}
.menu.on,
.menu:hover {
  color: var(--text);
  background: var(--hover);
}
.win {
  flex: none;
  display: flex;
  align-items: center;
  gap: 12px; /* Zed: gap-3 px-3 */
  padding: 0 8px 0 12px;
  height: 20px;
}
.wc {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--text);
}
.wc:hover {
  background: var(--hover);
  color: var(--text);
}
.wc.close:hover {
  background: var(--approval-bg);
  color: var(--approval);
}
</style>
