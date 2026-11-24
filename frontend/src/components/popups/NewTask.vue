<script setup lang="ts">
// New task (⌘N): a project, then an issue, a description, or both. A picked
// issue shows in full (description, comments, images) before the task starts.
import { computed, onMounted, ref, watch } from 'vue'
import { api, errorText } from '../../lib/api'
import { derby } from '../../lib/state'
import { isMac } from '../../lib/keymap'
import Icon from '../ui/Icon.vue'
import Dropdown from '../ui/Dropdown.vue'
import ArtifactView from '../ArtifactView.vue'
import type { Artifact, Issue } from '../../lib/types'

// derby.previewNew pre-fills the preview's artboards 4 to 6 in the browser mock.
const presets: Record<string, { p: string; i: string; d: string }> = {
  issue: { p: 'core', i: '1190', d: '' },
  both: { p: 'core', i: '1190', d: 'Start with the API flag only; the app change is a separate task.' },
  local: { p: 'rider', i: '', d: 'Add haptic feedback when a ride starts and ends.' },
}
const pre = presets[derby.previewNew]

const project = ref(pre?.p ?? derby.selTask?.project ?? derby.projects[0]?.id ?? '')
const issue = ref(pre?.i ?? '')
// The repository of the picked issue when it is not the project's issues repository.
const issueRepo = ref('')
// What the API takes: a number, or owner/name#number for another repository.
const issueRef = computed(() => (issueRepo.value ? `${issueRepo.value}#${issue.value.trim()}` : issue.value.trim()))
const desc = ref(pre?.d ?? '')
const files = ref<string[]>([])
const issueTitle = ref('')
const titleError = ref('')
const busy = ref(false)
const error = ref('')
const issueInput = ref<HTMLInputElement>()
onMounted(() => {
  issueInput.value?.focus()
  if (pre?.i) void fetchTitle()
})

