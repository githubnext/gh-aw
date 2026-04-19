---
description: Daily optimizer that identifies a high-token-usage Copilot workflow, audits its runs, and recommends efficiency improvements
on:
  schedule:
    - cron: "daily around 14:00 on weekdays"
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
tracker-id: copilot-token-optimizer
engine: copilot
tools:
  github:
    toolsets: [issues]
  bash:
    - "*"
safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[copilot-token-optimizer] "
    close-older-issues: true
    max: 1
timeout-minutes: 30
imports:
  - uses: shared/repo-memory-standard.md
    with:
      branch-name: "memory/token-audit"
      description: "Historical daily Copilot token usage snapshots (shared with copilot-token-audit)"
      max-patch-size: 51200
  - copilot-setup-steps.yml
  - shared/reporting.md
features:
  mcp-cli: true
  cli-proxy: true
steps:
  - name: Download recent Copilot workflow logs
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail
      mkdir -p /tmp/gh-aw/token-audit

      echo "📥 Downloading Copilot workflow logs (last 7 days)..."

      LOGS_EXIT=0
      gh aw logs \
        --engine copilot \
        --start-date -7d \
        --json \
        -c 50 \
        > /tmp/gh-aw/token-audit/all-runs.json || LOGS_EXIT=$?

      if [ -s /tmp/gh-aw/token-audit/all-runs.json ]; then
        TOTAL=$(jq '.runs | length' /tmp/gh-aw/token-audit/all-runs.json)
        echo "✅ Downloaded $TOTAL Copilot workflow runs (last 7 days)"
        if [ "$LOGS_EXIT" -ne 0 ]; then
          echo "⚠️ gh aw logs exited with code $LOGS_EXIT (partial results — likely API rate limit)"
        fi
      else
        echo "❌ No log data downloaded (exit code $LOGS_EXIT)"
        echo '{"runs":[],"summary":{}}' > /tmp/gh-aw/token-audit/all-runs.json
      fi

  - name: Pre-aggregate top workflows by token usage
    run: |
      set -euo pipefail
      mkdir -p /tmp/gh-aw/token-audit

      jq '{
        generated_at: (now | todateiso8601),
        window_days: 7,
        top_workflows: (
          [.runs[]
            | select(.status == "completed")
            | {
                workflow_name: .workflow_name,
                tokens: (.token_usage // 0),
                cost: (.estimated_cost // 0),
                turns: (.turns // 0),
                action_minutes: (.action_minutes // 0)
              }
          ]
          | group_by(.workflow_name)
          | map({
              workflow_name: .[0].workflow_name,
              run_count: length,
              total_tokens: (map(.tokens) | add),
              avg_tokens: ((map(.tokens) | add) / length),
              total_cost: (map(.cost) | add),
              total_turns: (map(.turns) | add),
              total_action_minutes: (map(.action_minutes) | add)
            })
          | sort_by(.total_tokens)
          | reverse
          | .[:10]
        )
      }' /tmp/gh-aw/token-audit/all-runs.json > /tmp/gh-aw/token-audit/top-workflows.json

      echo "✅ Generated top workflow summary at /tmp/gh-aw/token-audit/top-workflows.json"
      jq '.top_workflows' /tmp/gh-aw/token-audit/top-workflows.json

  - name: Load optimization history
    run: |
      set -euo pipefail

      OPT_LOG="/tmp/gh-aw/repo-memory/default/optimization-log.json"
      if [ -f "$OPT_LOG" ]; then
        echo "✅ Previous optimizations:"
        jq -r '.[] | "\(.date): \(.workflow_name)"' "$OPT_LOG"
      else
        echo "ℹ️ No previous optimization history found."
      fi
---
{{#runtime-import? .github/shared-instructions.md}}

# Copilot Token Usage Optimizer

You are the Copilot Token Optimizer. Pick one high-cost workflow, audit recent runs, and create a conservative optimization issue with measurable savings.

## Objectives

1. Select one workflow using repo-memory and pre-aggregated data.
2. Analyze tokens, turns, errors, and tool usage patterns across multiple runs.
3. Propose safe, high-impact optimizations with evidence.
4. Publish one issue and update optimization history.

## Data Inputs

- `/tmp/gh-aw/token-audit/all-runs.json`: full 7-day run data (`gh aw logs --json`).
- `/tmp/gh-aw/token-audit/top-workflows.json`: pre-aggregated top 10 workflows by total tokens.
- `/tmp/gh-aw/repo-memory/default/YYYY-MM-DD.json`: daily audit snapshots.
- `/tmp/gh-aw/repo-memory/default/optimization-log.json`: prior optimizations (if present).

Treat missing numeric fields (`token_usage`, `estimated_cost`, `turns`, `action_minutes`) as `0`.

## Phase 1 — Select Target

- Start from `top-workflows.json`.
- Exclude workflows optimized in the last 14 days (use `optimization-log.json`).
- Exclude workflows with "Token" in the name to avoid self-targeting.
- Choose the highest token workflow that remains.
- If no snapshot/history exists, derive candidates directly from `all-runs.json`.

Then collect run-level data for the selected workflow:

- run count
- total and average tokens
- total and average cost
- total and average turns
- conclusions/error patterns

## Phase 2 — Analyze

Use this compact analysis matrix:

| Area | Required checks | Output |
|---|---|---|
| Tool usage | Compare configured tools from workflow source (read via `gh api` through cli-proxy) vs observed usage across multiple runs | Keep / Consider removing / Remove |
| Token efficiency | Evaluate token totals, effective tokens, cache efficiency, turns | Top token waste drivers |
| Reliability | Repeated errors, warnings, retries, missing tools | Token waste from failures |
| Prompt efficiency | Redundant instructions, overlong sections, avoidable iteration | Prompt reduction opportunities |

Rules:

- Audit at least 5 runs when available before removal recommendations.
- Never recommend removing a tool used in any successful run unless there is strong contrary evidence.
- Prioritize highest expected savings first.

## Phase 3 — Read Workflow Source

Use `gh` CLI requests (via cli-proxy) to read the target workflow `.md` source and validate. Run `gh` commands normally in bash steps; cli-proxy forwards them over its HTTP interface:

- configured tools and feature flags
- imported shared components
- prompt structure and verbosity
- network/sandbox constraints relevant to recommendations

## Phase 4 — Publish Optimization Issue

Create one issue with:

- **Target workflow + reason selected**
- **Analysis period + runs analyzed**
- **Token profile table** (total tokens, avg tokens/run, total cost, avg turns/run, cache efficiency)
- **Ranked recommendations** with:
  - title
  - estimated token savings per run
  - concrete action
  - evidence from observed runs
- **Caveats** (sampling limits, edge cases)

Use `<details>` blocks for long supporting tables.

## Phase 5 — Update Optimization Log

Append one entry to `/tmp/gh-aw/repo-memory/default/optimization-log.json`:

`{"date":"YYYY-MM-DD","workflow_name":"...","total_tokens_analyzed":N,"runs_audited":N,"recommendations_count":N,"estimated_savings_per_run":N}`

Load existing array if present, append, keep only last 30 entries, and save.

## Guardrails

- Use pre-downloaded data; do not re-download logs.
- Keep recommendations evidence-based and low-risk.
- Do not modify audit snapshots; only update `optimization-log.json`.
