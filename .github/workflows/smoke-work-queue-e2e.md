---
private: true
emoji: "🧪"
name: Smoke Work Queue E2E
description: Exercise work submission, native claim dispatch, and claim completion across separate workflows
intent: Detect end-to-end work queue regressions without generating an unbounded chain of work.
on:
  schedule: daily
  workflow_dispatch:
if: github.run_attempt == 1
permissions:
  contents: read
  actions: read
  copilot-requests: write
engine: codex
model: copilot/auto
checkout: false
network:
  allowed: []
tools:
  github: false
  work-queue: true
  cli-proxy: true
safe-outputs:
  threat-detection: false
  dispatch-workflow:
    workflows: [smoke-work-queue-e2e-worker]
    max: 1
jobs:
  verify_smoke_dispatch:
    needs: [safe_outputs]
    if: always()
    runs-on: ubuntu-slim
    timeout-minutes: 3
    permissions: {}
    steps:
      - name: Verify native smoke dispatch
        env:
          SAFE_OUTPUTS_RESULT: ${{ needs.safe_outputs.result }}
          WORK_QUEUE_REQUESTS: ${{ needs.safe_outputs.outputs.work_queue_requests }}
        run: |
          node <<'NODE'
          const assert = require('node:assert/strict');
          assert.equal(process.env.SAFE_OUTPUTS_RESULT, 'success');
          const result = JSON.parse(process.env.WORK_QUEUE_REQUESTS);
          assert.equal(result.version, 3);
          assert.equal(result.status, 'ok');
          assert.equal(result.success, true);
          const launches = result.receipts.flatMap(receipt => receipt.launches || []);
          assert.equal(launches.length, 1, 'Smoke test requires exactly one native worker dispatch');
          assert.equal(launches[0].state, 'bound', 'Smoke worker must be bound to its native run');
          assert.equal(launches[0].dispatched, true);
          NODE
timeout-minutes: 10
strict: true
---

# Work Queue End-to-End Smoke Dispatcher

Use only the advertised work-queue tools (or their mounted CLI wrappers).
Call `work_queue_read` once to exercise the activation snapshot.

Call `work_queue_submit` once with exactly one node:

```json
{"nodes":[{"graph_id":"smoke-work-queue-e2e:${{ github.run_id }}","node_key":"smoke","pool":"default","worker_profile":"smoke-work-queue-e2e-worker","payload":{"task":"Consume this smoke-test Claim without creating more Work.","effect_contract":{"kind":"none"}}}]}
```

Then call `work_queue_dispatch_next` once with
`{"pool":"default","max_claims":1,"max_dispatches":1}`. Let the trusted scheduler
select and launch the separate worker workflow; do not call `dispatch_workflow` directly.
Report only that the intents were staged, not that a Claim or Result was verified.
Surface any tool failure and stop; do not emit `noop` or retry uncertain writes.

The trusted verification job requires exactly one bound native worker dispatch;
an empty grant or staged preview is not smoke success. Worker Claim completion is
verified separately by the worker workflow, not by this agent.
