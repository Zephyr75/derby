<script setup lang="ts">
// Settings > Appearance: one row per theme (Fleet, Catppuccin, GitHub
// Copilot), each with Dark, Light or System, then fonts and the Zed command.
import { computed } from 'vue'
import { api } from '../../lib/api'
import { debounce } from '../../lib/debounce'
import { derby } from '../../lib/state'
import { themes, type ThemeDef } from '../../lib/themes'
import type { ThemeMode, ThemeName } from '../../lib/types'
import Icon from '../ui/Icon.vue'

const ap = computed(() => derby.settings?.config.appearance)
const modes: { id: ThemeMode; name: string }[] = [
  { id: 'dark', name: 'Dark' },
  { id: 'light', name: 'Light' },
  { id: 'system', name: 'System' },
]

/** The theme's note; in use with System, the variant the OS picks now. */
function note(t: ThemeDef) {
  if (derby.themeName !== t.id || derby.themeMode !== 'system') return t.note
  return `${t.note}, now ${derby.systemLight ? t.lightName : t.darkName}`
}
const chosen = (t: ThemeDef, m: ThemeMode) => derby.themeName === t.id && derby.themeMode === m

const views = [
  { id: 'terminal', name: 'Terminal' },
  { id: 'chat', name: 'Chat' },
] as const

async function pick(theme: ThemeName, mode: ThemeMode) {
  const a = derby.settings?.config.appearance
  if (!a) return
  a.theme = theme
  a.mode = mode
  derby.edited()
  await derby.attempt('Could not save the theme', () => api.SetTheme(theme, mode))
}

const save = debounce(() => {
  const s = derby.settings
  if (!s) return
  const c = JSON.parse(JSON.stringify(s.config))
  void derby.attempt('Could not save appearance', () => api.SaveAppearance(c.appearance, c.zedCommand))
}, 600)

function setFont(kind: 'ui' | 'term', e: Event) {
  const a = derby.settings?.config.appearance
  if (!a) return
  const v = (e.target as HTMLInputElement).value.trim()
  const m = /^(.*?)\s+([\d.]+)$/.exec(v)
  const name = m ? m[1] : v
  const size = m ? Number(m[2]) : kind === 'ui' ? a.uiFontSize : a.termFontSize
  if (!name) return
  if (kind === 'ui') {
    a.uiFont = name
    a.uiFontSize = size
  } else {
    a.termFont = name
    a.termFontSize = size
  }
  derby.edited()
  save()
}
function setNotify(e: Event) {
  const on = (e.target as HTMLInputElement).checked
  if (derby.settings) derby.settings.config.notifications = on
  void derby.attempt('Could not save notifications', () => api.SetNotifications(on))
}

function setMaxTabs(e: Event) {
  const n = Math.round(Number((e.target as HTMLInputElement).value))
  if (!derby.settings || !(n >= 1)) return
  derby.settings.config.maxTabs = n
  derby.trimTabs()
  void derby.attempt('Could not save the tab limit', () => api.SetMaxTabs(n))
}

function setZed(e: Event) {
  if (!derby.settings) return
  derby.settings.config.zedCommand = (e.target as HTMLInputElement).value
  derby.edited()
  save()
}
</script>

