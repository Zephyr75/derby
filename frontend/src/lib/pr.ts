// The PR view's behaviour: Copilot's latest review and the CI checks, with
// buttons to ask Copilot again, rerun checks and start the steps that fix them.
import { computed, ref } from 'vue'
import { api } from './api'
import { derby } from './state'
import type { CICheck, Feedback, Step, Task } from './types'

const rank: Record<string, number> = { fail: 0, cancel: 1, pending: 2, pass: 3, skipping: 4 }
export const word = (b: string) => ({ fail: 'failed', cancel: 'cancelled', pending: 'running', pass: 'passed', skipping: 'skipped' })[b] ?? b
export const where = (f: Feedback) => (f.path ? `${f.path}:${f.line}` : 'on the PR')

export function usePR(props: { task: Task; step?: Step }) {
  // The Copilot step shown: the one opened, else the task's latest.
  const cop = computed(() => props.step ?? [...props.task.steps].reverse().find((s) => s.external))
  const live = computed(() => props.task.prState === 'open' && !props.task.archived)
  const ready = computed(() => derby.prReady(props.task))

  // One row per workflow (Tests, Regtests, Lint…), worst check first.
  const workflows = computed(() => {
    const by = new Map<string, CICheck[]>()
    for (const c of props.task.checks) by.set(c.workflow || c.name, [...(by.get(c.workflow || c.name) ?? []), c])
    return [...by.entries()]
      .map(([name, cs]) => {
        cs = [...cs].sort((a, b) => rank[a.bucket] - rank[b.bucket])
        const passed = cs.filter((c) => c.bucket === 'pass' || c.bucket === 'skipping').length
        return { name, cs, bucket: cs[0].bucket, passed, open: cs.filter((c) => c.bucket !== 'pass' && c.bucket !== 'skipping') }
      })
      .sort((a, b) => rank[a.bucket] - rank[b.bucket] || a.name.localeCompare(b.name))
  })
  const counts = computed(() => {
    const n = { fail: 0, pending: 0, pass: 0, total: props.task.checks.length }
    for (const c of props.task.checks) {
      if (c.bucket === 'fail' || c.bucket === 'cancel') n.fail++
      else if (c.bucket === 'pending') n.pending++
      else n.pass++
    }
    return n
  })

  const waiting = computed(() => cop.value?.state === 'running')
  const open = (link: string) => link && derby.attempt('Could not open the link', () => api.OpenURL(link))
  const stop = () => cop.value && derby.attempt('Could not stop waiting', () => api.MarkDone(cop.value!.id))
  const askCopilot = () => derby.attempt('Could not ask Copilot for a review', () => api.RequestCopilotReview(props.task.id))

  // Only GitHub Actions jobs can be rerun from derby; a workflow row reruns its whole run.
  const actions = (link: string) => /\/actions\/runs\/\d+/.test(link)
  const runOf = (link: string) => link.replace(/\/job\/.*$/, '')
  const rerunning = ref('')
  async function rerun(link: string, what: string) {
    rerunning.value = link || 'all'
    await derby.attempt('Could not rerun ' + what, () => api.RerunChecks(props.task.id, link))
    rerunning.value = ''
  }
  const anyActions = computed(() => props.task.checks.some((c) => actions(c.link)))

  // Fixes go to the agent Copilot offers first when there is something to fix
  // (Implement by default), as a new step with guidance.
  const fixer = computed(() => derby.fixers(props.task)[0])
  const failing = computed(() => props.task.checks.filter((c) => c.bucket === 'fail' || c.bucket === 'cancel'))
  const canFix = computed(() => live.value && derby.canFix(props.task))
  const fix = (guidance: string) => derby.fixWith(props.task, guidance)

  // Review comments: every one is sent unless unticked.
  // Ticked by default unless outdated (the code it points at changed since);
  // flipped holds the ones the user ticked or unticked.
  const flipped = ref(new Set<string>())
  // Unticked too when a step rejected it (found it wrong or unnecessary).
  const rejected = (f: Feedback) => props.task.rejected?.[f.id]
  const ticked = (f: Feedback) => (f.outdated || !!rejected(f)) === flipped.value.has(f.id)
  // The GitHub id of a thread's first comment: steps name comments by it.
  const commentId = (f: Feedback) => /#discussion_r(\d+)/.exec(f.url)?.[1] ?? ''
  const toggle = (id: string) => {
    const s = new Set(flipped.value)
    if (!s.delete(id)) s.add(id)
    flipped.value = s
  }
  const openFb = computed(() => props.task.feedback.filter((f) => !f.resolvedBy))
  const resolvedFb = computed(() => props.task.feedback.filter((f) => f.resolvedBy))
  const showResolved = ref(false)
  const picked = computed(() => openFb.value.filter(ticked))
  // Resolve marks a thread resolved on GitHub; it then moves to the resolved ones.
  const resolving = ref('')
  async function resolve(f: Feedback) {
    resolving.value = f.id
    await derby.attempt('Could not resolve the comment', () => api.ResolveThread(props.task.id, f.id))
    resolving.value = ''
  }
  function fixFeedback() {
    const comments = picked.value
      .map((f) => {
        const replies = f.replies.map((r) => `> ${r.author}: ${r.body.replace(/\n/g, '\n> ')}`).join('\n')
        const id = commentId(f) ? ` (comment ${commentId(f)})` : ''
        const was = [f.outdated && 'outdated: the code changed since', rejected(f) && `rejected by ${rejected(f)!.name}`].filter(Boolean).join('; ')
        return `### ${where(f)}${id}${was ? ` (${was})` : ''}, by ${f.author}\n${f.body}${replies ? '\n\n' + replies : ''}`
      })
      .join('\n\n')
    fix(derby.prompt('fixFeedback', { pr: String(props.task.pr), comments }))
  }
  const rebase = () => fix(derby.prompt('rebase', { pr: String(props.task.pr), base: props.task.baseBranch, behind: String(props.task.behind || 'some') }))
  const fixChecks = () =>
    fix(
      derby.prompt('fixChecks', {
        pr: String(props.task.pr),
        checks: failing.value.map((c) => (c.workflow ? `${c.workflow} / ${c.name}` : c.name)).join(', '),
        links: failing.value.map((c) => c.link).filter(Boolean).join(' '),
      }),
    )
  const refreshing = ref(false)
  async function refresh() {
    refreshing.value = true
    await derby.attempt('Could not refresh PR #' + props.task.pr, () => api.RefreshPR(props.task.id))
    refreshing.value = false
  }
  const markReady = () => derby.markReady(props.task)
  const copyLink = () => derby.copy(props.task.prUrl)
  const openStep = (step: string) => derby.openStep(props.task.id, step)

  return {
    cop, live, ready, workflows, counts, waiting, open, stop, askCopilot, actions, runOf, rerunning, rerun, anyActions,
    fixer, failing, canFix, fixFeedback, fixChecks, rebase, rejected, ticked, toggle, openFb, resolvedFb, showResolved, picked,
    resolving, resolve, refreshing, refresh, markReady, copyLink, openStep,
  }
}
