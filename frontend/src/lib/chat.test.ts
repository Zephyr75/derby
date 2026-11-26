// Run with `npm test` (Node's own test runner: no extra dependency).
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { commandFailure, commandLabel } from './chat.ts'

test('a command is labelled by its description, else by a sentence made from it', () => {
  assert.equal(commandLabel({ description: 'Run the tests', command: 'go test ./...' }), 'Run the tests')
  assert.equal(commandLabel({ command: 'go test ./internal/unlock/...' }), 'Run the Go tests')
  assert.equal(commandLabel({ command: 'cd /code/pony-core && ENV=test make test-path TEST_PATH=tests/x.py 2>&1 | tail -40' }), 'Run make test-path')
  assert.equal(commandLabel({ command: 'git diff origin/dev...HEAD --stat | sort | head -20' }), 'Show the diff')
  assert.equal(commandLabel({ command: "cat > /tmp/repro.py <<'EOF'\nimport os\nEOF" }), 'Write repro.py')
  assert.equal(commandLabel({ command: 'sed -n 1,40p internal/app/app.go' }), 'Read app.go')
  assert.equal(commandLabel({ command: 'S=/tmp/x; ls -la $S' }), 'List files')
  assert.equal(commandLabel({ command: 'terraform plan -out=x' }), 'Run terraform plan')
  assert.equal(commandLabel({}), 'Run a command')
})

test('a failed command is tagged with how many things failed', () => {
  assert.equal(commandFailure('FAILED tests/x.py::test_a - assert 0 == 3\n=== 1 failed, 11 passed in 4.82s ==='), '1 failed')
  assert.equal(commandFailure('--- FAIL: TestA (0.00s)\n--- FAIL: TestB (0.00s)\nFAIL'), '2 failed')
  assert.equal(commandFailure('npm ERR! missing script'), 'failed')
  assert.equal(commandFailure(''), 'failed')
})
