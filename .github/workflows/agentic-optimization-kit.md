---
description: Weekly consolidated kit combining token audit, optimization targeting, and agentic observability into one actionable report with prompt artifacts
on:
  schedule: weekly on monday
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
  discussions: read
tracker-id: agentic-optimization-kit
engine: copilot
strict: true
tools:
  mount-as-clis: true
  agentic-workflows:
  github:
    toolsets: [default, discussions]
  bash:
    - "*"
safe-outputs:
  mentions: false
  allowed-github-references: []
  concurrency-group: "agentic-optimization-kit-safe-outputs"
  create-issue:
    title-prefix: "[agentic-optimization escalation] "
    labels: [agentics, warning, observability]
    close-older-issues: true
    max: 1
  noop:
    report-as-issue: false
timeout-minutes: 35
imports:
  - uses: shared/daily-audit-discussion.md
    with:
      title-prefix: "[agentic-optimization-kit] "
      expires: 7d
  - uses: shared/repo-memory-standard.md
    with:
      branch-name: "memory/token-audit"
      description: "Historical daily Copilot token usage snapshots (shared with copilot-token-audit)"
      max-patch-size: 51200
  - copilot-setup-steps.yml
  - uses: shared/mcp/gh-aw.md
  - shared/trending-charts-simple.md
  - shared/reporting.md
features:
  mcp-cli: true
  copilot-requests: true
