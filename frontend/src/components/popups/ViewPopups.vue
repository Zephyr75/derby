<script setup lang="ts">
// The artifact and changes pop-ups: one alone is big, both open side by
// side at half size. A click around them closes both.
import { computed } from 'vue'
import { jockey } from '../../lib/state'
import ArtifactPopup from './ArtifactPopup.vue'
import DiffPopup from './DiffPopup.vue'
import DiagramPopup from './DiagramPopup.vue'
// A diagram opened full size covers them; closing it goes back to them.
const size = computed(() => (jockey.artifactOf && jockey.diffOf ? 'pop-half' : 'pop-big'))
</script>

<template>
  <button class="scrim dim" aria-label="Close" @click="jockey.closeViews()"></button>
  <div v-if="jockey.diagramOf" class="pair">
    <DiagramPopup class="pop-big" />
  </div>
  <div v-else class="pair">
    <ArtifactPopup v-if="jockey.artifactOf" :class="size" />
    <DiffPopup v-if="jockey.diffOf" :class="size" />
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
