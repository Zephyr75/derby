<script setup lang="ts">
// What a task is about, opened from its name in the right panel: its issue
// (description, images and comments, read again from GitHub) and the
// instructions the user wrote when creating it.
import { computed, ref, watch } from 'vue'
import { api, errorText } from '../../lib/api'
import { derby, taskName } from '../../lib/state'
import Icon from '../ui/Icon.vue'
import TaskRef from '../ui/TaskRef.vue'
import ArtifactView from '../ArtifactView.vue'
import type { Artifact, TaskIssue } from '../../lib/types'

const t = computed(() => derby.selTask)
const close = () => (derby.overlay = 'none')

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
const heading = computed(() => issue.value?.title || (t.value ? taskName(t.value) : ''))
const meta = computed(() => {
  const i = issue.value
  if (!t.value?.issue) return 'No GitHub issue · LOCAL task'
  if (!i) return error.value ? '' : 'Loading the issue…'
  if (i.offline) return `${i.repo} · the copy made when the task was created (GitHub: ${i.offline})`
  return [i.state, i.author && `opened by ${i.author}`, i.created, i.labels.join(', '), i.repo].filter(Boolean).join(' · ')
})
const url = computed(() => issue.value?.url || (t.value?.issue ? `https://github.com/${t.value.issueRepo || t.value.repo}/issues/${t.value.issue}` : ''))
const openGitHub = () => url.value && derby.attempt('Could not open the issue', () => api.OpenURL(url.value))
</script>

<template>
  <button class="scrim dim" aria-label="Close the description" @click="close"></button>
  <div v-if="t" class="box popup pop pop-half" role="dialog" aria-label="Task description">
    <div class="head">
      <div class="line">
        <TaskRef :issue="t.issue" :task="t.id" />
        <span class="heading" :title="heading">{{ heading }}</span>
        <button v-if="url" class="x" title="Open the issue on GitHub" aria-label="Open on GitHub" @click="openGitHub"><Icon name="external" :size="14" /></button>
        <button class="x" aria-label="Close" @click="close"><Icon name="close" :size="14" /></button>
      </div>
      <span v-if="meta" class="meta">{{ meta }}</span>
    </div>
    <div class="body">
      <section v-if="t.description">
        <span class="label">Your instructions</span>
        <p class="instructions">{{ t.description }}</p>
      </section>
      <section v-if="t.issue">
        <span v-if="t.description" class="label">Issue</span>
        <p v-if="error" class="err">Could not load issue #{{ t.issue }}: {{ error }}</p>
        <ArtifactView v-else-if="issue?.body" class="issue-md" :art="md(issue.body)" :task="t.id" :zoom="1" />
        <p v-else-if="issue" class="muted">No description.</p>
      </section>
      <section v-if="issue?.comments.length">
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
.box {
  overflow: hidden;
}
.head {
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 10px 12px 16px;
  border-bottom: 1px solid var(--border);
}
.line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.heading {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: var(--fs-14);
  font-weight: 600;
}
.meta {
  padding-right: 6px;
  font-size: var(--fs-12);
  color: var(--muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.x {
  flex: none;
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--muted);
}
.x:hover {
  background: var(--hover);
  color: var(--text);
}
.body {
  display: flex;
  flex-direction: column;
  gap: 22px;
  padding: 16px 16px 22px;
  overflow: auto;
  user-select: text;
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
