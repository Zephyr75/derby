<script setup lang="ts">
// Settings > General: what applies to Jockey as a whole, in one list of rows:
// where agents run (Zed, Claude), how tasks are named and found, what the
// window does, and whether agents and prompts are the repository's defaults.
import { computed } from 'vue'
import { api } from '../../lib/api'
import { debounce } from '../../lib/debounce'
import { jockey } from '../../lib/state'

const cfg = computed(() => jockey.settings?.config)
const issuesRepo = computed(() => cfg.value?.issuesRepo ?? '')

const saveZed = debounce(() => {
  const c = cfg.value
  if (c) void jockey.attempt('Could not save the Zed command', () => api.SaveAppearance(JSON.parse(JSON.stringify(c.appearance)), c.zedCommand))
}, 600)
function setZed(e: Event) {
  if (!cfg.value) return
  cfg.value.zedCommand = (e.target as HTMLInputElement).value
  jockey.edited()
  saveZed()
}
function setPrefix(e: Event) {
  const v = (e.target as HTMLInputElement).value.trim()
  if (!v || !cfg.value) return
  cfg.value.branchPrefix = v
  void jockey.attempt('Could not save the branch prefix', () => api.SetBranchPrefix(v))
}
function setIssuesRepo(e: Event) {
  const v = (e.target as HTMLInputElement).value.trim()
  if (!cfg.value) return
  cfg.value.issuesRepo = v
  void jockey.attempt('Could not save the issues repository', () => api.SetIssuesRepo(v))
}
function setMaxTabs(e: Event) {
  const n = Math.round(Number((e.target as HTMLInputElement).value))
  if (!cfg.value || !(n >= 1)) return
  cfg.value.maxTabs = n
  jockey.trimTabs()
  void jockey.attempt('Could not save the tab limit', () => api.SetMaxTabs(n))
}
function setFoldWords(e: Event) {
  const words = (e.target as HTMLInputElement).value.split(',').map((w) => w.trim()).filter(Boolean)
  if (!cfg.value) return
  cfg.value.foldWords = words
  void jockey.attempt('Could not save the fold words', () => api.SetFoldWords(words))
}
function setNotify(e: Event) {
  const on = (e.target as HTMLInputElement).checked
  if (cfg.value) cfg.value.notifications = on
  void jockey.attempt('Could not save notifications', () => api.SetNotifications(on))
}
function setDefaults(e: Event) {
  const on = (e.target as HTMLInputElement).checked
  if (cfg.value) cfg.value.useDefaults = on
  jockey.edited()
  void jockey.attempt('Could not change the defaults', () => api.SetUseDefaults(on))
}
</script>

<template>
  <div class="intro">
    <h1>General</h1>
    <p>Settings that apply to the whole of Jockey. Themes, fonts and the session view are in Appearance and the repositories in Projects.</p>
  </div>

  <div v-if="cfg" class="rows card">
    <label class="r">
      <span class="grow">Use Jockey's default agents and prompts <span class="muted small">agents, prompts and skills come from the repository's defaults/, not from agents.toml and prompts.toml in {{ jockey.settings?.dir ?? '~/.jockey' }} (which are left as they are). Settings &gt; Agents and Context then only show them. For working on Jockey itself</span></span>
      <input type="checkbox" :checked="cfg.useDefaults" @change="setDefaults" />
    </label>
    <label class="r">
      <span class="grow">Zed command <span class="muted small">opens worktrees and commits</span></span>
      <input class="val" :value="cfg.zedCommand" @change="setZed" />
    </label>
    <label class="r">
      <span class="grow">Branch prefix <span class="muted small">every new task's branch starts with it</span></span>
      <input class="val" :value="cfg.branchPrefix" placeholder="ar-" spellcheck="false" @change="setPrefix" />
    </label>
    <label class="r">
      <span class="grow">Issues repository <span class="muted small">where the issues of projects set to use it live, e.g. one repo for all the team's issues</span></span>
      <input class="val" :value="issuesRepo" placeholder="owner/name" spellcheck="false" @change="setIssuesRepo" />
    </label>
    <label class="r">
      <span class="grow">Open tabs <span class="muted small">at most, one per task. Opening one more closes the oldest</span></span>
      <input class="val num" type="number" min="1" max="30" :value="cfg.maxTabs" @change="setMaxTabs" />
    </label>
    <label class="r">
      <span class="grow">Fold words <span class="muted small">comma separated. A changed file whose path has one starts folded</span></span>
      <input class="val wide" :value="cfg.foldWords.join(', ')" @change="setFoldWords" />
    </label>
    <label class="r">
      <span class="grow">Desktop notifications <span class="muted small">when a step needs you or ends, while Jockey is in the background</span></span>
      <input type="checkbox" :checked="cfg.notifications" @change="setNotify" />
    </label>
  </div>
</template>

<style scoped>
  .small {
    font-size: var(--fs-12);
  }
  .rows {
    display: flex;
    flex-direction: column;
    max-width: 720px;
  }
  .r {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
  }
  .r + .r {
    border-top: 1px solid var(--border);
  }
  .grow {
    flex: 1;
  }
  .val {
    font-family: var(--mono);
    font-size: var(--fs-12-5);
    padding: 3px 8px;
    border-radius: 4px;
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text);
    width: 220px;
  }
  .val.wide {
    width: 260px;
  }
  .val.num {
    width: 70px;
    text-align: right;
  }
  .val:focus {
    outline: none;
  }
</style>
