<script setup lang="ts">
import { watchEffect } from 'vue'
import { jockey } from './lib/state'
import TitleBar from './components/TitleBar.vue'
import StatusBar from './components/StatusBar.vue'
import LeftPanel from './components/LeftPanel.vue'
import Center from './components/Center.vue'
import RightPanel from './components/RightPanel.vue'
import Settings from './components/settings/Settings.vue'
import Palette from './components/popups/Palette.vue'
import ScopePicker from './components/popups/ScopePicker.vue'
import NewTask from './components/popups/NewTask.vue'
import Activity from './components/popups/Activity.vue'
import Todos from './components/popups/Todos.vue'
import ViewPopups from './components/popups/ViewPopups.vue'

watchEffect(() => {
  document.documentElement.dataset.theme = jockey.theme
})
watchEffect(() => {
  const ap = jockey.settings?.config.appearance
  if (!ap) return
  const root = document.documentElement.style
  root.setProperty('--ui-font', `'${ap.uiFont}', 'Geist', system-ui, sans-serif`)
  root.setProperty('--mono', `'${ap.termFont}', 'Geist Mono', ui-monospace, monospace`)
  root.setProperty('--ui-size', `${ap.uiFontSize}px`)
})
</script>

<template>
  <div v-if="jockey.failed" class="fatal">
    <strong>Jockey could not load</strong>
    <span class="mono">{{ jockey.failed }}</span>
  </div>
  <div v-else-if="jockey.ready" class="app">
    <TitleBar />
    <main>
      <template v-if="jockey.view === 'board'">
        <LeftPanel v-if="!jockey.leftHidden" />
        <Center />
        <RightPanel v-if="!jockey.rightHidden" />
      </template>
      <Settings v-else />
    </main>
    <StatusBar />

    <Palette v-if="jockey.overlay === 'palette'" />
    <ScopePicker v-if="jockey.overlay === 'scope'" />
    <NewTask v-if="jockey.overlay === 'new'" />
    <Activity v-if="jockey.overlay === 'activity'" />
    <Todos v-if="jockey.overlay === 'todos'" />
    <ViewPopups v-if="jockey.overlay === 'views' && (jockey.diffOf || jockey.artifactOf || jockey.diagramOf)" />
  </div>
</template>

<style scoped>
.app {
  position: relative;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg);
  color: var(--text);
}
main {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: stretch;
}
.fatal {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--muted);
}
.fatal strong {
  color: var(--text);
}
</style>
