# Campaign Workflows

Shared reference for **campaign workflows**: coordinated, time-bounded pushes with measurable outcomes, including **KPI workflows** (measure and improve a metric over time).

## Design principles

Treat campaigns as a **workflow design pattern**, not a separate feature.

### Minimum viable campaign spec

1. **Goal**: What is the measurable success criteria (metric, source, target, deadline)?
2. **Cadence**: How often should it run? Should it allow `workflow_dispatch` for manual control?
3. **Stop condition**: What does "goal met" look like, and what should the workflow do when it’s met (report + stop early)?
4. **Outputs**: What should be written (comment, issue, PR) vs only reported (stdout/stderr)?
5. **Scope**: Single repo or cross-repo? If cross-repo, who owns coordination and auth?
6. **Constraints**: Budget/time/quality constraints per run (max PRs, max issues, runtime limit, etc.)

### Composable building blocks (start simple)

- **Agentic (default)**: Use when work requires judgment, synthesis, or ambiguous decisions.
- **Deterministic core (opt-out)**: Use when tasks are precise, repeatable, and easy to validate.
- **Hybrid**: Keep deterministic prep in `steps:` and use an agentic prompt for decisions/edge cases.
- **Metrics + memory**: Use `cache-memory` (and optionally `repo-memory`) to persist goal tracking and state across runs.

### Pacing levers (how to avoid overwhelming humans and systems)

Use the **minimum levers** for safe throughput:

- **Cadence**: Prefer fuzzy `schedule:` (and weekdays for daily) to spread runs.
- **No overlap**: Use workflow-level `concurrency:` so only one campaign run is active at a time.
- **Global throughput**: If you have multiple campaigns, consider using the **same `concurrency.group`** across them.
- **Hard deadline**: Use `on.stop-after` when the campaign should stop after a date/time or relative window.
- **Output caps**: Enforce “how much writing” with `safe-outputs.*.max` (e.g., max 1 PR per run; max 1–3 comments per run).
- **Rate limiting**: For large scopes, use round-robin + cache-memory (one component per run).
- **Goal-aware early exit**: Add a deterministic pre-check and stop immediately when the goal is already met.

**Minimal pacing example (pick what you need):**

```yaml
---
on:
  schedule: weekly
  stop-after: "+30d"

concurrency:
  group: "campaign-weekly-ci-kpi"
  cancel-in-progress: false

permissions:
  content: read
  issues: read
tools:
  cache-memory: true

safe-outputs:
  create-pull-request:
    max: 1
  add-comment:
    max: 1
  noop:
---
```

### Goal-aware early exit (deterministic pre-check + agentic report)

Use a deterministic pre-check step when possible. If the goal is already met, exit early and still report outcomes.

```markdown
---
on:
  workflow_dispatch:
permissions: read-all
tools:
  cache-memory: true
steps:
  - name: Precompute goal status
    run: |
      # Replace with real metric computation.
      # Keep this step deterministic so it’s easy to validate.
      echo '{"goal_met": true, "metric": "coverage", "value": 82, "target": 80}' > /tmp/gh-aw/agent/goal_status.json
safe-outputs:
  add-comment:
    max: 1
  noop:
---

# Goal-aware run

Read `/tmp/gh-aw/agent/goal_status.json`.

If `goal_met` is true:

- Post a short summary (3–5 bullets) and stop.

Otherwise:

- Proceed with the plan, then end with a summary and learnings.
```

### KPI workflows (measure + improve)

KPI workflows are campaigns where the first-class output is a **metric** and an **interpretation**.

**Strong default:** Make KPI computation deterministic and easy to validate.

- Compute a KPI in `steps:` and write a small JSON payload (e.g., `/tmp/gh-aw/agent/kpi.json`).
- The agent reads that JSON, decides what to do (report-only vs follow-up), and always ends with a short summary.

**Inputs (when you need knobs):**

- Use `workflow_dispatch` inputs for user-controlled parameters (e.g., target threshold, window size) and have a deterministic `steps:` block normalize those inputs into a JSON config the agent reads.
- Use `safe-inputs:` when the agent needs a constrained, auditable tool to fetch privileged data (it’s not a human input mechanism).

**Minimum viable KPI spec (keep it explicit):**

- `kpi.name` + `kpi.definition` (formula)
- `kpi.source` (command, GitHub API read, file parse)
- `kpi.target` (threshold + timeframe)
- `kpi.scope` (branch, directory, package set)
- `kpi.publish_to` (comment/issue/discussion) + “update existing?”

**Standard deterministic payload (suggested):**

```json
{
  "kpi": "ci_success_rate",
  "value": 0.92,
  "target": 0.95,
  "window": "last_30_runs",
  "goal_met": false,
  "notes": "2 failures were flaky tests"
}
```

### Cross-repo coordination (advanced; keep it explicit)

- `safe-outputs.dispatch-workflow` is same-repo only.
- For org-wide or multi-org campaigns, use a coordinator that sends `repository_dispatch` to each target repo.
  - This requires a PAT or GitHub App token with access to every repo it dispatches.
  - Prefer a fine-grained PAT scoped to the specific repos with `Actions: Read & Write`.
  - Treat this as a privileged operation: keep permissions minimal and lock down inputs.
