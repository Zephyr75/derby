<script setup lang="ts">
import { computed } from 'vue'
import { jockey, taskName } from '../lib/state'
import Icon from './ui/Icon.vue'
import TaskCard from './TaskCard.vue'
import ArchivePicker from './ArchivePicker.vue'
import NextStep from './NextStep.vue'
import Splitter from './ui/Splitter.vue'
import type { Task } from '../lib/types'

const counts = computed(() => ({
  active: jockey.tasks.filter((t) => !t.archived).length,
  archived: jockey.tasks.filter((t) => t.archived).length,
}))
const groups = computed(() => jockey.groups.map((p) => ({ p, tasks: jockey.panelTasks(p.id) })).filter((g) => !g.p.gone || g.tasks.length))

// A sister family's category, named after its big sister (even when only
// one of its tasks is in this list).
function family(tasks: Task[]) {
  // Every task of the family, archived ones included.
  const all = (jockey.sisterGroups.get(tasks[0].id) ?? []).map((id) => jockey.task(id)).filter((t): t is Task => !!t)
  const big = all.find((t) => !t.sister) ?? tasks[0]
  return { key: jockey.familyKey(tasks[0]), tasks, name: taskName(big), tip: 'Sister tasks in ' + all.map((t) => t.projectName).join(', ') }
}

// Tasks: sister families first, each in its own category; single tasks
// under their project.
const families = computed(() => jockey.panelFamilies().map(family))

// Archive: the pick's tasks (a project's, the sister families or all), newest
// first, each family folded into one row.
const archive = computed(() => jockey.archiveEntries().map((e) => (e.family ? { ...family(e.tasks), family: true } : { key: e.key, tasks: e.tasks, family: false, name: '', tip: '' })))

function addProject() {
  jockey.view = 'settings'
  jockey.settingsTab = 'projects'
}
</script>

<template>
  <aside :style="{ width: jockey.leftWidth + 'px' }">
    <Splitter side="left" />
    <div class="tabs" role="tablist">
      <button role="tab" :aria-selected="jockey.left === 'tasks'" :class="{ on: jockey.left === 'tasks' }" @click="jockey.left = 'tasks'">Tasks <span class="count">{{ counts.active }}</span></button>
      <button role="tab" :aria-selected="jockey.left === 'archive'" :class="{ on: jockey.left === 'archive' }" @click="jockey.left = 'archive'">Archive <span class="count">{{ counts.archived }}</span></button>
    </div>

    <ArchivePicker v-if="jockey.left === 'archive' && jockey.groups.length" />

    <div class="list">
      <template v-if="jockey.left === 'archive'">
        <template v-for="e in archive" :key="e.key">
          <template v-if="e.family">
            <button class="group fam" :aria-expanded="!jockey.isCollapsed(e.key)" :title="e.tip" @click="jockey.toggleCollapsed(e.key)">
              <Icon name="sisters" />
              <span class="name">{{ e.name }}</span>
              <span class="n">{{ e.tasks.length }}</span>
            </button>
            <div v-if="!jockey.isCollapsed(e.key)" class="nest">
              <div v-for="(t, i) in e.tasks" :key="t.id" class="cardw"><TaskCard :task="t" family :big="i === 0 && !t.sister" /></div>
            </div>
          </template>
          <div v-else class="cardw"><TaskCard :task="e.tasks[0]" :project="!jockey.archiveScope" /></div>
        </template>
        <span v-if="!archive.length && jockey.groups.length" class="nothing">Nothing archived</span>
      </template>
      <template v-else>
        <template v-for="f in families" :key="f.key">
          <button class="group" :aria-expanded="!jockey.isCollapsed(f.key)" :title="f.tip" @click="jockey.toggleCollapsed(f.key)">
            <Icon name="sisters" />
            <span class="name">{{ f.name }}</span>
          </button>
          <div v-if="!jockey.isCollapsed(f.key)" class="nest">
            <div v-for="(t, i) in f.tasks" :key="t.id" class="cardw"><TaskCard :task="t" family :big="i === 0 && !t.sister" /></div>
          </div>
        </template>
        <template v-for="g in groups" :key="g.p.id">
          <button class="group" :aria-expanded="!jockey.isCollapsed(g.p.id)" @click="jockey.toggleCollapsed(g.p.id)">
            <Icon :name="jockey.isCollapsed(g.p.id) ? 'folder' : 'folder-open'" />
            <span class="name">{{ g.p.name }}</span>
            <span v-if="g.p.gone" class="gone" title="Project removed">removed</span>
          </button>
          <span v-if="!jockey.isCollapsed(g.p.id) && !g.tasks.length" class="empty">No open tasks</span>
          <div v-else-if="!jockey.isCollapsed(g.p.id)" class="nest">
            <div v-for="t in g.tasks" :key="t.id" class="cardw"><TaskCard :task="t" /></div>
          </div>
        </template>
      </template>
      <div v-if="jockey.left === 'archive' ? !jockey.groups.length : !groups.length" class="none">
        <p>No projects yet.</p>
        <button class="btn" @click="addProject">Add a project</button>
      </div>
    </div>

    <NextStep />
  </aside>
