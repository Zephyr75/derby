<script setup lang="ts">
// Preview full context: a popup with everything a step of the agent gets, as
// being edited, for an example task of the chosen project.
import { computed, nextTick, ref, watch } from 'vue'
import { api } from '../../lib/api'
import { debounce } from '../../lib/debounce'
import { jockey } from '../../lib/state'
import type { Agent, ContextPart } from '../../lib/types'
import Dropdown from '../ui/Dropdown.vue'
import Icon from '../ui/Icon.vue'

const props = defineProps<{ agent: Agent }>()

const parts = ref<ContextPart[] | null>(null) // null: closed
const project = ref('')
const box = ref<HTMLElement | null>(null)
const projects = computed(() => jockey.settings?.config.projects ?? [])
const projectItems = computed(() => projects.value.map((p, i) => ({ value: i ? p.id : '', label: p.name })))
const load = debounce(async () => {
  if (parts.value === null) return
  parts.value = (await jockey.attempt('Could not build the context', () => api.AgentContext(JSON.parse(JSON.stringify(props.agent)), project.value))) ?? []
}, 300)
async function open() {
  parts.value = []
  load()
  await nextTick()
  box.value?.focus()
}
const close = () => (parts.value = null)
function key(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    close()
  }
}
watch(() => [JSON.stringify(props.agent), JSON.stringify(jockey.settings?.prompts), JSON.stringify(jockey.settings?.config), project.value], load)
</script>

<template>
  <div class="ctx">
    <button class="btn" @click="open"><Icon name="eye" :size="13" />Preview full context</button>
    <span class="hint">Everything a step of {{ agent.name }} gets, in the order Claude reads it.</span>
  </div>
  <Teleport v-if="parts !== null" to="body">
    <button class="scrim dim" aria-label="Close the context preview" @click="close"></button>
    <div ref="box" class="box popup pop pop-big" role="dialog" :aria-label="`Full context of ${agent.name}`" tabindex="-1" data-own-escape @keydown="key">
      <div class="head pop-head">
        <span class="title">Full context of {{ agent.name }}</span>
        <span class="grow"></span>
        <div v-if="projects.length > 1" class="proj">
          <span class="muted">Example task of</span>
          <Dropdown label="Project of the example task" :items="projectItems" :value="project" @pick="project = $event" />
        </div>
        <button class="pop-x" aria-label="Close" @click="close"><Icon name="close" :size="14" /></button>
      </div>
      <div class="body">
        <span v-if="!parts.length" class="muted">Building the context…</span>
        <details v-for="p in parts" :key="p.title" class="part" open>
          <summary>
            <span class="title mono">{{ p.title }}</span>
            <span class="where">{{ p.where }}</span>
          </summary>
          <pre class="text mono">{{ p.text }}</pre>
        </details>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.ctx {
  display: flex;
  align-items: center;
  gap: 10px;
}
.ctx .btn {
  gap: 6px;
  height: 28px;
}
.hint {
  font-size: var(--fs-12);
  color: var(--muted);
}
.box {
  outline: none;
}
.head .title {
  font-weight: 600;
}
.grow {
  flex: 1;
}
.proj {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--fs-12);
}
.body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
}
.part summary {
  display: flex;
  align-items: baseline;
  gap: 8px;
  cursor: pointer;
  padding: 4px 0;
}
.part .title {
  font-size: var(--fs-12);
  font-weight: 600;
}
.where {
  font-size: var(--fs-11-5);
  color: var(--muted);
}
.text {
  margin: 0;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--surface);
  font-size: var(--fs-11-5);
  line-height: 1.5;
  white-space: pre-wrap;
}
</style>
