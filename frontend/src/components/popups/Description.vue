<script setup lang="ts">
// The description of a task that has no GitHub issue, opened from the
// Description button under the task's title in the right panel.
import { derby } from '../../lib/state'
import Icon from '../ui/Icon.vue'
import TaskRef from '../ui/TaskRef.vue'

const close = () => (derby.overlay = 'none')
</script>

<template>
  <button class="scrim dim" aria-label="Close the description" @click="close"></button>
  <div v-if="derby.selTask" class="box popup" role="dialog" aria-label="Task description">
    <div class="head">
      <Icon name="description" :size="15" />
      <span class="title">Description</span>
      <TaskRef :issue="derby.selTask.issue" :task="derby.selTask.id" />
      <span class="name muted" :title="derby.selTask.title">{{ derby.selTask.title }}</span>
      <button class="x" aria-label="Close" @click="close"><Icon name="close" :size="14" /></button>
    </div>
    <p class="body">{{ derby.selTask.description }}</p>
  </div>
</template>

<style scoped>
.box {
  position: absolute;
  z-index: 21;
  top: 12%;
  left: 50%;
  transform: translateX(-50%);
  width: 640px;
  max-width: calc(100% - 48px);
  max-height: 70%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.head {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 42px;
  flex: none;
  padding: 0 10px 0 14px;
  border-bottom: 1px solid var(--border);
}
.title {
  flex: none;
  font-weight: 600;
}
.name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--fs-12-5);
}
.x {
  flex: none;
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--muted);
}
.x:hover {
  background: var(--hover);
  color: var(--text);
}
.body {
  margin: 0;
  padding: 14px 16px 18px;
  overflow: auto;
  white-space: pre-wrap;
  line-height: 1.6;
  user-select: text;
}
</style>
