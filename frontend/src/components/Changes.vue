<script setup lang="ts">
// Files a step changed: its commit, or the uncommitted work while it is the
// task's latest step. A click opens the diff pop-up on that file.
import { computed } from 'vue'
import { jockey } from '../lib/state'
import type { Changes } from '../lib/types'
import ChangedFiles, { sourceLabel, totalsOf } from './ChangedFiles.vue'
import NotesBar from './NotesBar.vue'
import PaneHead from './ui/PaneHead.vue'

const props = defineProps<{ changes: Changes | null }>()
const totals = computed(() => props.changes && totalsOf(props.changes))

function pick(path: string) {
  if (jockey.selStep) jockey.openDiffPopup(jockey.selStep.id, path)
}
</script>

<template>
  <p v-if="!changes" class="empty">Loading…</p>
  <p v-else-if="!changes.source" class="empty">This step did not change code: it only wrote its artifact.</p>
  <p v-else-if="!changes.files.length" class="empty">{{ changes.source === 'branch' ? `The branch changes nothing against ${changes.base}.` : changes.source === 'worktree' ? 'No uncommitted changes in the worktree.' : 'The commit changed no files.' }}</p>
  <template v-else>
    <PaneHead :title="`${totals!.n} file${totals!.n === 1 ? '' : 's'}`" :meta="sourceLabel(changes)">
      <span class="c-add">+{{ totals!.add }}</span>
      <span class="c-del">−{{ totals!.del }}</span>
    </PaneHead>
    <NotesBar v-if="jockey.selTask?.notes.length" class="notes" :task="jockey.selTask" />
    <ChangedFiles :changes="changes" bare @pick="pick" />
    <p class="hint">Open a file and click the + that shows on a line to comment on it (drag it for several lines).</p>
  </template>
</template>

<style scoped>
.notes {
  margin-bottom: 10px;
}
.c-add,
.c-del {
  font-family: var(--mono);
  font-size: var(--fs-11-5);
}
.c-add {
  color: var(--done);
}
.c-del {
  color: var(--approval);
  margin-left: -2px;
}
.hint {
  margin: 10px 0 0;
  color: var(--faint);
  font-size: var(--fs-12);
}
.empty {
  margin: 0;
  color: var(--muted);
  font-size: var(--fs-12-5);
  line-height: 1.5;
}
</style>