</template>

<style scoped>
aside {
  position: relative;
  flex: 0 1 auto; /* narrow windows: shrink before the terminal is cut */
  min-width: 200px;
  display: flex;
  flex-direction: column;
  background: var(--panel);
  border-right: 1px solid var(--border);
  min-height: 0;
}
/* Zed's panel tabs (as Git's Changes and History): two equal tabs on the panel's
   colour (only the centre's tab bar takes the darker chrome); the open one has
   no line under it, so it runs into the list */
.tabs {
  display: flex;
  flex: none;
  height: 32px; /* lines up with the centre's tab bar */
}
.tabs > button {
  flex: 1 1 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 0;
  padding: 0 12px;
  border: 0;
  border-bottom: 1px solid var(--border);
  background: var(--panel);
  color: var(--muted);
  font-size: var(--fs-13);
  white-space: nowrap;
}
.tabs > button + button {
  border-left: 1px solid var(--border);
}
.tabs > button:hover {
  color: var(--text);
}
.tabs > button.on {
  border-bottom-color: transparent;
  background: var(--panel);
  color: var(--text);
  font-weight: 500;
}
.count {
  color: var(--faint);
  font-weight: 400;
}
.tabs .on .count {
  color: var(--muted);
}
.list {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 8px 14px;
}
/* a project (its open-or-closed folder says whether it is collapsed) or a
   sister family */
.group {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  flex: none;
  padding: 0 12px;
  margin-top: 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  font-size: var(--fs-12);
  color: var(--muted);
  text-align: left;
}
.list > .group:first-child {
  margin-top: 0;
}
.group:hover {
  background: var(--hover);
}
.group > :deep(svg) {
  flex: none;
}
.group .name {
  min-width: 0;
  color: var(--text);
  font-weight: 600;
  font-size: var(--fs-12-5);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.empty {
  padding: 4px 12px 8px 36px;
  font-size: var(--fs-12);
  color: var(--muted);
}
/* the Archive: a family folded into a row like a task's, its tasks indented when unfolded */
.group.fam {
  height: 36px;
  margin-top: 0;
}
.fam .name {
  font-weight: 500;
  font-size: var(--fs-13);
}
.fam .n {
  margin-left: auto;
  flex: none;
  color: var(--faint);
}
.nest {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-left: 19px;
  padding-left: 4px;
}
.nothing {
  padding: 4px 12px;
  font-size: var(--fs-12);
  color: var(--muted);
}
.cardw {
  position: relative;
}
.none {
  padding: 16px 10px;
  color: var(--muted);
}
.none p {
  margin: 0 0 10px;
}
.gone {
  font-size: var(--fs-11);
  padding: 0 5px;
  border-radius: 4px;
  border: 1px solid var(--border);
  color: var(--muted);
}
</style>
