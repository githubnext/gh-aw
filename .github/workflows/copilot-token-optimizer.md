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
    mode: gh-proxy
    toolsets: [issues]
  bash:
    - "*"
  repo-memory:
    branch-name: "memory/token-audit"
    description: "Historical daily Copilot token usage snapshots (shared with copilot-token-audit)"
    file-glob: ["*.json", "*.jsonl", "*.csv", "*.md"]
    max-file-size: 102400
    max-patch-size: 51200
safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[copilot-token-optimizer] "
    close-older-issues: true
    max: 1
  threat-detection: false
timeout-minutes: 30
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

  - name: Aggregate top workflows by token usage
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
source: githubnext/agentic-ops/workflows/copilot-token-optimizer.md@c780d8324935fe1fb30fce545832c10a8f904039
---

# Copilot Token Usage Optimizer

You are the Copilot Token Optimizer. Pick one high-cost workflow, audit recent runs, and create a conservative optimization issue with measurable savings.

## Objectives

1. Select one workflow using repo-memory and pre-aggregated data.
2. Analyze tokens, turns, errors, and tool usage patterns across multiple runs.
3. Propose safe, high-impact optimizations with evidence.
4. Publish one issue and update optimization history.

## Data Access Guidelines

All GitHub API access goes through the `gh` CLI via the cli-proxy — there are **no GitHub MCP tools** available. Always filter API responses with `--jq` or pipe through `jq` to extract only the fields you need. Loading full JSON payloads into context wastes tokens; every extra field is overhead.

**Preferred patterns:**

```bash
REPO="${{ github.repository }}"

# ✅ Extract only the fields you need from a file
gh api "repos/$REPO/contents/.github/workflows/my-workflow.md" \
  --jq '.content' | base64 -d

# ✅ List workflow runs — keep only essential metadata
gh api "repos/$REPO/actions/workflows/my-workflow.yml/runs?per_page=10" \
  --jq '.workflow_runs[] | {id, name, conclusion, run_started_at}'

# ✅ Combine multi-step reads into one bash block with pipes
gh api "repos/$REPO/contents/.github/workflows/my-workflow.md" \
  --jq '.content' | base64 -d | sed -n '1,/^---$/{ /^---$/d; p }' | head -40

# ❌ Never load full unfiltered responses — drops everything into context
gh api "repos/$REPO/actions/workflows/my-workflow.yml/runs"
```

Prefer `--jq` on `gh api` calls over a separate `| jq` step when the filter is simple — it avoids piping the full response through the shell. Use `| jq` for multi-step transformations or when chaining with other commands.

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
| Tool usage | Compare configured tools from workflow source (read with `gh api … --jq '.content' \| base64 -d \| sed -n …` to extract only the frontmatter) vs observed usage across multiple runs | Keep / Consider removing / Remove |
| Token efficiency | Evaluate token totals, effective tokens, cache efficiency, turns | Top token waste drivers |
| Reliability | Repeated errors, warnings, retries, missing tools | Token waste from failures |
| Prompt efficiency | Redundant instructions, overlong sections, avoidable iteration | Prompt reduction opportunities |

### Tool-Usage Efficiency Patterns

When auditing runs, check for these common anti-patterns that waste tokens:

- **Batch independent reads**: look for sequential file reads or API calls that could be requested in a single tool-use block — each extra turn repeats the full context
- **Chain bash commands**: look for separate bash tool calls that could be combined with `&&` — each call adds a full context echo
- **Prefer typed tools**: look for `bash cat`, `bash grep`, `bash find -name` when `view`, `grep`, `glob` would return more concise output
- **Consolidate GitHub API sequences**: look for multiple sequential `gh api` calls that could be combined into fewer round-trips with `jq` filtering
- **Don't retry without diagnosing**: look for blind retries of the same failing operation without error analysis — each retry wastes a full turn

Rules:

- Audit at least 5 runs when available before removal recommendations.
- Never recommend removing a tool used in any successful run unless there is strong contrary evidence.
- Prioritize highest expected savings first.

## Phase 3 — Read Workflow Source

Use `gh api` with `--jq` (via cli-proxy) to read the target workflow `.md` source. Extract only the sections you need — do not load the whole file if a targeted slice is sufficient.

```bash
REPO="${{ github.repository }}"
# Replace <workflow-name> with the actual .md filename, e.g. "copilot-agent-analysis"
WF_PATH=".github/workflows/<workflow-name>.md"

# Read the full source (only when necessary — prefer targeted slices below)
gh api "repos/$REPO/contents/$WF_PATH" --jq '.content' | base64 -d

# Extract frontmatter only (tools, features, network, permissions)
gh api "repos/$REPO/contents/$WF_PATH" --jq '.content' | base64 -d \
  | awk '/^---$/{n++; if(n==2) exit} n==1'

# Extract the prompt body only (everything after the closing ---)
gh api "repos/$REPO/contents/$WF_PATH" --jq '.content' | base64 -d \
  | awk 'f; /^---$/{f=1}'
```

Validate from the source:

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

### Report Formatting Requirements

- Use `###` for main sections and `####` for subsections inside the issue body.
- Keep the selected workflow, token profile summary, and ranked recommendations visible without collapsible sections.
- Use `<details><summary>...</summary>` blocks for long supporting tables, raw run evidence, and lower-priority context.
- If you cite specific workflow runs, format them as links like `[§12345](https://github.com/${{ github.repository }}/actions/runs/12345)` and include up to 3 under `**References:**`.

## Phase 5 — Update Optimization Log

Append one entry to `/tmp/gh-aw/repo-memory/default/optimization-log.json`:

`{"date":"YYYY-MM-DD","workflow_name":"...","total_tokens_analyzed":N,"runs_audited":N,"recommendations_count":N,"estimated_savings_per_run":N}`

Load existing array if present, append, keep only last 30 entries, and save.

## Guardrails

- Use pre-downloaded data; do not re-download logs.
- Keep recommendations evidence-based and low-risk.
- Do not modify audit snapshots; only update `optimization-log.json`.