<template>
  <div class="intro">
    <h1>Appearance</h1>
    <p>Each theme has a dark and a light variant; System follows your OS setting. Fleet is Fleet Dark and Flatwhite, the themes you use in Zed, Catppuccin is Mocha and Latte, and GitHub Copilot has the GitHub Copilot app's colours.</p>
  </div>

  <div class="themes card" role="radiogroup" aria-label="Theme">
    <div v-for="t in themes" :key="t.id" class="theme" :class="{ on: derby.themeName === t.id }">
      <span class="shot">
        <span v-for="th in [t.dark, t.light]" :key="th" class="pane" :data-theme="th" :title="th === t.dark ? t.darkName : t.lightName">
          <span class="pchrome"></span>
          <span class="pbody">
            <span class="pside">
              <span class="bar" style="width: 80%; background: var(--muted)"></span>
              <span class="bar" style="width: 55%; background: var(--running-dot)"></span>
              <span class="bar" style="width: 70%; background: var(--muted)"></span>
              <span class="bar" style="width: 50%; background: var(--waiting-dot)"></span>
            </span>
            <span class="pmain">
              <span class="bar" style="width: 60%; background: var(--faint)"></span>
              <span class="blk" style="width: 85%; background: var(--add-bg)"></span>
              <span class="blk" style="width: 70%; background: var(--del-bg)"></span>
              <span class="bar" style="width: 40%; background: var(--accent)"></span>
            </span>
          </span>
        </span>
      </span>
      <span class="label">
        <span class="name">{{ t.name }}</span>
        <span class="muted small">{{ note(t) }}</span>
      </span>
      <span class="pill modes">
        <button v-for="m in modes" :key="m.id" role="radio" :aria-checked="chosen(t, m.id)" :aria-label="`${t.name} ${m.name}`" :class="{ on: chosen(t, m.id) }" @click="pick(t.id, m.id)"><Icon :name="m.id" :size="13" />{{ m.name }}</button>
      </span>
    </div>
  </div>

  <div v-if="ap" class="rows card">
    <label class="r">
      <span class="grow">UI font</span>
      <input class="val" :value="`${ap.uiFont} ${ap.uiFontSize}`" @change="setFont('ui', $event)" />
    </label>
    <label class="r">
      <span class="grow">Terminal font</span>
      <input class="val" :value="`${ap.termFont} ${ap.termFontSize}`" @change="setFont('term', $event)" />
    </label>
    <div class="r">
      <span class="grow">Session view <span class="muted small">how the center shows a step's Claude Code session; the toolbar switches it too</span></span>
      <span class="pill views" role="radiogroup" aria-label="Session view">
        <button v-for="v in views" :key="v.id" role="radio" :aria-checked="derby.sessionView === v.id" :class="{ on: derby.sessionView === v.id }" @click="derby.setSessionView(v.id)"><Icon :name="v.id === 'chat' ? 'chat' : 'terminal'" :size="13" />{{ v.name }}</button>
      </span>
    </div>
    <label class="r">
      <span class="grow">Zed command</span>
      <input class="val" :value="derby.settings?.config.zedCommand" @change="setZed" />
    </label>
    <label class="r">
      <span class="grow">Open tabs <span class="muted small">at most; opening one more closes the oldest</span></span>
      <input class="val num" type="number" min="1" max="30" :value="derby.settings?.config.maxTabs" @change="setMaxTabs" />
    </label>
    <label class="r">
      <span class="grow">Desktop notifications <span class="muted small">when a step needs you or ends, while Derby is in the background</span></span>
      <input type="checkbox" :checked="derby.settings?.config.notifications" @change="setNotify" />
    </label>
  </div>
</template>

<style scoped>
  .themes {
    display: flex;
    flex-direction: column;
    max-width: 720px;
    overflow: hidden;
  }
  .theme {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 10px 14px;
    font-size: var(--fs-13);
  }
  .theme + .theme {
    border-top: 1px solid var(--border);
  }
  .theme.on {
    background: var(--sel);
  }
  .shot {
    flex: none;
    display: flex;
    width: 168px;
    height: 64px;
    border-radius: 6px;
    overflow: hidden;
    border: 1px solid var(--border);
  }
  .pane {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    background: var(--bg);
  }
  .pchrome {
    height: 9px;
    background: var(--chrome);
    border-bottom: 1px solid var(--border);
  }
  .pbody {
    flex: 1;
    display: flex;
  }
  .pside {
    width: 34%;
    border-right: 1px solid var(--border);
    background: var(--panel);
    padding: 5px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .pmain {
    flex: 1;
    padding: 5px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .bar {
    height: 4px;
    border-radius: 2px;
  }
  .blk {
    height: 6px;
    border-radius: 2px;
  }
  .label {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .name {
    font-weight: 600;
  }
  .modes {
    flex: none;
    width: 246px;
  }
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
  .views > button {
    flex: none; /* each as wide as its label, so the icons keep their size */
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
    text-align: right;
    width: 220px;
  }
  .val.num {
    width: 70px;
  }
  .val:focus {
    outline: none;
  }
</style>
