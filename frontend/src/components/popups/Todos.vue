<script setup lang="ts">
// Every todo list: the open tasks with todos (and the selected task), then
// Others, for the todos of no task; the Archived tab has the todos of archived
// tasks. A task's name opens it.
import { computed, ref } from 'vue'
import { jockey, taskName } from '../../lib/state'
import type { Task, Todo } from '../../lib/types'
import Icon from '../ui/Icon.vue'
import TaskRef from '../ui/TaskRef.vue'
import TodoList from '../TodoList.vue'

const open = (ts: Todo[]) => ts.filter((x) => !x.done).length
const tab = ref<'active' | 'archived'>(jockey.selTask?.archived ? 'archived' : 'active')
const byName = (a: Task, b: Task) => a.projectName.localeCompare(b.projectName) || a.id.localeCompare(b.id)
const tasks = computed(() =>
  tab.value === 'archived'
    ? jockey.tasks.filter((t) => t.archived && t.todos.length).sort(byName)
    : jockey.tasks.filter((t) => !t.archived && (t.id === jockey.sel || t.todos.length)).sort(byName),
)
const openActive = computed(() => open(jockey.todos) + jockey.tasks.reduce((n, t) => n + (t.archived ? 0 : open(t.todos)), 0))
const openArchived = computed(() => jockey.tasks.reduce((n, t) => n + (t.archived ? open(t.todos) : 0), 0))

function go(t: Task) {
  jockey.overlay = 'none'
  jockey.left = t.archived ? 'archive' : 'tasks'
  jockey.pickTask(t.id)
}
</script>

<template>
  <button class="scrim dim" aria-label="Close the todo lists" @click="jockey.overlay = 'none'"></button>
  <div class="box popup pop pop-half" role="dialog" aria-label="Todo lists">
    <div class="head pop-head">
      <Icon name="todo" :size="15" />
      <span class="title">Todos</span>
      <div class="tabs" role="tablist">
        <button role="tab" :aria-selected="tab === 'active'" :class="{ on: tab === 'active' }" @click="tab = 'active'">Tasks <span class="count">{{ openActive }}</span></button>
        <button role="tab" :aria-selected="tab === 'archived'" :class="{ on: tab === 'archived' }" @click="tab = 'archived'">Archived <span class="count">{{ openArchived }}</span></button>
      </div>
      <span class="grow"></span>
      <label class="hide small"><input v-model="jockey.hideDone" type="checkbox" />Hide done</label>
      <button class="pop-x" aria-label="Close" @click="jockey.overlay = 'none'"><Icon name="close" :size="14" /></button>
    </div>
    <div class="body">
      <section v-for="t in tasks" :key="t.id">
        <button class="task" :title="`Open ${taskName(t)}`" @click="go(t)">
          <TaskRef :issue="t.issue" :task="t.id" />
          <span class="name">{{ taskName(t) }}</span>
          <span class="muted small">{{ t.projectName }}</span>
        </button>
        <TodoList :todos="t.todos" :hide-done="jockey.hideDone" @change="jockey.setTodos(t, $event)" :run="jockey.todoAgent(t)?.name ?? 'Implement'" :blocked="jockey.todoBlocked(t)" @run="jockey.runTodo(t, $event)" />
      </section>
      <p v-if="tab === 'archived' && !tasks.length" class="muted small none">No archived task has todos.</p>
      <section v-if="tab === 'active'">
        <span class="task static"><span class="name">Others</span><span class="muted small">not related to a task</span></span>
        <TodoList :todos="jockey.todos" :hide-done="jockey.hideDone" @change="jockey.setTodos(null, $event)" />
      </section>
    </div>
  </div>
</template>

<style scoped>
.title {
  font-weight: 600;
}
.grow {
  flex: 1;
}
.small {
  font-size: var(--fs-12);
}
/* text tabs in the header: the open one has an accent line on the header's border */
.tabs {
  display: flex;
  align-self: stretch;
  gap: 4px;
  margin-left: 12px;
}
.tabs > button {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  border: 0;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  background: transparent;
  color: var(--muted);
  font-size: var(--fs-13);
}
.tabs > button:hover {
  color: var(--text);
}
.tabs > button.on {
  border-bottom-color: var(--text);
  color: var(--text);
  font-weight: 500;
}
.count {
  min-width: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--hover);
  color: var(--muted);
  font-size: var(--fs-12);
  line-height: 18px;
  text-align: center;
}
.none {
  margin: 0;
  padding: 0 8px;
}
.hide {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--muted);
}
.body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 14px 12px 20px;
}
.task {
  display: flex;
  align-items: baseline;
  gap: 8px;
  width: 100%;
  padding: 6px 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--text);
  text-align: left;
}
button.task:hover {
  background: var(--hover);
}
.task .name {
  font-weight: 600;
  font-size: var(--fs-13);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}
.task .small {
  flex: none;
}
</style>
