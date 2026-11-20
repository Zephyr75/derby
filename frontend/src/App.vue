<script setup lang="ts">
import { watchEffect } from 'vue'
import { derby } from './lib/state'
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
import Description from './components/popups/Description.vue'

watchEffect(() => {
  document.documentElement.dataset.theme = derby.theme
})
watchEffect(() => {
  const ap = derby.settings?.config.appearance
  if (!ap) return
  const root = document.documentElement.style
  root.setProperty('--ui-font', `'${ap.uiFont}', 'Geist', system-ui, sans-serif`)
  root.setProperty('--mono', `'${ap.termFont}', 'Geist Mono', ui-monospace, monospace`)
  root.setProperty('--ui-size', `${ap.uiFontSize}px`)
})
</script>

<template>
  <div v-if="derby.failed" class="fatal">
    <strong>Derby could not load</strong>
    <span class="mono">{{ derby.failed }}</span>
  </div>
  <div v-else-if="derby.ready" class="app">
    <TitleBar />
    <main>
      <template v-if="derby.view === 'board'">
        <LeftPanel v-if="!derby.leftHidden" />
        <Center />
        <RightPanel v-if="!derby.rightHidden" />
      </template>
      <Settings v-else />
    </main>
    <StatusBar />

    <Palette v-if="derby.overlay === 'palette'" />
    <ScopePicker v-if="derby.overlay === 'scope'" />
    <NewTask v-if="derby.overlay === 'new'" />
    <Activity v-if="derby.overlay === 'activity'" />
    <Todos v-if="derby.overlay === 'todos'" />
    <Description v-if="derby.overlay === 'description' && derby.selTask" />
    <ViewPopups v-if="derby.overlay === 'views' && (derby.diffOf || derby.artifactOf || derby.diagramOf)" />
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
