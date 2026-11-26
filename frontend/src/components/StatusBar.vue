<script setup lang="ts">
import { computed } from 'vue'
import { jockey } from '../lib/state'
import { label } from '../lib/keymap'
import Icon from './ui/Icon.vue'

const counts = computed(() => {
  const c = { running: 0, waiting: 0, approval: 0 }
  const projects = new Set<string>()
  for (const t of jockey.tasks) {
    if (t.archived) continue
    const k = t.status.kind === 'failed' ? 'waiting' : t.status.kind // a failure waits for you
    if (k !== 'running' && k !== 'waiting' && k !== 'approval') continue
    c[k]++
    projects.add(t.project) // the projects of every task the line counts
  }
  return { ...c, projects: projects.size }
})
const modelLabel = (m: string) => (m ? m[0].toUpperCase() + m.slice(1) : '')
const right = computed(() => {
  const t = jockey.selTask
  if (!t) return ''
  const parts = [t.branch, t.projectName]
  if (jockey.selStep) parts.unshift(modelLabel(jockey.selStep.model))
  return parts.filter(Boolean).join(' · ')
})
const tip = (what: string, hidden: boolean, action: string) => `${hidden ? 'Show' : 'Hide'} ${what} (${label(jockey.settings?.keymap[action]?.[0])})`
</script>

<template>
  <footer>
    <button class="dock" :class="{ off: jockey.leftHidden }" :title="tip('tasks', jockey.leftHidden, 'toggleLeft')" @click="jockey.togglePanel('left')">
      <Icon :name="jockey.leftHidden ? 'panel-left-off' : 'panel-left'" :size="19" />
    </button>
    <button v-if="counts.running + counts.waiting + counts.approval" class="activity" :class="{ on: jockey.overlay === 'activity' }" title="Show running tasks" @click="jockey.openOverlay(jockey.overlay === 'activity' ? 'none' : 'activity')">
      <span v-if="counts.running" class="item"><span class="dot s-running"></span>{{ counts.running }} running</span>
      <span v-if="counts.waiting" class="item"><span class="dot s-waiting"></span>{{ counts.waiting }} waiting for input</span>
      <span v-if="counts.approval" class="item"><span class="dot s-approval"></span>{{ counts.approval }} needs approval</span>
      <span v-if="counts.projects">across {{ counts.projects }} project{{ counts.projects > 1 ? 's' : '' }}</span>
    </button>
    <span class="grow"></span>
    <span>{{ right }}</span>
    <button class="dock" :class="{ off: jockey.rightHidden }" :title="tip('steps', jockey.rightHidden, 'toggleRight')" @click="jockey.togglePanel('right')">
      <Icon :name="jockey.rightHidden ? 'panel-right-off' : 'panel-right'" :size="19" />
    </button>
  </footer>
</template>

<style scoped>
footer {
  display: flex;
  align-items: center;
  gap: 16px;
  flex: none;
  height: 31px; /* as Zed: 4px padding around 22px buttons, plus the border */
  padding: 0 12px;
  background: var(--chrome);
  border-top: 1px solid var(--border);
  font-size: var(--fs-13-5);
  color: var(--muted);
  white-space: nowrap;
  overflow: hidden;
}
.activity {
  display: flex;
  align-items: center;
  gap: 16px;
  height: 22px;
  margin-left: -6px;
  padding: 0 6px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: inherit;
  font-size: var(--fs-13-5);
}
.activity:hover,
.activity.on {
  background: var(--hover);
  color: var(--text);
}
.item {
  display: flex;
  align-items: center;
  gap: 7px;
}
.dot {
  width: 8px;
  height: 8px;
  background: var(--dot) !important;
}
.grow {
  flex: 1;
}
.dock {
  flex: none;
  width: 28px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 -6px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--text);
}
.dock.off {
  color: var(--muted);
}
.dock:hover {
  background: var(--hover);
}
</style>
