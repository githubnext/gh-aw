---
name: Agentic Workflow Portfolio Yield
description: Weekly portfolio analysis of agentic workflows using deterministic scoring, overlap detection, and OTel-backed evidence for governance recommendations
on:
  schedule: weekly on monday around 09:00
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
engine: copilot
strict: true
timeout-minutes: 25
network:
  allowed: [defaults, github]
tools:
  bash: true
  github:
    mode: gh-proxy
    toolsets: [default, actions, pull_requests]
safe-outputs:
  mentions: false
  allowed-github-references: []
  create-issue:
    labels: [automation, report, observability]
    max: 1
    close-older-issues: true
    expires: 30d
imports:
  - shared/otel-observability.md
pre-agent-steps:
  - name: Precompute workflow portfolio data
    uses: actions/github-script@v9
    env:
      AW_YIELD_WORKSPACE: ${{ github.workspace }}
      AW_YIELD_WORKFLOWS: .github/workflows
      AW_YIELD_OUT: /tmp/aw-yield-precompute.json
    with:
      script: |
        const path = require("path");
        const { runPrecompute } = require(path.join(process.env.AW_YIELD_WORKSPACE, "scripts/aw_yield_precompute.cjs"));
        await runPrecompute({
          workspace: process.env.AW_YIELD_WORKSPACE,
          workflows: process.env.AW_YIELD_WORKFLOWS,
          out: process.env.AW_YIELD_OUT,
        });
post-steps:
  - name: Finalize workflow portfolio report
    uses: actions/github-script@v9
    env:
      AW_YIELD_WORKSPACE: ${{ github.workspace }}
      AW_YIELD_PRECOMPUTE: /tmp/aw-yield-precompute.json
      AW_YIELD_AGENT_OUTPUT: /tmp/gh-aw
      AW_YIELD_OUT: /tmp/aw-yield-final.json
    with:
      script: |
        const path = require("path");
        const { runPostcompute } = require(path.join(process.env.AW_YIELD_WORKSPACE, "scripts/aw_yield_postcompute.cjs"));
        await runPostcompute({
          workspace: process.env.AW_YIELD_WORKSPACE,
          precompute: process.env.AW_YIELD_PRECOMPUTE,
          agentOutput: process.env.AW_YIELD_AGENT_OUTPUT,
          out: process.env.AW_YIELD_OUT,
        });
---
# Agentic Workflow Portfolio Yield

You are the semantic interpreter for the repository's agentic workflow portfolio.

## Hard Rules

- Treat `/tmp/aw-yield-precompute.json` as the factual source of truth.
- OTel = facts. Deterministic precompute/postcompute = math. Agent = interpretation.
- Do **not** recompute raw scores, ranking, overlap values, fractions, or portfolio math from scratch.
- Do **not** invent telemetry, economics, confidence, or success evidence.
- Use the `otel` MCP server only for aggregated summaries when the precompute file explicitly indicates that telemetry exists but needs brief interpretation.
- Do not request or inspect raw traces.
- Do not perform write actions with GitHub tools.

## Required Interpretation Scope

Explicitly evaluate these three levels:

1. **Workflow level** — is each workflow worth running?
2. **Episode level** — do related workflow groups create value or coordination drag?
3. **Portfolio level** — is the overall workflow ecosystem becoming more coherent and reusable, or more fragmented and noisy?

## Inputs

Read and rely on:

- `/tmp/aw-yield-precompute.json`
- workflow recommendation seeds already computed there
- overlap clusters already computed there
- organizational health signals already computed there
- optional OTel summaries already folded into the precompute payload

## Deliverables

1. Write `/tmp/gh-aw/portfolio-yield-agent.json` with this shape:

```json
{
  "executive_summary": "",
  "recommendations": {
    "keep": [{"path": "", "reason": ""}],
    "revise": [{"path": "", "reason": ""}],
    "merge": [{"path": "", "reason": ""}],
    "instrument": [{"path": "", "reason": ""}],
    "retire": [{"path": "", "reason": ""}]
  },
  "highest_value_actions": ["", "", ""],
  "deterministic_vs_agentic_findings": [""],
  "episode_observations": [""],
  "retirement_candidates": [""],
  "consolidation_opportunities": [""],
  "instrumentation_gaps": [""],
  "telemetry_claims": []
}
```

2. Produce exactly one `create_issue` safe output titled:

`Agentic Workflow Portfolio Yield Report — YYYY-MM-DD`

3. The issue body must include these sections:

- `# Agentic Workflow Portfolio Yield Report`
- `## Executive Summary`
- `## Portfolio Health`
- `## Workflow Portfolio`
- `## Overlap Clusters`
- `## Episode-Level Observations` (only if evidence exists)
- `## Organizational Health Signals`
- `## Deterministic vs Agentic Findings`
- `## Highest-Value Actions`
- `## Retirement Candidates`
- `## Consolidation Opportunities`
- `## Instrumentation Gaps`
- `## Deterministic Portfolio JSON`

## Recommendation Rules

- Keep = high yield, high trust, low risk, low overlap.
- Revise = plausible usefulness but excessive cost, maintenance drag, risk, or agentic fraction.
- Merge = overlapping workflows or clusters competing for the same niche.
- Instrument = missing telemetry, observability, or safe evidence.
- Retire = low yield, low trust, and high drag.

## Usage

This workflow runs weekly and also supports manual `workflow_dispatch` for on-demand portfolio reviews.
