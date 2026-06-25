# Campaign Workflows

Coordinated, time-bounded pushes with measurable outcomes, including **KPI workflows** (measure and improve a metric over time).

## Design principles

### Minimum viable campaign spec

1. **Goal**: measurable criteria (metric, source, target, deadline).
2. **Cadence**: schedule + optional `workflow_dispatch`.
3. **Stop condition**: define "goal met"; report + stop early.
4. **Outputs**: comment, issue, PR vs stdout/stderr only.
5. **Scope**: single-repo or cross-repo.
6. **Constraints**: per-run caps (max PRs, issues, runtime).

### Composable building blocks

- **Agentic (default)**: judgment, synthesis, ambiguous decisions.
- **Deterministic core**: precise, repeatable, validatable.
- **Hybrid**: deterministic prep in `steps:`, agentic prompt for decisions.
- **Metrics + memory**: `cache-memory` (optionally `repo-memory`) for cross-run goal tracking.

### Pacing levers

- **Cadence**: prefer fuzzy `schedule:` (weekdays for daily) to spread runs.
- **No overlap**: workflow-level `concurrency:`.
- **Global throughput**: share `concurrency.group` across campaigns.
- **Hard deadline**: `on.stop-after` for date/time or relative window.
- **Output caps**: `safe-outputs.*.max` (e.g., max 1 PR per run; max 1–3 comments).
- **Rate limiting**: round-robin + cache-memory (one component per run) for large scopes.
- **Goal-aware early exit**: deterministic pre-check, stop when goal met.

**Minimal pacing example:**

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

### Goal-aware early exit

Deterministic pre-check; exit early when goal met, still report.

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
      echo '{"goal_met": true, "metric": "coverage", "value": 82, "target": 80}' > /tmp/gh-aw/agent/goal_status.json
safe-outputs:
  add-comment:
    max: 1
  noop:
---

# Goal-aware run

Read `/tmp/gh-aw/agent/goal_status.json`.

If `goal_met` is true: post a short summary (3–5 bullets) and stop.

Otherwise: proceed with the plan, then end with a summary and learnings.
```

### KPI workflows (measure + improve)

Output a **metric** and **interpretation**. Make KPI computation deterministic.

- Compute KPI in `steps:`, write JSON (e.g., `/tmp/gh-aw/agent/kpi.json`).
- Agent reads JSON, decides report-only vs follow-up, ends with short summary.

**Inputs:**

- `workflow_dispatch` inputs for user parameters; normalize via `steps:` into JSON.
- `mcp-scripts:` for constrained, auditable access to privileged data (not human input).

**Minimum viable KPI spec:**

- `kpi.name` + `kpi.definition` (formula)
- `kpi.source` (command, GitHub API read, file parse)
- `kpi.target` (threshold + timeframe)
- `kpi.scope` (branch, directory, package set)
- `kpi.publish_to` (comment/issue/discussion) + "update existing?"

**Standard deterministic payload:**

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

### Cross-repo coordination

- `safe-outputs.dispatch-workflow` is same-repo by default; cross-repo needs `target-repo` plus `allowed-repos` allowlist and a token with `actions: write` on the target.
- For org-wide/multi-org, use a coordinator sending `repository_dispatch` to each target.
  - Requires PAT or GitHub App token with access to every dispatched repo.
  - Prefer fine-grained PAT scoped to specific repos with `Actions: Read & Write`.
  - Keep permissions minimal, lock down inputs.
