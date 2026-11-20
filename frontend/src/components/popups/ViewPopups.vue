<script setup lang="ts">
// The artifact and changes pop-ups: one alone is big, both open side by
// side at half size. A click around them closes both.
import { computed } from 'vue'
import { derby } from '../../lib/state'
import ArtifactPopup from './ArtifactPopup.vue'
import DiffPopup from './DiffPopup.vue'
import DiagramPopup from './DiagramPopup.vue'
// A diagram opened full size covers them; closing it goes back to them.
const size = computed(() => (derby.artifactOf && derby.diffOf ? 'pop-half' : 'pop-big'))
</script>

<template>
  <button class="scrim dim" aria-label="Close" @click="derby.closeViews()"></button>
  <div v-if="derby.diagramOf" class="pair">
    <DiagramPopup class="pop-big" />
  </div>
  <div v-else class="pair">
    <ArtifactPopup v-if="derby.artifactOf" :class="size" />
    <DiffPopup v-if="derby.diffOf" :class="size" />
  </div>
</template>

<style scoped>
  .pair {
    position: absolute;
    z-index: 21;
    top: 48px;
    bottom: 40px;
    left: 24px;
    right: 24px;
    display: flex;
    justify-content: center;
    gap: 16px;
    pointer-events: none; /* the gaps let clicks reach the scrim */
  }
  .pair > * {
    pointer-events: auto;
  }
</style>
