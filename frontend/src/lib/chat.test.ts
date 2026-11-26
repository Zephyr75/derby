// Run with `npm test` (Node's own test runner: no extra dependency).
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { commandLabel, commandResult } from './chat.ts'

test('a command is labelled by its description, else by what it runs', () => {
  assert.equal(commandLabel({ description: 'Run the tests', command: 'go test ./...' }), 'Run the tests')
  assert.equal(commandLabel({ command: 'cd /code/pony-core && ENV=test make test-path TEST_PATH=tests/x.py 2>&1 | tail -40' }), 'make test-path TEST_PATH=tests/x.py 2>&1')
  assert.equal(commandLabel({ command: "git diff origin/dev...HEAD --stat | sort | head -20" }), 'git diff origin/dev...HEAD --stat')
  assert.equal(commandLabel({ command: "cat > /tmp/repro.py <<'EOF'\nimport os\nEOF" }), 'cat > /tmp/repro.py')
  assert.equal(commandLabel({}), 'Run Command')
})

test('the result line sums up the output', () => {
  assert.equal(commandResult('', false), 'Done, no output')
  assert.equal(commandResult('ok  \tjockey/a\t0.1s\nok  \tjockey/b\t0.2s\n', false), '2 packages ok')
  assert.equal(commandResult('FAILED tests/x.py::test_a - assert 0 == 3\n=== 1 failed, 11 passed in 4.82s ===', true), '1 failed, 11 passed in 4.82s · FAILED tests/x.py::test_a - assert 0 == 3')
  assert.equal(commandResult('=== 12 passed in 4.82s ===', false), '12 passed in 4.82s')
  assert.equal(commandResult(' a.go | 2 +-\n 3 files changed, 9 insertions(+), 2 deletions(-)', false), '3 files changed, 9 insertions(+), 2 deletions(-)')
  assert.equal(commandResult('FAIL jockey/a 0.1s\nsomething', true), 'FAIL jockey/a 0.1s')
  assert.equal(commandResult('one\ntwo\nthree', false), 'one · 3 lines')
})
