---
name: Smoke Work Queue Dispatcher
description: Submit one synthetic task daily and request one native smoke worker Claim
intent: Detect failures in native work queue admission, dispatch, and worker completion without changing repository content.
on:
  schedule: daily around 14:00
if: github.run_attempt == 1
imports:
  - shared/work-queue-staged-plan.md
permissions:
  contents: read
  copilot-requests: write
engine: copilot
network:
  allowed: []
tools:
  work-queue: true
  cli-proxy: true
  bash:
    - cat /tmp/gh-aw/agent/smoke-work-queue-plan.json
safe-outputs:
  dispatch-workflow:
    workflows: [smoke-work-queue-worker]
    target-ref: ${{ github.event.repository.default_branch }}
    max: 1
steps:
  - name: Prepare synthetic work and bounded dispatch
    uses: actions/github-script@v9.0.0
    with:
      script: |
        const fs = require("node:fs");
        const path = require("node:path");
        const { stagePlanIntents } = require(path.join(process.env.RUNNER_TEMP, "gh-aw/actions/work_queue_intents.cjs"));
        const plan = {
          version: 1,
          nodes: [{
            graph_id: `smoke-work-queue:${context.runId}`,
            node_key: "smoke",
            pool: "smoke-work-queue",
            worker_profile: "smoke-work-queue-worker",
            payload: {
              task: "Complete the synthetic work queue smoke task.",
              smoke: "work-queue-v1",
              resource_scope: {
                version: 1,
                resources: [{
                  host: "github.com",
                  repository: `${context.repo.owner}/${context.repo.repo}`,
                  repository_id: String(context.payload.repository.id),
                }],
              },
              effect_contract: { kind: "none" },
            },
            depends_on: [],
          }],
          dispatch: { pool: "smoke-work-queue", max_claims: 1, max_dispatches: 1 },
        };
        fs.mkdirSync("/tmp/gh-aw/agent", { recursive: true });
        fs.writeFileSync("/tmp/gh-aw/agent/smoke-work-queue-plan.json", JSON.stringify(plan));
        stagePlanIntents(path.join(process.env.RUNNER_TEMP, "gh-aw/safeoutputs/work-queue/work-queue.intents.jsonl"), plan);
        core.info("Staged one synthetic Work and one bounded smoke dispatch request");
timeout-minutes: 10
strict: true
---

# Work Queue Dispatcher Smoke

Read `/tmp/gh-aw/agent/smoke-work-queue-plan.json`, then call `work_queue_read`
with `{"pool":"smoke-work-queue","limit":8}`. An absent queue is empty; the
trusted submission installs Policy and Work together. An existing policyless
ledger or a tool error is a failure, not an empty backlog.

The trusted preparation step stages exactly one synthetic Work followed by
`{"pool":"smoke-work-queue","max_claims":1,"max_dispatches":1}`. Explain the
staged plan; do not submit again, choose a winner, dispatch directly, or emit
`noop`. Work identity is stable for this daily activation, and whole-run reruns
are disabled. The scheduler may select an older smoke task when recovering a
backlog; never force-release capacity or modify other pools.

The assignment-only smoke worker checks the synthetic payload and finishes its
original Claim without resource effects. A staged request is not a launch, and
a staged finish is not a verified Result. Check the authoritative queue Result
and worker run when diagnosing failures; do not wait for workers in this run.

Publish both smoke workflows and their lock files together with `aw.json`.
The dedicated pool has concurrency one and still shares repository-wide limits.
An already-installed Policy needs an explicit, quiescent administrator Policy
update to adopt new scheduling pools; never reset or reseed an existing queue.
