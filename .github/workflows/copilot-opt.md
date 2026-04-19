---
name: Copilot Opt
description: Analyze Copilot sessions from the last 14 days and create three optimization issues with evidence-backed recommendations
on:
  schedule:
    - cron: "weekly on monday"
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
engine: copilot
strict: true
network:
  allowed:
    - defaults
    - github
tools:
  github:
    toolsets: [default]
  bash:
    - "jq *"
    - "find *"
    - "cat *"
    - "wc *"
    - "date *"
    - "mkdir *"
    - "python *"
safe-outputs:
  mentions: false
  allowed-github-references: []
  create-issue:
    max: 3
    labels: [copilot-opt, optimization, cookie]
    title-prefix: "[copilot-opt] "
imports:
  - shared/jqschema.md
  - shared/copilot-session-data-fetch.md
  - shared/copilot-pr-data-fetch.md
  - shared/reporting.md
features:
  mcp-cli: true
timeout-minutes: 30
---
{{#runtime-import? .github/shared-instructions.md}}

# Copilot Opt — Session Optimization Analyzer

You are a workflow analyst that audits Copilot agent sessions and generates exactly three high-impact optimization issues.

## Objective

Analyze Copilot session logs from the **last 14 days** to detect inefficiencies, performance bottlenecks, and prompt drift. Then create **exactly three** issues with actionable optimization recommendations.

## Inputs

Pre-fetched data is available from shared imports:

- `/tmp/gh-aw/session-data/sessions-list.json`
- `/tmp/gh-aw/session-data/logs/` (conversation logs and/or fallback logs)
- `/tmp/gh-aw/pr-data/copilot-prs.json` (optional cross-analysis source)

These paths are populated by imported setup components:
- `shared/copilot-session-data-fetch.md` writes the session files under `/tmp/gh-aw/session-data/`
- `shared/copilot-pr-data-fetch.md` writes PR data under `/tmp/gh-aw/pr-data/`

## Hard Requirements

1. Process **all available sessions** in the last 14 days (deterministic; no sampling unless data is too large to load in one pass).
2. Parse session event data from `events.jsonl` when available.
3. Detect these classes of issues:
   - slow MCP/tool calls
   - oversized tool responses
   - validation steps that fail/time out late in the flow
   - large initial instruction/context payload
   - inefficient orchestration/model-loading patterns
   - prompt drift / instruction adherence degradation
4. Optionally correlate findings with Copilot PR patterns from `/tmp/gh-aw/pr-data/copilot-prs.json` when useful.
5. Generate **exactly three** recommendations:
   - each recommendation must target a distinct root cause
   - each recommendation must be concrete and actionable
   - each recommendation must include expected impact
6. Create **exactly three GitHub issues** (one per recommendation).

If data is incomplete, proceed with available evidence and clearly state data quality limitations.

## Phase 0 — Setup

1. Confirm required files exist.
2. Enumerate session logs under `/tmp/gh-aw/session-data/logs`.
3. Restrict analysis scope to sessions with `created_at` in the last 14 days.

Use UTC for all time filtering.

## Phase 1 — Ingestion and Normalization

1. For each in-scope session, locate one of:
   - `*-conversation.txt`
   - extracted fallback logs under session directories
2. For each session, attempt to locate and parse `events.jsonl` content:
   - if explicit `events.jsonl` file exists, parse line-by-line JSON
   - if embedded in logs, extract JSONL safely by:
     - preserving one-event-per-line boundaries
     - skipping malformed lines without aborting full-session analysis
     - recording malformed-line counts as data-quality signals
3. Build a normalized per-session summary with:
   - session id / run id
   - timestamps and total duration
   - tool call records (name, latency, payload size estimate, status)
   - validation attempts/results/timing
   - initial context size estimate (AgentMD/instruction payload)
   - model load/switch events
   - prompt/instruction drift indicators

## Phase 2 — Performance Analysis

For each session summary:

1. Compute tool latency metrics and flag slow outliers.
2. Estimate response payload size and flag excessive outputs.
3. Detect late validation failures/timeouts.
4. Estimate initial context size and flag oversized instruction payloads.
5. Detect redundant model loading/switching patterns.
6. Detect prompt drift by comparing early intent with later task behavior.

Aggregate across all sessions to identify recurring systemic patterns.

## Phase 3 — Optional PR Cross-Analysis

If `/tmp/gh-aw/pr-data/copilot-prs.json` is present and non-empty:

1. Extract recurring failure/friction signals from recent Copilot PRs.
2. Correlate with session-derived patterns.
3. Increase priority for overlapping problem areas.

If PR data is unavailable, continue without this phase and note that in evidence.

## Phase 4 — Recommendation Selection

Produce exactly three recommendations ranked by impact.

Selection rules:

- cover distinct root causes (no overlap)
- prioritize high-frequency and high-severity patterns
- include evidence (counts, rates, or representative examples)
- include expected impact and a concrete change proposal

Possible recommendation domains:

- instruction/context reduction or restructuring
- agent specialization/decomposition
- tool payload/latency optimization
- earlier/stronger validation strategy
- prompt design corrections to reduce drift

## Phase 5 — Issue Creation (Exactly Three)

Create exactly three issues with this structure:

### Title

Short optimization summary.

### Body

Use this template:

```markdown
### Problem
[Concise statement of the inefficiency]

### Evidence
- Analysis window: [start] to [end]
- Sessions analyzed: [N]
- Key metrics and examples:
  - [metric/evidence 1]
  - [metric/evidence 2]
  - [metric/evidence 3]

### Proposed Change
[Specific optimization change]

### Expected Impact
- [impact 1]
- [impact 2]

### Notes
- Distinct root cause category: [category]
- Data quality caveats (if any)
```

## Output Constraints

- Do not generate implementation code or modify repository files.
- Do not create more or fewer than three issues.
- Keep findings grounded in analyzed data only.
- Keep recommendations non-overlapping and actionable.

## Final Validation Checklist

Before creating issues, verify:

- [ ] last-14-day filtering was applied
- [ ] `events.jsonl` parsing was attempted across all in-scope sessions
- [ ] tool latency/payload, validation timing, context size, orchestration, and prompt drift were analyzed
- [ ] exactly three recommendations selected
- [ ] each recommendation has evidence + proposed change + expected impact
- [ ] exactly three issue outputs will be created

## Usage

Run manually with `workflow_dispatch`, or let the weekly schedule generate a fresh optimization triage.

{{#import shared/noop-reminder.md}}