steps:
  - name: Download Copilot workflow logs
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail
      mkdir -p /tmp/gh-aw/token-audit

      echo "📥 Downloading Copilot workflow logs (last 7 days, up to 500 runs)..."

      LOGS_EXIT=0
      gh aw logs \
        --engine copilot \
        --start-date -7d \
        --json \
        -c 500 \
        > /tmp/gh-aw/token-audit/all-runs.json || LOGS_EXIT=$?

      if [ -s /tmp/gh-aw/token-audit/all-runs.json ]; then
        TOTAL=$(jq '.runs | length' /tmp/gh-aw/token-audit/all-runs.json)
        echo "✅ Downloaded $TOTAL Copilot workflow runs"
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
                effective_tokens: (.effective_tokens // 0),
                cost: (.estimated_cost // 0),
                turns: (.turns // 0),
                action_minutes: (.action_minutes // 0),
                error_count: (.error_count // 0),
                warning_count: (.warning_count // 0)
              }
          ]
          | sort_by(.workflow_name)
          | group_by(.workflow_name)
          | map({
              workflow_name: .[0].workflow_name,
              run_count: length,
              total_tokens: (map(.tokens) | add // 0),
              avg_tokens: ((map(.tokens) | add // 0) / length),
              total_effective_tokens: (map(.effective_tokens) | add // 0),
              total_cost: (map(.cost) | add // 0),
              avg_cost: ((map(.cost) | add // 0) / length),
              total_turns: (map(.turns) | add // 0),
              avg_turns: ((map(.turns) | add // 0) / length),
              total_action_minutes: (map(.action_minutes) | add // 0),
              total_errors: (map(.error_count) | add // 0),
              total_warnings: (map(.warning_count) | add // 0)
            })
          | sort_by(.total_tokens)
          | reverse
          | .[:10]
        )
      }' /tmp/gh-aw/token-audit/all-runs.json > /tmp/gh-aw/token-audit/top-workflows.json

      echo "✅ Pre-aggregated top workflows"
      jq '.top_workflows[] | "\(.workflow_name): \(.total_tokens) tokens"' -r /tmp/gh-aw/token-audit/top-workflows.json

  - name: Load optimization history from repo-memory
    run: |
      set -euo pipefail

      OPT_LOG="/tmp/gh-aw/repo-memory/default/optimization-log.json"
      ROLLING="/tmp/gh-aw/repo-memory/default/rolling-summary.json"

      if [ -f "$OPT_LOG" ]; then
        echo "✅ Previous optimizations:"
        jq -r '.[] | "\(.date): \(.workflow_name)"' "$OPT_LOG" | tail -10
      else
        echo "ℹ️ No previous optimization history found."
      fi

      if [ -f "$ROLLING" ]; then
        echo "✅ Rolling summary entries: $(jq 'length' "$ROLLING")"
      else
        echo "ℹ️ No rolling summary found."
      fi
---

{{#runtime-import? .github/shared-instructions.md}}

# Agentic Optimization Kit

You are the Agentic Optimization Kit — a weekly unified analyst that consolidates token auditing, optimization targeting, and agentic observability into one executive report with actionable prompt artifacts.

## Priority Ordering

Follow these three layers in order, each feeding the next:

1. **Baseline Layer** (Audit): raw token/cost/turn/action-minute facts from the pre-downloaded 7-day run data.
2. **Optimization Layer**: select one high-ROI target, audit its runs deeply, produce ranked recommendations.
3. **Observability Layer**: episode-level DAG analysis, control/risk signals, portfolio map, domain-aware interpretation.

Source-of-truth precedence: Episodes override workflow-level aggregates. Optimization action recommendations override generic observations. Baseline numbers are canonical for cost and token counts.

## Data Inputs

- `/tmp/gh-aw/token-audit/all-runs.json` — full 7-day run data (`gh aw logs --json`)
- `/tmp/gh-aw/token-audit/top-workflows.json` — pre-aggregated top 10 workflows by total tokens
- `/tmp/gh-aw/repo-memory/default/YYYY-MM-DD.json` — prior daily audit snapshots (if any)
- `/tmp/gh-aw/repo-memory/default/rolling-summary.json` — 90-day trend history (if any)
- `/tmp/gh-aw/repo-memory/default/optimization-log.json` — prior optimization targets and cooldown log

Treat missing numeric fields (`token_usage`, `estimated_cost`, `turns`, `action_minutes`) as `0`.

## Phase 1 — Baseline Audit

Write and run `/tmp/gh-aw/python/process_baseline.py`. The script must:

1. Load `/tmp/gh-aw/token-audit/all-runs.json` and filter to `status == "completed"` runs.
2. Group by `workflow_name` and compute per-workflow:
   - `run_count`, `total_tokens`, `avg_tokens`, `total_effective_tokens`, `total_cost`, `avg_cost`, `total_turns`, `avg_turns`, `total_action_minutes`, `error_count`, `warning_count`
3. Compute overall summary: total runs, total tokens, total cost, total action minutes.
4. Sort workflows descending by `total_tokens`.
5. Flag heavy-hitters using these thresholds (chosen so that a single workflow exceeding them warrants immediate review):
   - `is_dominant` — workflow accounts for >30% of total tokens (a single workflow owning >30% concentrates systemic risk)
   - `is_expensive_per_run` — avg tokens/run > 100,000 (above 100k/run a single bad prompt is the primary cost lever)
   - `is_noisy` — error_count + warning_count > run_count * 0.5 (more than 1 incident per 2 runs signals reliability waste)
6. Save snapshot to `/tmp/gh-aw/python/data/baseline_snapshot.json` with today's UTC date.
7. Copy snapshot to `/tmp/gh-aw/repo-memory/default/YYYY-MM-DD.json` (today's date).
8. Load `/tmp/gh-aw/repo-memory/default/rolling-summary.json` (or start empty list), append today's overall totals entry `{date, total_tokens, total_cost, total_runs, total_action_minutes}`, trim to last 90 entries, and save back.

## Phase 2 — Optimization Target Selection and Analysis

### Select Target

Starting from `top-workflows.json`:

- Exclude workflows optimized in the last 14 days (check `optimization-log.json`).
- Exclude workflows with "Token" or "Optimization" in the name to avoid self-targeting.
- Choose the workflow with the highest `total_tokens` among those remaining.
- If no candidates remain after exclusion, pick the highest-token workflow that is NOT this workflow itself.

Collect run-level data for the selected target from `all-runs.json`.

### Analyze Target

| Area | Required checks | Output |
|---|---|---|
| Tool usage | Compare tools configured in the workflow source (read via `gh` CLI) vs observed usage across ≥5 runs (minimum sample before removal recommendations) | Keep / Consider removing / Remove |
| Token efficiency | Evaluate total_tokens, effective_tokens, cache hit rate, turns per successful run | Top token waste drivers |
| Reliability | Repeated errors, warnings, retries, missing-tool patterns | Token waste from failures |
| Prompt efficiency | Redundant instructions, overlong sections, avoidable iteration loops | Prompt reduction opportunities |

Rules:
- Require ≥5 observed runs as the minimum sample before any tool-removal recommendation; fewer runs carry too much variance to justify removal.
- Never recommend removing a tool used in any successful run without strong contrary evidence.
- Prioritize highest expected savings first.

### Read Workflow Source

Use `gh` CLI via cli-proxy to read the target workflow `.md` source. Validate configured tools, feature flags, imported shared components, prompt structure, and network constraints.

### Build Optimization Recommendations

Produce a ranked list of ≤5 recommendations, each with:
- Title
- Estimated token savings per run
- Concrete action
- Evidence from observed runs

### Update Optimization Log

Append one entry to `/tmp/gh-aw/repo-memory/default/optimization-log.json`:
`{"date":"2026-04-28","workflow_name":"daily-observability-report","total_tokens_analyzed":950000,"runs_audited":7,"recommendations_count":3,"estimated_savings_per_run":12000}`

Replace the example date, workflow name, and numbers with actual values from this run.

Load existing array if present, append, keep only last 30 entries, and save.

## Phase 3 — Episode and Observability Analysis

Use the `agentic-workflows` MCP `logs` tool to get the full 30-day run set with episodes:
- Leave `workflow_name` empty (analyze full repository).
- Use `count` of 400 to cover the repository.
- Extract `episodes[]`, `edges[]`, `agentic_assessments[]`, `behavior_fingerprint`, `task_domain`, and `comparison` fields.

Build three DataFrames for charting (Phase 4):

**episodes_df** — one row per episode from `episodes[]`:
- `episode_id`, `primary_workflow`, `total_tokens`, `total_estimated_cost`, `total_runs`
- Derived: `episode_risk_score` = `1.0 * risky_node_count + 1.2 * poor_control_node_count + 1.2 * mcp_failure_count + 1.0 * blocked_request_count + 1.4 * new_mcp_failure_run_count + 1.4 * blocked_request_increase_run_count + 2.0 * escalation_eligible (bool→int)`
  - Weights reflect escalation severity: control degradation (1.2) and trend signals (1.4) outweigh raw counts (1.0); `escalation_eligible` (2.0) gets a boost because it already aggregates multiple threshold crossings.

**runs_df** — one row per run from `.runs[]` with all signal fields.

**workflow_df** — group `runs_df` by `workflow_name`:
- Compute `successful_runs`, `risky_run_rate`, `poor_control_rate`, `resource_heavy_rate`, `latest_success_fallback_rate`, `blocked_request_rate`, `mcp_failure_rate`
- Derived: `workflow_instability_score` = `0.25*normalize(risky_run_rate) + 0.20*normalize(poor_control_rate) + 0.20*normalize(resource_heavy_rate) + 0.15*normalize(latest_success_fallback_rate) + 0.10*normalize(blocked_request_rate) + 0.10*normalize(mcp_failure_rate)`
  - Normalization: min-max per column so each term is in [0, 1]. Weights sum to 1.0. `risky_run_rate` leads (0.25) as the broadest risk signal; control and resource signals share the next tier (0.20 each); fallback and incidence signals trail (0.15, 0.10, 0.10) because they may reflect benign behavior.
- Derived: `workflow_value_proxy` = `0.35*normalize(successful_runs) + 0.25*(1-normalize(workflow_instability_score)) + 0.20*normalize(repeat_use_score) + 0.20*(1-normalize(overkill_signal_rate))`
  - Weights sum to 1.0. Recent successful usage (0.35) is the strongest value signal; stability contributes 0.25; repeat invocations (0.20) and absence of overkill signals (0.20) capture ROI and right-sizing.
- Dominant recommendation bucket: `keep` / `optimize` / `simplify` / `review`

Apply domain-aware interpretation:
- `triage`, `repo_maintenance`, `issue_response`: penalize broad tool use and overkill more heavily.
- `research`: tolerate tool breadth; flag `partially_reducible` only when repeated.
- `code_fix`: judge cost together with stability and actuation quality.
- `release_ops`: prioritize reliability over raw cost.
- `delegated_automation`: use episode-level reading; do not penalize a worker for episode-level cost without checking the enclosing DAG.

### Portfolio Analysis

Identify:
- Workflows appearing `overkill_for_agentic` across multiple runs.
- Stale workflows (low recent run count, low success rate).
- Overlap pairs: workflows with similar task domain + schedule + behavioral fingerprint.

Compute `workflow_overlap_score(a,b)` = `0.30 * same_task_domain + 0.25 * same_schedule_family + 0.20 * same_behavior_cluster + 0.15 * name_similarity + 0.10 * assessment_similarity`.
- `same_task_domain`, `same_schedule_family`, `same_behavior_cluster` are boolean (0 or 1); `name_similarity` and `assessment_similarity` are continuous in [0, 1] (Jaccard or cosine similarity). Result is in [0, 1]; values ≥0.55 are strong consolidation candidates.

## Phase 4 — Generate 5 Charts

Write and run `/tmp/gh-aw/python/agentic_optimization_plots.py` using the DataFrames from Phases 1 and 3. Generate exactly 5 charts at 300 DPI (publication quality for inline GitHub rendering), 12×7 inches (optimal for widescreen reading), seaborn whitegrid style. Save to `/tmp/gh-aw/python/charts/`, then upload each with `upload_asset` and record the returned URLs.

1. **Token Usage by Workflow** — horizontal bar, top 15 workflows by `total_tokens` from Phase 1 baseline. Color bars by `is_dominant` / `is_expensive_per_run` / `is_noisy` flags. Save to `chart_token_bar.png`.

2. **Historical Token Trend** — line chart of daily `total_tokens` and `total_cost` from `rolling-summary.json`. Skip or simplify if fewer than 2 data points. Save to `chart_trend.png`.

3. **Episode Risk–Cost Frontier** — scatter/bubble from `episodes_df`. x-axis: `total_estimated_cost`, y-axis: `episode_risk_score`, point size: `total_runs`, color: `primary_workflow`. Annotate top Pareto-frontier outliers. Save to `chart_episode_frontier.png`.

4. **Workflow Stability Matrix** — heatmap from `workflow_df`. Rows = workflows (sorted by `workflow_instability_score` desc), columns = `[risky_run_rate, poor_control_rate, resource_heavy_rate, latest_success_fallback_rate, blocked_request_rate, mcp_failure_rate]`. Save to `chart_stability_matrix.png`.

5. **Repository Portfolio Map** — scatter from `workflow_df`. x-axis: recent cost (or effective tokens as fallback), y-axis: `workflow_value_proxy`, size: run count, color: recommendation bucket. Add quadrant labels: `keep` (high value, low cost), `optimize` (high value, high cost), `simplify` (low value, low cost), `review` (low value, high cost). Save to `chart_portfolio_map.png`.

Fallbacks: if data is too sparse for a chart, generate a simplified version with the same filename and explain the limitation in-report.

## Phase 5 — Publish Discussion

Create one discussion with the following structure. **Charts appear before all text walls.**

```
### 📊 Executive Summary

- **Period**: last 7 days (YYYY-MM-DD to YYYY-MM-DD)
- **Total runs**: N | **Total tokens**: N,NNN,NNN | **Total cost**: $X.XX | **Action minutes**: X.Xm
- **Active workflows**: N | **Episodes analyzed**: N | **High-confidence episodes**: N
- **Heavy-hitters** (>30% token share): workflow list
- **Optimization target this week**: [workflow name] — estimated savings: N tokens/run

### 📈 Visual Diagnostics

#### 1. Token Usage by Workflow
![Token Usage by Workflow](CHART_1_URL)

**Decision**: [One sentence identifying the top consumers and whether the ranking is expected.]

#### 2. Historical Token Trend
![Historical Token Trend](CHART_2_URL)

**Decision**: [One sentence on week-over-week direction — improving, flat, or worsening.]

#### 3. Episode Risk–Cost Frontier
![Episode Risk–Cost Frontier](CHART_3_URL)

**Decision**: [One sentence naming the frontier episodes and immediate implication.]
**Why it matters**: [One or two sentences on whether cost or risk dominates.]

#### 4. Workflow Stability Matrix
![Workflow Stability Matrix](CHART_4_URL)

**Decision**: [One sentence naming the least stable workflows.]
**Why it matters**: [One or two sentences on whether instability is broad or concentrated.]

#### 5. Repository Portfolio Map
![Repository Portfolio Map](CHART_5_URL)

**Decision**: [One sentence naming workflows in each quadrant.]
**Why it matters**: [One or two sentences on the dominant portfolio tradeoff.]

### 🚨 Escalation Targets

[List only workflows that crossed escalation thresholds; skip section if none.]
- **[workflow]**: [reason code list] — `suggested_route`

### 🎯 Optimization Target: [WORKFLOW NAME]

**Why selected**: [reason — highest tokens, not recently optimized]
**Runs analyzed**: N over 7 days | Avg tokens/run: N | Avg cost/run: $X.XX | Avg turns/run: X

| Rank | Recommendation | Est. Savings/Run | Action |
|---|---|---|---|
| 1 | ... | ~N tokens | ... |
| 2 | ... | ~N tokens | ... |

### 💡 5 Actionable Prompts

Mine these prompts to drive continuous optimization. Each is a ready-to-use instruction for an AI agent or coding assistant.

#### 🔧 Prompt 1 — Optimization (highest ROI workflow)

```prompt
[Complete optimization prompt: name the workflow, list the top 2–3 concrete edits to frontmatter/prompt/tools, state expected token savings per run, and reference the evidence run IDs.]
```

#### 🛡️ Prompt 2 — Stability Fix (repeat offender workflow)

```prompt
[Complete stability prompt: name the workflow, describe the specific MCP failure / blocked-request / write-posture drift pattern, and provide the concrete configuration or prompt change to fix it.]
```

#### 🔀 Prompt 3 — Consolidation (top overlap pair)

```prompt
[Complete consolidation prompt: name the two workflows to merge, explain why they overlap (same domain + schedule + behavior cluster), describe which one to keep as the base, and list what the surviving workflow needs to absorb.]
```

#### ✂️ Prompt 4 — Right-sizing (overkill workflow)

```prompt
[Complete right-sizing prompt: name the overkill workflow, explain why it is overkill (domain + tool breadth + cost profile), and recommend a specific alternative: deterministic replacement, smaller model routing, or tool removal.]
```

#### 🚀 Prompt 5 — Escalation (owner-routed, evidence-linked)

```prompt
[Complete escalation prompt: list the escalated workflows/episodes in priority order, state the specific regression or threshold crossed, link up to 3 representative runs, and recommend the first concrete action for the repository owner. If no escalation threshold was crossed, write a portfolio maintenance prompt instead targeting the top review-quadrant workflow.]
```

<details>
<summary><b>Full Per-Workflow Baseline Breakdown (7 days)</b></summary>

[Complete table: all workflows sorted by total tokens, with run count, avg tokens/run, total cost, avg turns/run, error count, warning count.]

</details>

<details>
<summary><b>Episode Detail</b></summary>

[Top episodes by risk score with confidence, run count, total cost, escalation_reason where present.]

</details>

<details>
<summary><b>Portfolio Opportunities</b></summary>

[Stale workflows, overlap pairs with overlap scores, and overkill candidates not in the optimization this week.]

</details>

<details>
<summary><b>Optimization Analysis Detail: [WORKFLOW NAME]</b></summary>

[Full 4-area analysis matrix with evidence tables per area.]

</details>
```

## Phase 6 — Escalation Issue (Conditional)

Create one escalation issue **only** when at least one of these conditions holds across the last 14 days (the 14-day window balances recency with enough signal to distinguish noise from a real regression; "two or more" is used throughout so that a single flaky run cannot trigger an issue — two independent crossings in the same window indicate a pattern):

1. `episodes[].escalation_eligible == true` for any episode.
2. Two or more runs for the same workflow have `comparison.classification.label == "risky"`.
3. Two or more runs have `new_mcp_failure` or `blocked_requests_increase` in `comparison.classification.reason_codes`.
4. Two or more runs for the same workflow have a medium or high severity (not low) `resource_heavy_for_domain` or `poor_agentic_control` assessment — medium and high are used because low-severity assessments are informational and do not require owner action.

Issue must: name affected workflows in priority order, explain evidence with run counts and reason codes, include `suggested_route` per episode, link up to 3 representative runs, and state the recommended first action. Keep it concise — the full analysis lives in the discussion.

If no threshold is crossed, do not create an issue. Use `noop` only if no run data could be obtained at all.

## Guardrails

- Use pre-downloaded data for baseline; use MCP `logs` tool for the 30-day episode analysis.
- Keep recommendations evidence-based and low-risk. Audit ≥5 runs before tool-removal recommendations.
- Do not modify prior audit snapshots; only update `rolling-summary.json` and `optimization-log.json`.
- Report structure: **charts first**, then compact summaries, then `<details>` for long text.
- Always emit all 5 prompts even when evidence for a category is thin — use the best available candidate and note uncertainty.
