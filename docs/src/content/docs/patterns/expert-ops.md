---
title: ExpertOps
description: Scheduled domain-expert workflows that examine a product and file improvement suggestions as issues, with a feedback loop to observe the effect of previous changes
sidebar:
  badge: { text: 'Scheduled', variant: 'tip' }
---

ExpertOps uses a scheduled workflow as a focused domain expert — for example, an OpenTelemetry expert or an A/B testing expert — to continuously examine a product and file targeted improvement suggestions as GitHub issues. Rather than making all improvements at once, the expert surfaces one or a few actionable items per run so changes remain easy to review, merge, and observe.

The pattern works best when the expert can close the loop: it reads live data from its domain (telemetry traces, experiment results, coverage reports) before deciding what to suggest next.

## The ExpertOps Pattern

### Narrow domain focus

An ExpertOps workflow is not a general-purpose improver. It covers a single, well-defined concern — instrumentation, accessibility, security headers, performance budgets — and ignores everything else. Breadth is traded for depth.

### Scheduled expert runs

The expert runs on a fixed schedule (typically daily or weekly) so improvements compound gradually:

```aw wrap
---
on:
  schedule:
    - cron: "0 7 * * 1-5"  # Weekdays, 07:00 UTC
  workflow_dispatch:
---
```

`workflow_dispatch` keeps manual testing easy without requiring a schedule change.

### Observation before action

Before proposing changes, the expert reads live state from its domain. This observation step is what separates ExpertOps from a simple linting or static-analysis workflow — the agent sees the *current runtime behavior*, not just the code:

```aw wrap
---
steps:
  - name: Export recent telemetry summary
    run: |
      # Fetch last 24 h of spans from your observability backend
      curl -s "$OTEL_BACKEND/api/traces?limit=500&lookback=24h" \
        > /tmp/gh-aw/traces.json
      echo "Fetched $(jq 'length' /tmp/gh-aw/traces.json) spans"

  - name: Check open expert issues
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      gh issue list \
        --label otel-expert \
        --state open \
        --json number,title,body \
        > /tmp/gh-aw/open-issues.json
      echo "$(jq 'length' /tmp/gh-aw/open-issues.json) open issues"
---
```

### Feedback loop

The expert reads its own previous suggestions (open issues labelled with a domain tag) before creating new ones. This prevents duplicate recommendations and lets the agent evaluate whether earlier suggestions have already been acted upon:

```aw wrap
---
steps:
  - name: Check merged PRs referencing expert issues
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      gh pr list \
        --state merged \
        --label otel-expert \
        --limit 20 \
        --json number,title,mergedAt \
        > /tmp/gh-aw/merged-prs.json
---

# OpenTelemetry Expert

You are an OpenTelemetry expert reviewing the instrumentation of this service.

## Observation data

- Live traces (last 24 h): `/tmp/gh-aw/traces.json`
- Your open suggestions: `/tmp/gh-aw/open-issues.json`
- Recently merged improvements: `/tmp/gh-aw/merged-prs.json`

## Your task

1. Identify the single most impactful instrumentation gap (missing span, wrong
   cardinality, absent error attribute, etc.).
2. Check `/tmp/gh-aw/open-issues.json` — do not file a duplicate.
3. If a recent merged PR addressed a previous suggestion, note the outcome.
4. File **one** focused issue with a clear title, problem description, and
   suggested fix. Use label `otel-expert`.
```

Keeping the suggestion count to one per run (or at most a handful) keeps the backlog manageable and prevents the team from feeling overwhelmed.

### Labelling expert issues

All issues created by the expert should carry a consistent label so they can be tracked and filtered. Configure this in `safe-outputs`:

```aw wrap
---
safe-outputs:
  create-issue:
    title-prefix: "[otel] "
    labels: [otel-expert, instrumentation]
    max: 2
---
```

## Example: OpenTelemetry Expert

A complete OpenTelemetry expert workflow that detects missing spans and bad attribute cardinality:

````aw wrap
---
name: OpenTelemetry Expert
description: Daily expert review of service instrumentation

on:
  schedule:
    - cron: "0 7 * * 1-5"
  workflow_dispatch:

engine: copilot
strict: true

permissions:
  contents: read
  issues: write

network:
  allowed:
    - defaults
    - github
    - otel.example.internal   # internal observability backend

safe-outputs:
  create-issue:
    title-prefix: "[otel] "
    labels: [otel-expert, instrumentation]
    max: 2

tools:
  bash: ["*"]
  github:
    toolsets: [default]

steps:
  - name: Fetch recent traces
    run: |
      mkdir -p /tmp/gh-aw/otel
      curl -s "https://otel.example.internal/api/traces?limit=500&lookback=24h" \
        > /tmp/gh-aw/otel/traces.json

  - name: Fetch open expert issues
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      gh issue list \
        --repo "${{ github.repository }}" \
        --label otel-expert \
        --state open \
        --json number,title \
        > /tmp/gh-aw/otel/open-issues.json

timeout-minutes: 15
---

# OpenTelemetry Expert

You are a senior OpenTelemetry engineer reviewing the instrumentation of
`${{ github.repository }}`.

## Data available

- Trace sample (last 24 h): `/tmp/gh-aw/otel/traces.json`
- Your open improvement suggestions: `/tmp/gh-aw/otel/open-issues.json`

## What to look for

- Operations without spans (HTTP handlers, DB queries, background jobs)
- Spans missing `error` attribute on failures
- High-cardinality span names (e.g. including user IDs or request paths)
- Missing service or component attributes
- Traces that never complete (orphaned spans)

## Your task

Identify the single most important gap. Check open issues to avoid duplicates.
File one concise issue describing the problem and a suggested fix.
````

## Example: A/B Testing Expert

An expert that queries live experiment results and suggests the next testing opportunity:

````aw wrap
---
name: A/B Testing Expert
description: Weekly review of experiment coverage and opportunities

on:
  schedule:
    - cron: "0 9 * * 1"   # Monday mornings
  workflow_dispatch:

engine: copilot
strict: true

permissions:
  contents: read
  issues: write

network:
  allowed:
    - defaults
    - github
    - experiments.example.internal

safe-outputs:
  create-issue:
    title-prefix: "[ab] "
    labels: [ab-expert, experimentation]
    max: 2

tools:
  bash: ["*"]

steps:
  - name: Fetch active and recent experiments
    run: |
      mkdir -p /tmp/gh-aw/ab
      curl -s "https://experiments.example.internal/api/experiments?state=all&limit=50" \
        > /tmp/gh-aw/ab/experiments.json

  - name: Fetch open AB expert issues
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      gh issue list \
        --repo "${{ github.repository }}" \
        --label ab-expert \
        --state open \
        --json number,title \
        > /tmp/gh-aw/ab/open-issues.json

timeout-minutes: 10
---

# A/B Testing Expert

You are an experimentation engineer reviewing the product's experiment coverage.

## Data available

- Active and recent experiments: `/tmp/gh-aw/ab/experiments.json`
- Your open suggestions: `/tmp/gh-aw/ab/open-issues.json`

## What to look for

- Features shipped without an experiment (missed learning opportunity)
- Experiments running too long without a decision
- Overlapping experiment assignments that confound results
- Metrics missing from experiment definitions
- UI surfaces with no experiment history (untested assumptions)

## Your task

Identify the highest-value next experimentation opportunity. Check open issues
to avoid duplicates. File one focused issue describing what to test, why it
matters, and what success looks like.
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
- **[Monitoring](/gh-aw/patterns/monitoring/)** — Track workflow activity with GitHub Projects
