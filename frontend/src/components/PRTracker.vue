<script setup lang="ts">
// A PR as flat lists, like the TODOs and Changes tabs: Copilot's review
// comments to tick and send to a fixing step, then the CI checks.
import { reactive } from 'vue'
import { usePR, word } from '../lib/pr'
import type { Feedback, Step, Task } from '../lib/types'
import Icon from './ui/Icon.vue'

const props = defineProps<{ task: Task; step?: Step }>()
const pr = reactive(usePR(props))

const dot = (b: string) => (b === 'fail' || b === 'cancel' ? 'd-fail' : b === 'pending' ? 'd-run' : b === 'skipping' ? 'd-skip' : 'd-pass')
const file = (f: Feedback) => (f.path ? `${f.path.slice(f.path.lastIndexOf('/') + 1)}:${f.line}` : 'on the PR')
</script>

<template>
  <div :class="['prv', { page: step }]">
    <div class="inner">
      <header class="head">
        <Icon :name="pr.ready === 'draft' ? 'pr-draft' : 'pr'" :size="14" :class="pr.ready === 'ready' ? 'c-pass' : 'muted'" />
        <span class="num">#{{ task.pr }}</span>
        <span v-if="pr.ready" class="muted" :title="pr.ready === 'ready' ? 'Ready for review' : 'A draft: reviewers and repo bots leave it alone until it is marked ready'">{{ pr.ready === 'ready' ? 'Ready' : 'Draft' }}</span>
        <span class="branch" :title="`${task.branch} → ${task.baseBranch}`">{{ task.branch }} → {{ task.baseBranch }}</span>
        <button v-if="pr.ready === 'draft'" class="tb" title="Turn the draft PR into one ready for review and add the ready label" aria-label="Mark ready" @click="pr.markReady"><Icon name="check" :size="13" /><span class="lbl">Mark ready</span></button>
        <button v-if="pr.live" class="ib" :disabled="pr.refreshing" title="Check Copilot's review and the CI checks now instead of waiting for the next minute" aria-label="Refresh" @click="pr.refresh"><Icon name="rerun" :size="13" /></button>
        <button v-if="task.prUrl" class="ib" :title="`Copy the link to PR #${task.pr}`" aria-label="Copy link" @click="pr.copyLink"><Icon name="copy" :size="13" /></button>
        <button v-if="task.prUrl" class="ib" :title="`Open PR #${task.pr} on GitHub`" aria-label="Open PR" @click="pr.open(task.prUrl)"><Icon name="external" :size="13" /></button>
      </header>

      <section v-if="task.feedback.length || pr.live">
        <div class="sh">
          <span class="st">Review</span>
          <span v-if="pr.openFb.length" class="cnt">{{ pr.openFb.length }}</span>
          <span v-if="pr.waiting" class="hint" title="Waiting for Copilot's review"><Icon name="pending" :size="12" class="c-run" /><span class="lbl">waiting for Copilot</span></span>
          <span class="grow"></span>
          <button v-if="pr.waiting" class="tb" title="Stop waiting for Copilot (Mark done)" aria-label="Stop waiting" @click="pr.stop"><Icon name="stop" :size="11" /><span class="lbl">Stop waiting</span></button>
          <button v-if="pr.live" class="tb" title="Ask Copilot to review the PR as it is now; a Copilot step waits for the review" :aria-label="pr.waiting ? 'Ask Copilot again' : 'Ask Copilot'" @click="pr.askCopilot"><Icon name="agent" :size="13" /><span class="lbl">{{ pr.waiting ? 'Ask again' : 'Ask Copilot' }}</span></button>
          <button v-if="pr.canFix && pr.openFb.length" class="tb fix" :disabled="!pr.picked.length" :title="`Start a ${pr.fixer!.name} step that addresses the ticked comments`" @click="pr.fixFeedback"><Icon name="pencil" :size="13" />Fix {{ pr.picked.length }}</button>
        </div>
        <p v-if="!pr.openFb.length" class="empty">No open comments.</p>
        <!-- open comments, then the resolved ones under their toggle -->
        <template v-for="(list, i) in [pr.openFb, pr.showResolved ? pr.resolvedFb : []]" :key="i">
          <button v-if="i === 1 && pr.resolvedFb.length" class="more" :aria-expanded="pr.showResolved" @click="pr.showResolved = !pr.showResolved">
            <Icon :name="pr.showResolved ? 'chevron' : 'chevron-right'" :size="12" />{{ pr.resolvedFb.length }} resolved
          </button>
          <div v-for="f in list" :key="f.id" :class="['item', { off: f.resolvedBy || !pr.ticked(f) }]">
            <span v-if="f.resolvedBy" class="box done" title="Resolved"><Icon name="check" :size="11" :stroke="3" /></span>
            <button v-else :class="['box', { on: pr.ticked(f) }]" role="checkbox" :aria-checked="pr.ticked(f)" :title="pr.ticked(f) ? 'Sent with Fix' : 'Left out of Fix'" :aria-label="'Send the comment of ' + f.author" @click="pr.toggle(f.id)"><Icon v-if="pr.ticked(f)" name="check" :size="11" :stroke="3" /></button>
            <div class="txt">
              <div class="body" :title="f.body">{{ f.body }}</div>
              <div class="meta">
                <span class="mono" :title="f.path ? `${f.path}:${f.line}` : 'A comment on the whole PR'">{{ file(f) }}</span>
                <span>{{ f.author }}</span>
                <span v-if="f.resolvedBy">resolved by {{ f.resolvedBy }}</span>
                <span v-else-if="f.outdated" title="The code it points at changed since">outdated</span>
                <button v-if="!f.resolvedBy && pr.rejected(f)" class="rej" :title="`${pr.rejected(f)!.name} found it wrong or unnecessary and left the code as it is: open its artifact to see why`" @click="pr.openStep(pr.rejected(f)!.step)">rejected</button>
                <span v-if="f.replies.length" :title="`Last: ${f.replies[f.replies.length - 1].author}`">{{ f.replies.length }} {{ f.replies.length > 1 ? 'replies' : 'reply' }}</span>
              </div>
            </div>
            <button v-if="pr.live && f.path && !f.resolvedBy" class="ha" :disabled="pr.resolving === f.id" title="Mark resolved on GitHub" aria-label="Mark resolved on GitHub" @click="pr.resolve(f)"><Icon name="check" :size="14" /></button>
            <button class="ha" title="Open on GitHub" aria-label="Open on GitHub" @click="pr.open(f.url)"><Icon name="external" :size="12" /></button>
          </div>
        </template>
      </section>

      <section>
        <div class="sh">
          <span class="st">Checks</span>
          <span v-if="pr.counts.total" class="cnt">{{ pr.counts.pass }}/{{ pr.counts.total }} passed</span>
          <span class="grow"></span>
          <button v-if="pr.live && pr.anyActions" class="tb" :disabled="!!pr.rerunning" title="Run every check again (each workflow run once)" aria-label="Rerun all" @click="pr.rerun('', 'the checks')"><Icon name="rerun" :size="13" /><span class="lbl">{{ pr.rerunning === 'all' ? 'Rerunning…' : 'Rerun all' }}</span></button>
          <button v-if="pr.canFix && pr.failing.length && !task.conflicting" class="tb fix" :title="`Start a ${pr.fixer!.name} step that reads the failing checks' logs and fixes them`" @click="pr.fixChecks"><Icon name="pencil" :size="13" />Fix</button>
        </div>
        <div v-if="pr.live && task.conflicting" class="note">
          <Icon name="warn" :size="13" class="c-warn" />
          <span class="ntext">Conflicts with {{ task.baseBranch }}: GitHub runs no checks until they are resolved.</span>
          <button v-if="pr.canFix" class="tb fix" :title="`Start a ${pr.fixer!.name} step that rebases the branch on origin/${task.baseBranch} and resolves the conflicts`" @click="pr.rebase"><Icon name="rebase" :size="13" />Rebase</button>
        </div>
        <p v-if="!pr.workflows.length && !task.conflicting" class="empty">No checks reported yet (refreshed every minute).</p>
        <template v-for="w in pr.workflows" :key="w.name">
          <div class="row">
            <button class="rname" :title="w.cs[0].link ? 'Open on GitHub' : ''" @click="pr.open(w.cs[0].link)">
              <span :class="['sd', dot(w.bucket)]"></span><span class="nm">{{ w.name }}</span>
            </button>
            <button v-if="pr.live && pr.actions(w.cs[0].link)" class="ha" :disabled="!!pr.rerunning" :title="`Rerun ${w.name}`" :aria-label="`Rerun ${w.name}`" @click="pr.rerun(pr.runOf(w.cs[0].link), w.name)"><Icon name="rerun" :size="12" /></button>
            <span class="rmeta">{{ w.cs.length > 1 ? `${w.passed}/${w.cs.length}` : word(w.bucket) }}</span>
          </div>
          <div v-for="c in w.open" :key="c.name" class="row sub">
            <button class="rname" title="Open on GitHub" @click="pr.open(c.link)">
              <span :class="['sd', dot(c.bucket)]"></span><span class="nm">{{ c.name }}</span>
            </button>
            <button v-if="pr.live && pr.actions(c.link) && w.cs.length > 1" class="ha" :disabled="!!pr.rerunning" :title="`Rerun ${c.name} only`" :aria-label="`Rerun ${c.name}`" @click="pr.rerun(c.link, c.name)"><Icon name="rerun" :size="12" /></button>
            <span class="rmeta">{{ word(c.bucket) }}</span>
          </div>
        </template>
      </section>
    </div>
  </div>
