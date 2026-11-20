<script setup lang="ts">
import { computed } from 'vue'
import { derby, SISTERS } from '../lib/state'
import Icon from './ui/Icon.vue'
import TaskCard from './TaskCard.vue'
import NextStep from './NextStep.vue'
import Splitter from './ui/Splitter.vue'
import type { Task } from '../lib/types'

const counts = computed(() => ({
  active: derby.tasks.filter((t) => !t.archived).length,
  archived: derby.tasks.filter((t) => t.archived).length,
}))
const groups = computed(() => derby.groups.map((p) => ({ p, tasks: derby.panelTasks(p.id) })).filter((g) => !g.p.gone || g.tasks.length))

// Sister families first, under Sisters, each in a box (even when only one of
// its tasks is in this list); single tasks under their project.
const families = computed(() => derby.panelFamilies())
// Every project of the family, archived tasks included.
const familyTitle = (f: Task[]) => 'The same work in ' + (derby.sisterGroups.get(f[0].id) ?? []).map((id) => derby.task(id)?.projectName ?? id).join(', ')

function addProject() {
  derby.view = 'settings'
  derby.settingsTab = 'projects'
}
</script>

<template>
  <aside :style="{ width: derby.leftWidth + 'px' }">
    <Splitter side="left" />
    <div class="tabs" role="tablist">
      <button role="tab" :aria-selected="derby.left === 'tasks'" :class="{ on: derby.left === 'tasks' }" @click="derby.left = 'tasks'">Tasks <span class="count">{{ counts.active }}</span></button>
      <button role="tab" :aria-selected="derby.left === 'archive'" :class="{ on: derby.left === 'archive' }" @click="derby.left = 'archive'">Archive <span class="count">{{ counts.archived }}</span></button>
    </div>

    <div class="list">
      <template v-if="families.length">
        <button class="group" :aria-expanded="!derby.isCollapsed(SISTERS)" @click="derby.toggleCollapsed(SISTERS)">
          <Icon name="sisters" />
          <span class="name">Sisters</span>
        </button>
        <template v-if="!derby.isCollapsed(SISTERS)">
          <div v-for="f in families" :key="f[0].id" class="family" :title="familyTitle(f)">
            <TaskCard v-for="(t, i) in f" :key="t.id" :task="t" family :big="i === 0 && !t.sister" />
          </div>
        </template>
      </template>
      <template v-for="g in groups" :key="g.p.id">
        <button class="group" :aria-expanded="!derby.isCollapsed(g.p.id)" @click="derby.toggleCollapsed(g.p.id)">
          <Icon :name="derby.isCollapsed(g.p.id) ? 'folder' : 'folder-open'" />
          <span class="name">{{ g.p.name }}</span>
          <span v-if="g.p.gone" class="gone" title="This project was removed from Settings > Projects">removed</span>
        </button>
        <template v-if="!derby.isCollapsed(g.p.id)">
          <span v-if="!g.tasks.length" class="empty">{{ derby.left === 'archive' ? 'Nothing archived' : 'No open tasks' }}</span>
          <div v-for="t in g.tasks" :key="t.id" class="cardw"><TaskCard :task="t" /></div>
        </template>
      </template>
      <div v-if="!groups.length" class="none">
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
/* Zed's panel tabs (as Git's Changes and History): two equal tabs; the open one
   takes the panel's colour with no line under it, so it runs into the list */
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
  background: var(--chrome);
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
/* a project (or Sisters): its open-or-closed folder says whether it is collapsed */
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
.group .name {
  color: var(--text);
  font-weight: 600;
  font-size: var(--fs-12-5);
}
.empty {
  padding: 4px 12px 8px 36px;
  font-size: var(--fs-12);
  color: var(--muted);
}
.cardw {
  position: relative;
}
/* a sister family: its tasks in one box, a little wider than a task */
.family {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 2px 0 6px;
  padding: 4px;
  border: 1px solid var(--border);
  border-radius: 9px;
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
