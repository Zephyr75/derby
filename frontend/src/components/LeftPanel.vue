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
    <div class="head">
      <div class="pill" role="tablist">
        <button role="tab" :aria-selected="derby.left === 'tasks'" :class="{ on: derby.left === 'tasks' }" @click="derby.left = 'tasks'">Tasks <span class="count">{{ counts.active }}</span></button>
        <button role="tab" :aria-selected="derby.left === 'archive'" :class="{ on: derby.left === 'archive' }" @click="derby.left = 'archive'">Archive <span class="count">{{ counts.archived }}</span></button>
      </div>
    </div>

    <div class="list">
      <template v-if="families.length">
        <button class="group" :aria-expanded="!derby.isCollapsed(SISTERS)" @click="derby.toggleCollapsed(SISTERS)">
          <Icon :name="derby.isCollapsed(SISTERS) ? 'chevron-right' : 'chevron'" :size="14" />
          <Icon name="sisters" />
          <span class="name">Sisters</span>
          <span>{{ families.length }}</span>
        </button>
        <template v-if="!derby.isCollapsed(SISTERS)">
          <div v-for="f in families" :key="f[0].id" class="family" :title="familyTitle(f)">
            <TaskCard v-for="(t, i) in f" :key="t.id" :task="t" family :big="i === 0 && !t.sister" />
          </div>
        </template>
      </template>
      <template v-for="g in groups" :key="g.p.id">
        <button class="group" :aria-expanded="!derby.isCollapsed(g.p.id)" @click="derby.toggleCollapsed(g.p.id)">
          <Icon :name="derby.isCollapsed(g.p.id) ? 'chevron-right' : 'chevron'" :size="14" />
          <Icon name="folder" />
          <span class="name">{{ g.p.name }}</span>
          <span v-if="g.p.gone" class="gone" title="This project was removed from Settings > Projects">removed</span>
          <span>{{ g.tasks.length }}</span>
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
.head {
  display: flex;
  align-items: center;
  flex: none;
  height: 32px;
  padding: 0 8px;
  border-bottom: 1px solid var(--border);
}
.head .pill {
  flex: 1;
}
.count {
  color: var(--faint);
}
.pill .on .count {
  color: inherit;
  opacity: 0.75;
}
.list {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  padding: 4px 6px 8px;
}
.group {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  flex: none;
  padding: 0 6px;
  margin-top: 4px;
  border: 0;
  background: transparent;
  font-size: var(--fs-12);
  color: var(--muted);
  text-align: left;
}
.group .name {
  color: var(--text);
  font-weight: 600;
  font-size: var(--fs-12-5);
}
.empty {
  padding: 2px 12px 6px 30px;
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
  gap: 1px;
  margin: 2px 0 6px;
  padding: 3px;
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