</template>

<style scoped>
  .prv {
    container-type: inline-size;
    min-width: 0;
    font-size: var(--fs-13);
  }
  /* the center view scrolls on its own; the PR tab's content already does */
  .prv.page {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 18px 24px 32px;
  }
  .inner {
    display: flex;
    flex-direction: column;
    gap: 18px;
    max-width: 860px;
    margin: 0 auto;
  }
  .prv svg {
    flex: none;
  }
  button {
    font: inherit;
    cursor: pointer;
  }
  button:disabled {
    cursor: default;
    opacity: 0.4;
  }
  .grow {
    flex: 1;
  }
  .mono {
    font-family: var(--mono);
  }
  .c-pass {
    color: var(--done);
  }
  .c-run {
    color: var(--running);
  }
  .c-warn {
    color: var(--waiting);
  }

  /* like the TODOs tab's head */
  .head {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    padding: 0 0 8px;
    border-bottom: 1px solid var(--border);
    white-space: nowrap;
  }
  .num {
    font-weight: 600;
    font-size: var(--fs-14);
  }
  .branch {
    /* takes only the room left, so it is the one cut short */
    flex: 1 1 0;
    max-width: max-content;
    min-width: 0;
    margin-right: auto;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--faint);
    font-family: var(--mono);
    font-size: var(--fs-12);
  }

  section {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .sh {
    display: flex;
    align-items: center;
    gap: 8px;
    height: 26px;
    margin-bottom: 2px;
    white-space: nowrap;
  }
  .st {
    color: var(--muted);
    font-size: var(--fs-12);
    font-weight: 600;
  }
  .cnt,
  .hint {
    color: var(--faint);
    font-size: var(--fs-12);
  }
  .hint {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    min-width: 0;
    overflow: hidden;
  }
  .empty {
    margin: 0;
    color: var(--muted);
    font-size: var(--fs-12-5);
  }

  /* ghost buttons, the name dropped when narrow */
  .tb,
  .ib,
  .ha {
    flex: none;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--muted);
  }
  .tb {
    gap: 5px;
    height: 24px;
    padding: 0 7px;
    font-size: var(--fs-12);
  }
  .ib {
    width: 24px;
    height: 24px;
    padding: 0;
  }
  .tb:hover:not(:disabled),
  .ib:hover:not(:disabled),
  .ha:hover:not(:disabled) {
    background: var(--hover);
    color: var(--text);
  }
  .tb.fix {
    color: var(--text);
  }
  .tb.fix:hover:not(:disabled) {
    color: var(--accent);
  }
  .sh .tb:last-child {
    margin-right: -7px;
  }
  .head .ib:last-child {
    margin-right: -4px;
  }

  /* a comment, like a todo */
  .item {
    display: flex;
    align-items: flex-start;
    gap: 9px;
    margin: 0 -6px;
    padding: 5px 6px;
    border-radius: 5px;
    line-height: 1.45;
  }
  .item:hover {
    background: var(--hover);
  }
  .box {
    flex: none;
    display: grid;
    place-items: center;
    width: 15px;
    height: 15px;
    margin-top: 2px;
    padding: 0;
    border: 1.5px solid var(--faint);
    border-radius: 4px;
    background: transparent;
    color: var(--button-text);
  }
  .box.on {
    border-color: var(--accent);
    background: var(--accent);
  }
  .box.done {
    border-color: var(--done-dot);
    background: var(--done-dot);
  }
  .txt {
    flex: 1;
    min-width: 0;
  }
  .body {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    overflow-wrap: anywhere;
  }
  .off .body {
    color: var(--muted);
  }
  .meta {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    column-gap: 6px;
    margin-top: 1px;
    color: var(--faint);
    font-size: var(--fs-12);
  }
  .meta > :not(:first-child)::before {
    content: '·';
    margin-right: 6px;
    color: var(--faint);
  }
  .meta .mono {
    font-size: var(--fs-11-5);
    color: var(--muted);
  }
  .rej {
    padding: 0;
    border: 0;
    background: none;
    color: var(--approval);
    font-size: inherit;
  }
  .rej:hover {
    text-decoration: underline;
  }
  .ha {
    width: 20px;
    height: 20px;
    padding: 0;
    visibility: hidden;
  }
  .item:is(:hover, :focus-within) .ha,
  .row:is(:hover, :focus-within) .ha {
    visibility: visible;
  }
  .more {
    align-self: flex-start;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin: 4px 0 2px -2px;
    padding: 2px 4px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--faint);
    font-size: var(--fs-12);
  }
  .more:hover {
    color: var(--text);
  }

  .note {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 2px 0 6px;
    color: var(--muted);
    font-size: var(--fs-12-5);
    line-height: 1.45;
  }
  .note > svg {
    margin-top: 2px;
  }
  .ntext {
    flex: 1;
    min-width: 0;
  }
  .note .tb {
    margin: -3px -7px 0 0;
  }

  /* a check, like a changed file */
  .row {
    display: flex;
    align-items: center;
    gap: 6px;
    height: 26px;
    margin: 0 -6px;
    padding: 0 6px;
    border-radius: 4px;
  }
  .row:hover {
    background: var(--hover);
  }
  .rname {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 9px;
    height: 100%;
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--text);
    text-align: left;
  }
  .nm {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sd {
    flex: none;
    width: 7px;
    height: 7px;
    margin: 0 4px;
    border-radius: 50%;
  }
  .d-fail { background: var(--approval-dot); }
  .d-run { background: var(--running-dot); }
  .d-pass { background: var(--done-dot); }
  .d-skip { background: var(--faint); }
  .rmeta {
    flex: none;
    color: var(--faint);
    font-size: var(--fs-12);
  }
  .row.sub {
    height: 24px;
    padding-left: 24px;
  }
  .row.sub .rname {
    color: var(--muted);
    font-size: var(--fs-12-5);
  }
  .row.sub .sd {
    width: 5px;
    height: 5px;
    margin: 0 5px;
  }

  @container (max-width: 420px) {
    .lbl {
      display: none;
    }
    .tb {
      padding: 0 5px;
    }
    .sh .tb:last-child {
      margin-right: -5px;
    }
  }
</style>
