<script setup lang="ts">
// The title line of the bottom-right tabs (Report, Changes, PR, TODOs): a
// title, a fainter detail cut short first, then the tab's figures and ghost
// buttons in the default slot.
defineProps<{ title: string; meta?: string; tip?: string }>()
</script>

<template>
  <div class="ph">
    <span class="t" :title="tip">{{ title }}</span>
    <span v-if="meta" class="m" :title="meta">{{ meta }}</span>
    <span class="grow"></span>
    <slot />
  </div>
</template>

<style scoped>
  .ph {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    height: 24px;
    min-width: 0;
    white-space: nowrap;
    font-size: var(--fs-12);
  }
  .t {
    flex: none;
    max-width: 70%;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--text);
    font-size: var(--fs-13);
    font-weight: 600;
  }
  .m {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--faint);
  }
  .grow {
    flex: 1;
  }
  /* ghost buttons: an icon, or an icon and a word */
  .ph > :slotted(button) {
    flex: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 5px;
    min-width: 24px;
    height: 24px;
    padding: 0 5px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--muted);
    font: inherit;
  }
  .ph > :slotted(button:hover:not(:disabled)) {
    background: var(--hover);
    color: var(--text);
  }
  .ph > :slotted(button:disabled) {
    opacity: 0.4;
  }
  .ph > :slotted(button:last-child) {
    margin-right: -5px;
  }
  .ph > :slotted(button + button) {
    margin-left: -6px;
  }
</style>
