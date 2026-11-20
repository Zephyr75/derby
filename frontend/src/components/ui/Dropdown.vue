<script setup lang="ts">
// derby's dropdown: a chip that opens a small menu. The chip shows the
// picked item, or the slot's content (e.g. "+ Add").
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import Icon from './Icon.vue'

export interface DropItem {
  value: string
  label: string
  disabled?: boolean
  title?: string
  tag?: string // a short coloured label before it, e.g. the issue's priority
  tagClass?: string // its colour: a status class (s-failed, s-waiting…)
}

const props = withDefaults(defineProps<{ items: DropItem[]; value?: string; placeholder?: string; wide?: boolean; strong?: boolean; disabled?: boolean }>(), {
  value: '',
  placeholder: '',
})
const emit = defineEmits<{ pick: [value: string] }>()
const picked = computed(() => props.items.find((i) => i.value === props.value))
const open = ref(false)
const root = ref<HTMLElement>()

function choose(it: DropItem) {
  if (it.disabled) return
  open.value = false
  emit('pick', it.value)
}
// Closes on a click anywhere else (WebKit does not focus buttons on click, so
// focus events cannot tell).
const outside = (e: PointerEvent) => root.value && !root.value.contains(e.target as Node) && (open.value = false)
watch(open, (on) => (on ? window.addEventListener('pointerdown', outside, true) : window.removeEventListener('pointerdown', outside, true)))
onBeforeUnmount(() => window.removeEventListener('pointerdown', outside, true))
</script>

<template>
  <span ref="root" class="dd" :class="{ wide }" @keydown.esc.stop="open = false">
    <button type="button" class="chip" :class="{ on: open, strong }" :disabled="disabled" aria-haspopup="menu" :aria-expanded="open" @click="open = !open">
      <slot>
        <span v-if="picked?.tag" :class="['tag', picked.tagClass]">{{ picked.tag }}</span>
        <span class="label">{{ picked?.label ?? placeholder }}</span>
        <Icon name="chevron" :size="12" />
      </slot>
    </button>
    <span v-if="open" class="menu popup" role="menu">
      <button v-for="it in items" :key="it.value" type="button" class="item" :class="{ cur: it.value === value }" :disabled="it.disabled" :title="it.title" role="menuitem" @click="choose(it)">
        <span v-if="it.tag" :class="['tag', it.tagClass]">{{ it.tag }}</span>{{ it.label }}
      </button>
    </span>
  </span>
</template>

<style scoped>
  .dd {
    position: relative;
    display: inline-flex;
    min-width: 0;
  }
  .dd.wide,
  .dd.wide .chip {
    width: 100%;
  }
  .chip {
    display: flex;
    align-items: center;
    gap: 5px;
    min-width: 0;
    height: 26px;
    padding: 0 8px 0 10px;
    border-radius: 4px;
    border: 1px solid var(--border);
    background: transparent;
    color: var(--text);
    font-size: var(--fs-12-5);
  }
  .chip.strong {
    font-weight: 600;
    font-size: var(--fs-13);
  }
  .chip:hover:not(:disabled),
  .chip.on {
    background: var(--hover);
  }
  .label {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: left;
  }
  .menu {
    position: absolute;
    z-index: 30;
    top: calc(100% + 4px);
    left: 0;
    min-width: 100%;
    max-width: 560px;
    max-height: 50vh;
    overflow: auto;
    padding: 4px;
    display: flex;
    flex-direction: column;
  }
  .item {
    flex: none; /* a long list scrolls instead of squeezing the rows */
    padding: 5px 8px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--text);
    text-align: left;
    font-size: var(--fs-12-5);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .tag {
    flex: none;
    display: inline-block;
    margin-right: 7px;
    padding: 0 6px;
    border-radius: 4px;
    font-size: var(--fs-11);
    font-weight: 600;
    line-height: 17px;
  }
  .item:hover:not(:disabled) {
    background: var(--hover);
  }
  .item.cur {
    background: var(--sel);
  }
</style>
