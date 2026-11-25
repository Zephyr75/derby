<script setup lang="ts">
// The task itself, so there is no need to open GitHub: the Task step's page in
// the centre, and its short form (compact) in the right panel's Report tab. It
// shows the user's instructions, the issue (its description and images, read
// again from GitHub) and its comments.
import { computed, ref, watch } from 'vue'
import { api, errorText } from '../lib/api'
import { derby } from '../lib/state'
import ArtifactView from './ArtifactView.vue'
import type { Artifact, TaskIssue } from '../lib/types'

defineProps<{ compact?: boolean }>()

const t = computed(() => derby.selTask)

const issue = ref<TaskIssue | null>(null)
const error = ref('')
let seq = 0
watch(
  () => t.value?.id,
  async (id) => {
    const want = ++seq
    issue.value = null
    error.value = ''
    if (!id || !t.value?.issue) return
    try {
      const v = await api.GetTaskIssue(id)
      if (want === seq) issue.value = v
    } catch (e) {
      if (want === seq) error.value = errorText(e)
    }
  },
  { immediate: true },
)

const md = (body: string): Artifact => ({ path: '', file: '', exists: true, front: null, body })
const heading = computed(() => issue.value?.title || t.value?.title || '')
const meta = computed(() => {
  const i = issue.value
  if (!t.value?.issue) return 'No GitHub issue · LOCAL task'
  if (!i) return error.value ? '' : 'Loading the issue…'
  if (i.offline) return `${i.repo} · the copy made when the task was created (GitHub: ${i.offline})`
  return [i.state, i.author && `opened by ${i.author}`, i.created, i.labels.join(', '), i.repo].filter(Boolean).join(' · ')
})
</script>

<template>
  <div v-if="t" class="task" :class="{ page: !compact }">
    <div class="inner">
      <header v-if="!compact" class="head">
        <span class="heading">{{ heading }}</span>
        <span v-if="meta" class="meta">{{ meta }}</span>
      </header>
      <section v-if="t.description">
        <span class="label">Your instructions</span>
        <p class="instructions">{{ t.description }}</p>
      </section>
      <section v-if="t.issue">
        <span v-if="t.description" class="label">Issue</span>
        <p v-if="error" class="err">Could not load issue #{{ t.issue }}: {{ error }}</p>
        <ArtifactView v-else-if="issue?.body" class="issue-md" :art="md(issue.body)" :task="t.id" :zoom="1" />
        <p v-else-if="issue" class="muted">No description.</p>
        <p v-else class="muted">Loading the issue…</p>
      </section>
      <p v-if="!t.description && !t.issue" class="muted">This task has no description.</p>
      <section v-if="!compact && issue?.comments.length">
        <span class="label">{{ issue.comments.length }} comment{{ issue.comments.length > 1 ? 's' : '' }}</span>
        <div v-for="(c, i) in issue.comments" :key="i" class="comment">
          <div class="who">
            <span class="author">{{ c.author }}</span>
            <span class="when">{{ c.created }}</span>
          </div>
          <ArtifactView class="issue-md" :art="md(c.body)" :task="t.id" :zoom="1" />
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
/* the centre view scrolls on its own; the right panel's tab already does */
.task.page {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 18px 24px 32px;
  user-select: text;
}
.inner {
  display: flex;
  flex-direction: column;
  gap: 24px;
  max-width: 860px;
  margin: 0 auto;
}
.task:not(.page) .inner {
  gap: 14px;
}
.head {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.heading {
  font-size: var(--fs-14);
  font-weight: 600;
}
.meta {
  font-size: var(--fs-12);
  color: var(--muted);
}
section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.label {
  font-size: var(--fs-11-5);
  font-weight: 600;
  color: var(--muted);
}
.instructions {
  margin: 0;
  white-space: pre-wrap;
  line-height: 1.6;
}
.comment {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px 0 2px;
  border-top: 1px solid var(--border);
}
.comment:first-of-type {
  border-top: 0;
  padding-top: 0;
}
.who {
  display: flex;
  align-items: baseline;
  gap: 8px;
  font-size: var(--fs-12);
}
.author {
  font-weight: 600;
}
.when {
  color: var(--muted);
}
.err {
  margin: 0;
  font-size: var(--fs-12-5);
  color: var(--approval);
}
.muted {
  margin: 0;
  color: var(--muted);
}
/* Issues use headings for short sections: plain bold lines here. */
.issue-md :deep(h1),
.issue-md :deep(h2),
.issue-md :deep(h3),
.issue-md :deep(h4) {
  margin: 14px 0 6px;
  padding: 0;
  border: 0;
  font-size: var(--fs-13);
}
.issue-md :deep(img) {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 4px 0 10px;
  border-radius: 6px;
}
.issue-md :deep(p:last-child) {
  margin-bottom: 0;
}
</style>
