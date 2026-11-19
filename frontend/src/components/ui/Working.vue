<script setup lang="ts">
// Claude is working: three balls in Claude's orange (Claude Code's "claude"
// colour) swelling in turn, as Claude's desktop app shows it.
defineProps<{ label?: string }>()
</script>

<template>
  <span class="working" role="status" :aria-label="label || 'Claude is working'">
    <span class="ball"></span><span class="ball"></span><span class="ball"></span>
    <span v-if="label" class="label">{{ label }}</span>
  </span>
</template>

<style scoped>
.working {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  --claude: #d77757;
}
.ball {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--claude);
  animation: swell 1.2s ease-in-out infinite;
}
.ball:nth-child(2) {
  animation-delay: 0.15s;
}
.ball:nth-child(3) {
  animation-delay: 0.3s;
}
@keyframes swell {
  0%,
  80%,
  100% {
    transform: scale(0.55);
    opacity: 0.45;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}
.label {
  margin-left: 4px;
  font-size: var(--fs-12);
  color: var(--muted);
}
</style>
