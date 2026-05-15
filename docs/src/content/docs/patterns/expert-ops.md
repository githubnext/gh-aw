---
title: ExpertOps
description: Scheduled domain-expert workflows that examine a product and file improvement suggestions as issues, with a feedback loop to observe the effect of previous changes
sidebar:
  badge: { text: 'Scheduled', variant: 'tip' }
---

ExpertOps uses a scheduled workflow as a focused domain expert — for example, an OpenTelemetry expert or an A/B testing expert — to continuously examine a product and file targeted improvement suggestions as GitHub issues. Rather than making all improvements at once, the expert surfaces one actionable item per run so changes remain easy to review, merge, and observe.

The pattern works best when the expert can close the loop: it reads live data from its domain (telemetry traces, experiment results) before deciding what to suggest next.

## The ExpertOps Pattern

An ExpertOps workflow covers a single, well-defined concern and ignores everything else. Breadth is traded for depth.

Before proposing changes, the expert reads live state from its domain — the agent sees the *current runtime behavior*, not just the code. It also reads its own previously filed issues to avoid duplicates and observe whether earlier suggestions have been acted upon.

All issues created by the expert carry a consistent label configured in `safe-outputs`:

```aw wrap
---
on:
  schedule: daily
  workflow_dispatch:

steps:
  - name: Fetch domain data and open expert issues
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      curl -s "$BACKEND/api/data" > /tmp/gh-aw/data.json
      gh issue list --label domain-expert --state open --json number,title \
        > /tmp/gh-aw/open-issues.json

safe-outputs:
  create-issue:
    title-prefix: "[domain] "
    labels: [domain-expert]
    max: 2
---

# Domain Expert

Review `/tmp/gh-aw/data.json`. Check `/tmp/gh-aw/open-issues.json` to avoid duplicates.
File one focused issue for the most impactful problem found.
```

## Example: OpenTelemetry Expert

````aw wrap
---
name: OpenTelemetry Expert

on:
  schedule: daily
  workflow_dispatch:

engine: copilot
strict: true

network:
  allowed:
    - defaults
    - github
    - otel.example.internal

safe-outputs:
  create-issue:
    title-prefix: "[otel] "
    labels: [otel-expert]
    max: 2

tools:
  bash: ["*"]

steps:
  - name: Fetch traces and open issues
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      curl -s "https://otel.example.internal/api/traces?lookback=24h" \
        > /tmp/gh-aw/traces.json
      gh issue list --label otel-expert --state open --json number,title \
        > /tmp/gh-aw/open-issues.json
---

# OpenTelemetry Expert

Review the trace sample at `/tmp/gh-aw/traces.json` for instrumentation gaps
(missing spans, wrong cardinality, absent error attributes). Check open issues
at `/tmp/gh-aw/open-issues.json` to avoid duplicates. File one concise issue.
````

## Example: A/B Testing Expert

````aw wrap
---
name: A/B Testing Expert

on:
  schedule: weekly
  workflow_dispatch:

engine: copilot
strict: true

network:
  allowed:
    - defaults
    - github
    - experiments.example.internal

safe-outputs:
  create-issue:
    title-prefix: "[ab] "
    labels: [ab-expert]
    max: 2

tools:
  bash: ["*"]

steps:
  - name: Fetch experiments and open issues
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      curl -s "https://experiments.example.internal/api/experiments" \
        > /tmp/gh-aw/experiments.json
      gh issue list --label ab-expert --state open --json number,title \
        > /tmp/gh-aw/open-issues.json
---

# A/B Testing Expert

Review experiment coverage at `/tmp/gh-aw/experiments.json` (features shipped
without tests, experiments running too long, missing success metrics). Check
`/tmp/gh-aw/open-issues.json` to avoid duplicates. File one focused issue.
````

## Persistent memory across runs

Use `cache-memory` to let the expert accumulate knowledge between runs — for example, a growing list of patterns observed over time:

```aw wrap
---
tools:
  cache-memory: true
---

# Expert instructions

Load your observation history from `/tmp/gh-aw/cache-memory/` if it exists.
After filing issues, append a brief summary of today's findings to the history.
```

See [Cache Memory](/gh-aw/reference/cache-memory/) for configuration details.

## Design considerations

**Keep the backlog small.** File one or two issues per run. If the expert creates ten issues at once, the team loses the gradual improvement benefit and the backlog becomes noise.

**Use domain-specific labels.** Labels like `otel-expert` or `ab-expert` let the team filter, assign, and close suggestions as a coherent set.

**Connect to live data.** Static analysis catches obvious problems; live data reveals runtime surprises. The quality of ExpertOps suggestions is directly proportional to the richness of the observation step.

**Track impact.** Have the expert periodically review merged PRs that addressed its previous suggestions and note whether the problem was resolved. This creates an improvement feedback loop and helps the expert refine its heuristics over time.

## Related Patterns

- **[DailyOps](/gh-aw/patterns/daily-ops/)** — General scheduled improvement workflows
- **[DataOps](/gh-aw/patterns/data-ops/)** — Deterministic data collection followed by agentic analysis
- **[IssueOps](/gh-aw/patterns/issue-ops/)** — Trigger workflows from issue events
- **[Monitoring](/gh-aw/experimental/monitoring-with-projects/)** — Track workflow activity with GitHub Projects
