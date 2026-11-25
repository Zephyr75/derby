// Run with `npm test` (Node's own test runner: no extra dependency).
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { compactTables, prSummary, summary } from './markdown.ts'

test('a report summary names its first section TLDR', () => {
  assert.deepEqual(summary('## TL;DR\n\nIt retries.\n\n## Files\n\n- a.go'), [{ label: 'TLDR', md: 'It retries.' }])
  assert.deepEqual(summary('**TLDR**: It retries.\n\nMore.'), [{ label: 'TLDR', md: 'It retries.' }])
})

test('a plan summary has its Options as small blocks, not a table', () => {
  const table = [
    '| Option | Pros | Cons | Size |',
    '|---|---|---|---|',
    '| A. In the client | One place | Each platform repeats it | +40 / -5 |',
    '| B. In the gateway | Old apps benefit | Hides failures | +120 / -10 |',
  ].join('\n')
  const plan = [
    '**TLDR**: Retry the handshake.',
    '## Options',
    table,
    'A is recommended.',
    '## Files',
    '- a.go',
  ].join('\n\n')
  const opts = summary(plan).find((p) => p.label === 'Options')!.md
  assert.ok(!opts.includes('|'), opts)
  assert.match(opts, /\*\*A\. In the client\*\* · \+40 \/ -5\n\n- \*\*Pros:\*\* One place\n- \*\*Cons:\*\* Each platform repeats it/)
  assert.match(opts, /\*\*B\. In the gateway\*\* · \+120 \/ -10/)
  assert.match(opts, /A is recommended\./)
})

test('compactTables leaves everything else as it is', () => {
  const md = 'Before\n\n- a | b\n- c\n\nAfter'
  assert.equal(compactTables(md), md)
})

test("the Copilot step's short report is the PR title and the start of its description", () => {
  const body = '# feat(core): retry #12\n\nFirst part.\n\nSecond part.\n\nCloses #12\n\n## Choices\n\nLong.\n\n## Flow\n\nLong too.'
  assert.deepEqual(prSummary(body), [
    { label: 'Title', md: 'feat(core): retry #12' },
    { label: 'Description', md: 'First part.\n\nSecond part.\n\nCloses #12' },
  ])
  assert.deepEqual(prSummary('# Only a title'), [{ label: 'Title', md: 'Only a title' }])
})
