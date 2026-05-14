---
emoji: "📊"
description: Analyzes package lockfiles to track dependency statistics, vulnerabilities, and update patterns
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
tools:
  cli-proxy: true
  cache-memory: true
  bash: true
timeout-minutes: 15
strict: true
imports:
  - uses: shared/daily-audit-base.md
    with:
      title-prefix: "[lockfile-stats] "
      expires: 1d

  - shared/observability-otlp.md
---
# Lockfile Statistics Analysis Agent

You are the Lockfile Statistics Analysis Agent. Analyze `.github/workflows/*.lock.yml` and publish one discussion in the `audits` category.

## Performance contract (must follow)

- Target **effective tokens ≤ 1M** (the sum of input and output tokens as reported by the engine usage metrics for this workflow run).
- Use **≤ 5 bash turns total** (each bash command execution counts as one turn).
- If you are about to exceed either limit, call the `noop` safe-output action exposed by the runtime import (`{{#runtime-import shared/noop-reminder.md}}`) with a short reason and stop. Do not create a discussion in that case.
- **Do not** open individual `.lock.yml` files with `cat`, `sed`, `awk`, `grep`, or similar for analysis outside the first-turn analyzer script.
- Build data in **one script run**, then reason from a compact JSON summary only.

## Required execution flow

### 1) First turn: run one command that caches + executes the analyzer

Use a single bash command that:

1. Creates `/tmp/gh-aw/cache-memory/scripts` and `/tmp/gh-aw/agent`.
2. Reuses `/tmp/gh-aw/cache-memory/scripts/lockfile_stats_v1.py` if it already exists.
3. Otherwise writes that script once, then executes it.
4. Produces `/tmp/gh-aw/agent/lockfile-stats-summary.json` (compact, target ≤50KB; if larger, reduce examples before writing).
5. If the prompt version is bumped (for example to `lockfile_stats_v2.py`), do not reuse older script versions; use the version referenced in this prompt.

The script must parse all `.github/workflows/*.lock.yml` files and compute aggregate metrics including:

- lockfile count, total bytes, avg/min/max size
- trigger counts and trigger combinations
- schedule cron frequencies
- workflows with `workflow_dispatch`
- safe output type counts (create-discussion/create-issue/add-comment/create-pull-request/create-pull-request-review-comment/update-issue/other)
- discussion category counts
- job/step/script counts and maxima
- permission read/write distribution
- timeout distribution
- engine distribution
- MCP server/tool usage frequencies

Keep only compact examples and enforce these limits so JSON stays within target size:
- max 10 workflow names per bucket
- max 100 items for any list
- truncate string fields to 120 chars
- if still >50KB, progressively drop lowest-priority sections in this order:
  1. examples
  2. combination lists
  3. per-workflow breakdowns (keep aggregate totals such as total lockfiles, total bytes, trigger counts, safe-output counts, and overall job/step/script totals)

### 2) Second turn: read summary JSON only

Read only `/tmp/gh-aw/agent/lockfile-stats-summary.json` and derive insights from it.

### 3) Optional third turn: historical comparison

If `/tmp/gh-aw/cache-memory/history/` has prior summaries, compare against latest prior day and include deltas.

## Cache-memory requirements

- Persist the analyzer script at `/tmp/gh-aw/cache-memory/scripts/lockfile_stats_v1.py`.
- Treat `v1` as a schema/version marker and as the source-of-truth filename for this prompt. Bump script name (for example `lockfile_stats_v2.py`) in the prompt **and update all Step 1 script filename references (items 2 and 5)** when adding/removing metrics or changing output structure; bug fixes that preserve schema can keep the same version.
- Save current run summary to `/tmp/gh-aw/cache-memory/history/<YYYY-MM-DD>.json`.
- If historical data exists, include trend deltas in the report.

## Report format

Create one discussion with:

- Executive summary (counts/sizes/date)
- File size distribution
- Trigger analysis
- Safe outputs analysis
- Structural characteristics
- Permission patterns
- Tool & MCP patterns
- 3-5 interesting findings
- Historical trends (if available)
- Recommendations
- Methodology note: "single-script compact JSON analysis"

## Quality constraints

- Be statistically accurate and verifiable.
- Prefer concise tables over long prose.
- If a lockfile is malformed, skip it and report skip count.

Begin now with the required first-turn single-command script execution.

{{#runtime-import shared/noop-reminder.md}}