// The picked issue in full, as Claude will read it (ISSUE.md).
const preview = ref<Artifact | null>(null)
const previewError = ref('')
const previewOpen = ref(true)
let pseq = 0
async function loadPreview() {
  const want = ++pseq
  preview.value = null
  previewError.value = ''
  if (!nIssue.value || !issuesRepo.value) return
  try {
    const a = await api.IssuePreview(project.value, issueRef.value)
    if (want === pseq) preview.value = a
  } catch (e) {
    if (want === pseq) previewError.value = errorText(e)
  }
}
// The title already shows on the issue line: the preview starts at the description.
const previewArt = computed(() => preview.value && { ...preview.value, body: preview.value.body.replace(/^\s*# .*\n+/, '') })
const front = (k: string) => preview.value?.front?.find((f) => f.k === k)?.v ?? ''
const comments = computed(() => (preview.value?.body.split('\n## Comments\n')[1]?.match(/^### /gm) ?? []).length)
const previewMeta = computed(() =>
  [front('author') && 'by ' + front('author'), front('created'), front('labels'), comments.value ? `${comments.value} comment${comments.value > 1 ? 's' : ''}` : 'no comments'].filter(Boolean).join(' · '),
)

const p = computed(() => derby.project(project.value))
const nIssue = computed(() => issue.value.trim())
const nDesc = computed(() => desc.value.trim())
const mode = computed(() => (nIssue.value && nDesc.value ? 'both' : nIssue.value ? 'issue' : nDesc.value ? 'local' : 'none'))
// The branch: the prefix from Settings, then words from the issue title or
// the description, until the user edits it.
const prefix = computed(() => derby.settings?.config.branchPrefix ?? 'ar-')
const words = (s: string) => s.toLowerCase().replace(/[^a-z0-9]+/g, ' ').trim().split(/\s+/).filter(Boolean).slice(0, 7).join('-')
const branchEdited = ref(false)
const branchInput = ref('')
const suggested = computed(() => prefix.value + (words(issueTitle.value) || words(nDesc.value) || nIssue.value || 'task'))
const branch = computed({
  get: () => (branchEdited.value ? branchInput.value : suggested.value),
  set: (v: string) => {
    branchEdited.value = true
    branchInput.value = v.replace(/\s+/g, '-')
  },
})
const branchOk = computed(() => branch.value.startsWith(prefix.value) && branch.value.length > prefix.value.length && /^[A-Za-z0-9._/-]+$/.test(branch.value))

// One quiet line in the footer says what Claude will start from.
const hint = computed(
  () =>
    ({
      issue: `Claude plans from issue ${issueRepo.value}#${nIssue.value}`,
      both: `Claude plans from issue ${issueRepo.value}#${nIssue.value} and your instructions`,
      local: 'LOCAL task: Claude works from your instructions',
      none: 'Pick an issue, describe the task, or both',
    })[mode.value],
)
const keys = isMac() ? '⌘⏎' : 'Ctrl+⏎'
const first = computed(() => derby.settings?.agents.find((a) => !a.builtin)?.name ?? 'Plan')
const valid = computed(() => mode.value !== 'none' && !!p.value && !busy.value && branchOk.value)

function onIssue(e: Event) {
  const el = e.target as HTMLInputElement
  // An issue URL or owner/name#number pasted here names its repository too.
  const m = el.value.trim().match(/^(?:(?:https:\/\/github\.com\/)?([\w.-]+\/[\w.-]+)(?:#|\/issues\/))?#?(\d+)\b/)
  if (m?.[1]) issueRepo.value = m[1] !== issuesRepo.value ? m[1] : ''
  issue.value = m ? m[2] : el.value.replace(/[^0-9]/g, '')
  if (!issue.value) issueRepo.value = ''
  el.value = issue.value
  issueTitle.value = ''
  titleError.value = ''
  pseq++
  preview.value = null
  previewError.value = ''
}

async function fetchTitle() {
  if (!nIssue.value || !issuesRepo.value) return
  if (!preview.value) void loadPreview()
  const want = nIssue.value
  try {
    const t = await api.IssueTitle(project.value, issueRef.value)
    if (want === nIssue.value) issueTitle.value = t
  } catch (e) {
    if (want === nIssue.value) titleError.value = errorText(e)
  }
}

// Where the project's issues live: the common issues repository when it uses it.
const issuesRepo = computed(() => (p.value?.commonIssues && derby.settings?.config.issuesRepo) || p.value?.repo || '')
// Open issues assigned to you; the # field takes any other issue.
const mine = ref<Issue[] | null>(null)
const mineError = ref('')
let mseq = 0
watch(
  project,
  async (id) => {
    const want = ++mseq
    mine.value = null
    mineError.value = ''
    if (!issuesRepo.value) return
    try {
      const is = await api.MyIssues(id)
      if (want === mseq) mine.value = is
    } catch (e) {
      if (want === mseq) (mineError.value = errorText(e)), (mine.value = [])
    }
  },
  { immediate: true },
)
// The issue's Priority field, coloured with the status colours of the theme.
const priorityClass: Record<string, string> = { urgent: 's-failed', high: 's-waiting', medium: 's-running', low: 's-idle' }
const hasTask = (n: number) => derby.tasks.some((t) => t.project === project.value && t.issue === String(n))
const shortRepo = (r: string) => r.split('/').pop() ?? r
// Issues of another repository than the issues repository are labelled with it.
const otherRepo = (is: Issue) => (is.repo && is.repo !== issuesRepo.value ? is.repo : '')
const mineItems = computed(() =>
  (mine.value ?? []).map((is) => ({
    value: `${is.repo}#${is.number}`,
    label: `${otherRepo(is) ? shortRepo(is.repo) : ''}#${is.number} ${is.title}${hasTask(is.number) ? ' (has a task)' : ''}`,
    tag: is.priority,
    tagClass: priorityClass[is.priority.toLowerCase()] ?? 's-idle',
    disabled: hasTask(is.number),
  })),
)
const mineTip = computed(() =>
  mine.value === null ? 'Loading…' : mineError.value ? 'Could not load: ' + mineError.value : mine.value.length ? 'Most urgent first' : 'None assigned to you',
)
const mineLabel = computed(() => (mine.value === null ? 'Assigned to you…' : mine.value.length ? `Assigned to you · ${mine.value.length}` : 'None assigned'))
function pickMine(v: string) {
  const is = mine.value?.find((x) => `${x.repo}#${x.number}` === v)
  if (!is) return
  issue.value = String(is.number)
  issueRepo.value = otherRepo(is)
  issueTitle.value = is.title
  titleError.value = ''
  void loadPreview()
}

// Files attached to the task: derby copies them to the run folder and TASK.md lists them.
const baseName = (f: string) => f.split(/[\\/]/).pop() ?? f
async function addFiles() {
  const picked = await derby.attempt('Could not open the file picker', () => api.ChooseFiles('Attach files to the task'))
  for (const f of picked ?? []) if (!files.value.includes(f)) files.value.push(f)
}

// The project, picked in the header; its last item adds one.
const ADD = '\u0000add'
const projectItems = computed(() => [...derby.projects.map((pr) => ({ value: pr.id, label: pr.name, title: pr.path })), { value: ADD, label: 'Add project…', title: 'Add a git repository' }])
const onProject = (v: string) => (v === ADD ? void addProject() : pickProject(v))

// Adds a project without leaving the pop-up, then selects it.
async function addProject() {
  const pr = await derby.attempt('Could not add the project', () => api.AddProject())
  if (!pr) return
  derby.toast('Added ' + pr.name, pr.path)
  pickProject(pr.id)
}

function pickProject(id: string) {
  project.value = id
  issueRepo.value = ''
  issueTitle.value = ''
  preview.value = null
  titleError.value = ''
  void fetchTitle()
}

async function create() {
  if (!valid.value) return
  busy.value = true
  error.value = ''
  try {
    const t = await api.CreateTask(project.value, issueRef.value, nDesc.value, branch.value, true, files.value)
    derby.overlay = 'none'
    // The task:changed event may arrive after the call returns.
    if (!derby.task(t.id)) derby.tasks.push(t)
    derby.pickTask(t.id)
  } catch (e) {
    error.value = errorText(e)
  } finally {
    busy.value = false
  }
}

function key(e: KeyboardEvent) {
  const mod = isMac() ? e.metaKey : e.ctrlKey
  if (e.key === 'Enter' && mod) {
    e.preventDefault()
    void create()
  }
}
</script>

<template>
  <button class="scrim dim" aria-label="Cancel new task" @click="derby.overlay = 'none'"></button>
  <div class="box popup pop pop-small" role="dialog" aria-label="New task" tabindex="-1" @keydown="key">
    <div class="head">
      <Dropdown class="proj" :items="projectItems" :value="project" placeholder="Pick a project" title="The project to work in" @pick="onProject" />
      <span class="grow"></span>
      <button class="x" aria-label="Close" @click="derby.overlay = 'none'"><Icon name="close" :size="14" /></button>
    </div>
    <div class="body">
      <div class="issue-line">
        <label for="nfissue" class="hash">#</label>
        <input id="nfissue" ref="issueInput" class="num" :value="issue" placeholder="Issue" inputmode="numeric" autocomplete="off" @input="onIssue" @blur="fetchTitle" :style="{ width: (issue ? issue.length + 0.5 : 5) + 'ch' }" />
        <span v-if="titleError" class="itext err" :title="titleError">{{ titleError }}</span>
        <span v-else-if="issueTitle" class="itext" :title="issueTitle">{{ issueTitle }}</span>
        <span v-else class="itext faint">{{ issuesRepo ? '' : 'the project has no GitHub repo' }}</span>
        <Dropdown v-if="issuesRepo" class="mine" :items="mineItems" :value="issueRepo ? `${issueRepo}#${nIssue}` : `${issuesRepo}#${nIssue}`" :disabled="!mine?.length" :title="mineTip" @pick="pickMine">
          <span class="mine-label">{{ mineLabel }}</span><Icon name="chevron" :size="12" />
        </Dropdown>
      </div>
      <div v-if="nIssue && (preview || previewError)" class="issue-view">
        <button class="iv-head" :aria-expanded="previewOpen" @click="previewOpen = !previewOpen">
          <Icon :name="previewOpen ? 'chevron' : 'chevron-right'" :size="12" />
          <span class="iv-meta">{{ previewError ? 'Could not load the issue' : previewMeta }}</span>
        </button>
        <div v-if="previewOpen" class="iv-body">
          <p v-if="previewError" class="err">{{ previewError }}</p>
          <ArtifactView v-else-if="previewArt" :art="previewArt" task="" :zoom="1" />
        </div>
      </div>
      <textarea id="nfdesc" v-model="desc" rows="5" aria-label="Instructions" :placeholder="nIssue ? 'Anything to add on top of the issue? (optional)' : 'What should Claude do?'"></textarea>
      <div v-if="files.length" class="files">
        <span v-for="f in files" :key="f" class="file" :title="f">
          <Icon name="file" :size="12" />{{ baseName(f) }}
          <button class="fx" :aria-label="`Remove ${baseName(f)}`" @click="files = files.filter((x) => x !== f)"><Icon name="close" :size="11" /></button>
        </span>
      </div>
    </div>
    <div class="details">
      <button class="btn ghost quiet" title="Attach files" @click="addFiles"><Icon name="plus" :size="13" />Attach files</button>
      <span class="grow"></span>
      <label class="branch" :class="{ bad: mode !== 'none' && !branchOk }" :title="`Starts with ${prefix}`">
        <Icon name="branch" :size="13" />
        <input id="nfbranch" v-model="branch" aria-label="Branch" autocomplete="off" spellcheck="false" :style="{ width: branch.length + 1 + 'ch' }" />
      </label>
    </div>
    <p v-if="mode !== 'none' && !branchOk" class="err line">The branch starts with {{ prefix }}, then names the task: letters, digits, . _ / -</p>
    <p v-if="error" class="err line">{{ error }}</p>
    <div class="foot">
      <span class="hint" :class="{ local: mode === 'local' }">{{ hint }}</span>
      <span class="grow"></span>
      <button class="btn ghost" @click="derby.overlay = 'none'">Cancel</button>
      <button class="btn primary" :disabled="!valid" :title="keys" @click="create()">Create and run {{ first }}<span class="keys">{{ keys }}</span></button>
    </div>
  </div>
</template>

<style scoped>
  .box {
    outline: none;
  }
  .head {
    flex: none;
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 10px 10px 0 12px;
  }
  .proj :deep(.chip) {
    border-color: transparent;
    padding: 0 6px 0 8px;
    font-weight: 600;
  }
  .grow {
    flex: 1;
  }
  .x {
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
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-height: 0;
    padding: 10px 20px 8px;
  }
  .issue-line {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    height: 32px;
  }
  .hash {
    color: var(--faint);
    font-family: var(--mono);
    font-size: var(--fs-15);
  }
  .num {
    flex: none;
    margin-left: -6px;
    padding: 0;
    border: 0;
    outline: none;
    background: transparent;
    color: var(--text);
    font-family: var(--mono);
    font-size: var(--fs-15);
  }
  .num::placeholder {
    color: var(--faint);
    font-family: var(--ui-font);
  }
  .itext {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: var(--fs-14);
    font-weight: 600;
  }
  .itext.faint {
    color: var(--faint);
    font-weight: 400;
    font-size: var(--fs-12-5);
  }
  .itext.err {
    font-weight: 400;
    font-size: var(--fs-12-5);
  }
  .mine :deep(.chip) {
    border-color: transparent;
    color: var(--muted);
  }
  .mine :deep(.chip:hover:not(:disabled)) {
    color: var(--text);
  }
  .mine :deep(.menu) {
    left: auto;
    right: 0;
    width: 460px;
  }
  .mine-label {
    white-space: nowrap;
  }
  /* The issue takes what it needs; the instructions keep the rest. */
  .issue-view {
    flex: 0 1 auto;
    min-height: 0;
    display: flex;
    flex-direction: column;
    margin-left: -4px;
  }
  .iv-head {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    height: 22px;
    padding: 0 4px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--muted);
    font-size: var(--fs-12);
    text-align: left;
  }
  .iv-head:hover {
    color: var(--text);
  }
  .iv-meta {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .iv-body {
    min-height: 0;
    overflow: auto;
    margin: 4px 0 4px 10px;
    padding: 0 10px 0 12px;
    border-left: 2px solid var(--border);
  }
  .iv-body :deep(.md) {
    font-size: var(--fs-12-5);
  }
  /* ISSUE.md's sections (Comments, each author): plain bold lines */
  .iv-body :deep(h1),
  .iv-body :deep(h2),
  .iv-body :deep(h3),
  .iv-body :deep(h4) {
    margin: 12px 0 4px;
    padding: 0;
    border: 0;
    font-size: var(--fs-12-5);
  }
  .iv-body :deep(h2) {
    color: var(--muted);
  }
  .iv-body :deep(img) {
    max-width: 100%;
    height: auto;
    border-radius: 4px;
  }
  textarea {
    flex: 1 1 96px;
    min-height: 96px;
    padding: 6px 0;
    resize: none;
    border: 0;
    outline: none;
    background: transparent;
    color: var(--text);
    font-size: var(--fs-13);
    line-height: 1.55;
  }
  textarea::placeholder {
    color: var(--faint);
  }
  .files {
    flex: none;
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .file {
    display: flex;
    align-items: center;
    gap: 5px;
    height: 24px;
    padding: 0 2px 0 7px;
    border-radius: 5px;
    background: var(--surface);
    color: var(--muted);
    font-size: var(--fs-12);
  }
  .fx {
    display: flex;
    padding: 3px;
    border: 0;
    border-radius: 4px;
    background: none;
    color: var(--muted);
  }
  .fx:hover {
    background: var(--hover);
    color: var(--text);
  }
  .details {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 12px;
    border-top: 1px solid var(--border);
  }
  .quiet {
    padding: 0 8px;
    color: var(--muted);
    font-size: var(--fs-12-5);
  }
  .quiet:hover {
    color: var(--text);
  }
  .branch {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    max-width: 60%;
    height: 26px;
    padding: 0 8px;
    border-radius: 6px;
    color: var(--muted);
  }
  .branch:hover,
  .branch:focus-within {
    background: var(--hover);
  }
  .branch.bad {
    color: var(--approval);
  }
  .branch input {
    flex: 0 1 auto;
    min-width: 8ch;
    padding: 0;
    border: 0;
    outline: none;
    background: transparent;
    color: var(--text);
    font-family: var(--mono);
    font-size: var(--fs-12);
  }
  .err {
    margin: 0;
    font-size: var(--fs-12);
    color: var(--approval);
  }
  .err.line {
    flex: none;
    padding: 0 20px 6px;
  }
  .foot {
    flex: none;
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 12px 10px 20px;
    border-top: 1px solid var(--border);
  }
  .hint {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: var(--fs-12);
    color: var(--muted);
  }
  .hint.local {
    color: var(--toggle-text);
  }
  .keys {
    margin-left: 4px;
    opacity: 0.7;
    font-size: var(--fs-11-5);
  }
</style>
