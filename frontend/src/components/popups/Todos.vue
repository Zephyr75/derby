<script setup lang="ts">
// Every todo list: the open tasks with todos (and the selected task), the
// archived ones that still have open todos, then Others, for the todos of
// no task. A task's name opens it.
import { computed } from 'vue'
import { derby, taskName } from '../../lib/state'
import type { Task, Todo } from '../../lib/types'
import Icon from '../ui/Icon.vue'
import TaskRef from '../ui/TaskRef.vue'
import TodoList from '../TodoList.vue'

const open = (ts: Todo[]) => ts.filter((x) => !x.done).length
const tasks = computed(() => {
  const shown = derby.tasks.filter((t) => t.id === derby.sel || (t.archived ? open(t.todos) > 0 : t.todos.length > 0))
  return shown.sort((a, b) => Number(a.archived) - Number(b.archived) || a.projectName.localeCompare(b.projectName) || a.id.localeCompare(b.id))
})
const total = computed(() => open(derby.todos) + derby.tasks.reduce((n, t) => n + open(t.todos), 0))

function go(t: Task) {
  derby.overlay = 'none'
  derby.left = t.archived ? 'archive' : 'tasks'
  derby.pickTask(t.id)
}
</script>

<template>
  <button class="scrim dim" aria-label="Close the todo lists" @click="derby.overlay = 'none'"></button>
  <div class="box popup pop pop-half" role="dialog" aria-label="Todo lists">
    <div class="head">
      <Icon name="todo" :size="15" />
      <span class="title">Todos</span>
      <span class="muted small">{{ total }} open</span>
      <span class="grow"></span>
      <label class="hide small"><input v-model="derby.hideDone" type="checkbox" />Hide done</label>
      <button class="x" aria-label="Close" @click="derby.overlay = 'none'"><Icon name="close" :size="14" /></button>
    </div>
    <div class="body">
      <section v-for="t in tasks" :key="t.id">
        <button class="task" :title="`Open ${taskName(t)}`" @click="go(t)">
          <TaskRef :issue="t.issue" :task="t.id" />
          <span class="name">{{ taskName(t) }}</span>
          <span class="muted small">{{ t.projectName }}{{ t.archived ? ' · archived' : '' }}</span>
        </button>
        <TodoList :todos="t.todos" :hide-done="derby.hideDone" @change="derby.setTodos(t, $event)" :run="derby.todoAgent(t)?.name ?? 'Implement'" :blocked="derby.todoBlocked(t)" @run="derby.runTodo(t, $event)" />
      </section>
      <section>
        <span class="task static"><span class="name">Others</span><span class="muted small">not related to a task</span></span>
        <TodoList :todos="derby.todos" :hide-done="derby.hideDone" @change="derby.setTodos(null, $event)" />
      </section>
    </div>
  </div>
</template>

<style scoped>
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
  font-weight: 600;
}
.grow {
  flex: 1;
}
.small {
  font-size: var(--fs-12);
}
.hide {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--muted);
}
.x {
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
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 12px 10px 16px;
}
.task {
  display: flex;
  align-items: baseline;
  gap: 8px;
  width: 100%;
  padding: 4px 6px;
  border: 0;
  border-radius: 5px;
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
