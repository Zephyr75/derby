<script setup lang="ts">
// Settings > Context > Prompts: every text Herd gives agents besides their
// instructions (prompts.toml): around them, and the fix steps' guidance.
import { reactive } from 'vue'
import { api } from '../../lib/api'
import { debounce } from '../../lib/debounce'
import { herd } from '../../lib/state'

const savers: Record<string, (t: string) => void> = {}
const drafts = reactive<Record<string, string>>({})
function edit(id: string, e: Event) {
  const text = (e.target as HTMLTextAreaElement).value
  drafts[id] = text
  savers[id] ??= debounce((t: string) => void herd.attempt('Could not save the prompt', () => api.SetPrompt(id, t)), 500)
  savers[id](text)
}
async function reset(id: string) {
  delete drafts[id]
  await herd.attempt('Could not reset the prompt', () => api.SetPrompt(id, ''))
}
const textOf = (id: string, text: string) => drafts[id] ?? text
// Tall enough to read the whole prompt: each line counts once per ~90
// characters it wraps over, between 8 and 30 rows.
const rows = (text: string) => Math.max(8, Math.min(30, text.split('\n').reduce((n, l) => n + Math.max(1, Math.ceil(l.length / 90)), 1)))
</script>

<template>
  <div class="prompts">
    <p class="muted small">Every text Herd gives agents besides their instructions, saved in {{ herd.settings?.dir ?? '~/.herd' }}/prompts.toml: around each agent's command (Preview full context in Settings › Agents shows the result), the files it writes for them, and the guidance of the steps the PR tab starts. Words in braces are replaced by Herd; prompts marked “Go template” use <code v-pre>{{.Field}}</code>, and fall back to their default if they no longer work. An emptied prompt goes back to its default.</p>
    <div v-for="p in herd.settings?.prompts ?? []" :key="p.id" class="editor card">
      <div class="title">
        <span class="name">{{ p.name }}</span>
        <span class="muted small">{{ p.where }}</span>
        <span class="muted small mono">{{ p.vars }}</span>
      </div>
      <label :for="'prompt-' + p.id" class="sr-only">{{ p.name }}</label>
      <textarea :id="'prompt-' + p.id" class="area mono" :rows="rows(textOf(p.id, p.text))" :value="textOf(p.id, p.text)" @input="edit(p.id, $event)"></textarea>
      <div v-if="textOf(p.id, p.text).trim() !== p.default.trim()" class="row">
        <span class="muted small">Edited</span>
        <button class="btn ghost" @click="reset(p.id)">Reset to default</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
  .prompts {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .editor {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 16px 18px;
  }
  .title {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .name {
    font-weight: 600;
  }
  .small {
    font-size: 12px;
  }
  .row {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 10px;
  }
  p {
    margin: 0;
  }
</style>
