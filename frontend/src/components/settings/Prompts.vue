<script setup lang="ts">
// Settings > Context > Prompts: every text Jockey gives agents besides their
// instructions (prompts.toml): around them, and the fix steps' guidance. With
// skills, the internal skills Jockey ships (each a prompt with a skill name,
// /jockey:<skill> in every session) instead.
import { computed, reactive } from 'vue'
import { api } from '../../lib/api'
import { debounce } from '../../lib/debounce'
import { jockey } from '../../lib/state'

const props = defineProps<{ skills?: boolean }>()
const locked = computed(() => !!jockey.settings?.config.useDefaults)
const shown = computed(() => (jockey.settings?.prompts ?? []).filter((p) => !!p.skill === !!props.skills))
const savers: Record<string, (t: string) => void> = {}
const drafts = reactive<Record<string, string>>({})
function edit(id: string, e: Event) {
  const text = (e.target as HTMLTextAreaElement).value
  drafts[id] = text
  savers[id] ??= debounce((t: string) => void jockey.attempt('Could not save the prompt', () => api.SetPrompt(id, t)), 500)
  savers[id](text)
}
async function reset(id: string) {
  delete drafts[id]
  await jockey.attempt('Could not reset the prompt', () => api.SetPrompt(id, ''))
}
const textOf = (id: string, text: string) => drafts[id] ?? text
// Tall enough to read the whole prompt: each line counts once per ~90
// characters it wraps over, between 8 and 30 rows.
const rows = (text: string) => Math.max(8, Math.min(30, text.split('\n').reduce((n, l) => n + Math.max(1, Math.ceil(l.length / 90)), 1)))
</script>

<template>
  <div class="prompts">
    <p v-if="skills" class="muted small">Skills are Claude Code skills Jockey ships with its agents, saved in {{ jockey.settings?.dir ?? '~/.jockey' }}/prompts.toml: an agent's instructions name the one to use. Each one's description, the first line below its name, is what tells Claude when to use it. An emptied skill goes back to its default.</p>
    <p v-else class="muted small">Every text Jockey gives agents besides their instructions, saved in {{ jockey.settings?.dir ?? '~/.jockey' }}/prompts.toml: around each agent's command (Preview full context in Settings › Agents shows the result), the files it writes for them, and the guidance of the steps the PR tab starts. Words in braces are replaced by Jockey. Prompts marked “Go template” use <code v-pre>{{.Field}}</code>, and fall back to their default if they no longer work. An emptied prompt goes back to its default.</p>
    <p v-if="locked" class="muted small locked">Jockey's default {{ skills ? 'skills are' : 'prompts are' }} in use (Settings &gt; General): they are shown here but cannot be edited.</p>
    <div v-for="p in shown" :key="p.id" class="editor card">
      <div class="title">
        <span class="name">{{ p.name }}<span v-if="p.skill" class="mono muted small"> /jockey:{{ p.skill }}</span></span>
        <span class="muted small">{{ p.where }}</span>
        <span class="muted small mono">{{ p.vars }}</span>
      </div>
      <label :for="'prompt-' + p.id" class="sr-only">{{ p.name }}</label>
      <textarea :id="'prompt-' + p.id" class="area mono" :rows="rows(textOf(p.id, p.text))" :value="textOf(p.id, p.text)" :disabled="locked" @input="edit(p.id, $event)"></textarea>
      <div v-if="textOf(p.id, p.text).trim() !== p.default.trim()" class="row">
        <span class="muted small">Edited</span>
        <button class="btn ghost" :disabled="locked" @click="reset(p.id)">Reset to default</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
  .locked {
    color: var(--waiting);
  }
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
    font-size: var(--fs-12);
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
