<script setup lang="ts">
// "#4356" or "LOCAL", in mono and gray; with task, a tooltip names its sister tasks.
import { sisterTitle } from '../../lib/sisters'

defineProps<{ issue: string; width?: string; task?: string }>()
</script>

<template>
  <span class="issue-ref" :style="width ? { width } : undefined" :title="task ? sisterTitle(task) : ''">{{ issue ? '#' + issue : 'LOCAL' }}</span>
</template>
