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
  - shared/observability-otlp.md
  - shared/otel-queries.md
pre-agent-steps:
  - name: Collect workflow telemetry snapshot
    uses: actions/github-script@v9
    env:
      AW_YIELD_TELEMETRY_OUT: /tmp/aw-yield-telemetry-summary.json
    with:
      script: |
        const fs = require("fs");
        const owner = context.repo.owner;
        const repo = context.repo.repo;
        const now = Date.now();
        const windowMs = 90 * 24 * 60 * 60 * 1000;
        const workflowIdToSourcePath = new Map();
        const workflows = await github.paginate(github.rest.actions.listRepoWorkflows, {
          owner,
          repo,
          per_page: 100,
        });
        for (const workflow of workflows) {
          const workflowPath = workflow.path || "";
          if (!workflowPath.startsWith(".github/workflows/") || !workflowPath.endsWith(".lock.yml")) {
            continue;
          }
          workflowIdToSourcePath.set(workflow.id, workflowPath.replace(/\.lock\.yml$/, ".md"));
        }

        const aggregates = new Map();
        let pageCount = 0;
        let reachedWindowLimit = false;
        for await (const page of github.paginate.iterator(github.rest.actions.listWorkflowRunsForRepo, {
          owner,
          repo,
          status: "completed",
          per_page: 100,
        })) {
          pageCount += 1;
          for (const run of page.data.workflow_runs || []) {
            const sourcePath = workflowIdToSourcePath.get(run.workflow_id);
            if (!sourcePath) {
              continue;
            }
            const createdAt = run.created_at ? Date.parse(run.created_at) : Number.NaN;
            if (!Number.isNaN(createdAt) && createdAt < now - windowMs) {
              reachedWindowLimit = true;
              break;
            }
            const startedAt = run.run_started_at ? Date.parse(run.run_started_at) : Number.NaN;
            const updatedAt = run.updated_at ? Date.parse(run.updated_at) : Number.NaN;
            const durationSeconds =
              !Number.isNaN(startedAt) && !Number.isNaN(updatedAt) && updatedAt >= startedAt
                ? (updatedAt - startedAt) / 1000
                : 0;
            const aggregate = aggregates.get(sourcePath) || {
              runs: 0,
              successfulRuns: 0,
              runtimeSeconds: 0,
              runtimeSamples: 0,
            };
            aggregate.runs += 1;
            if (run.conclusion === "success") {
              aggregate.successfulRuns += 1;
            }
            if (durationSeconds > 0) {
              aggregate.runtimeSeconds += durationSeconds;
              aggregate.runtimeSamples += 1;
            }
            aggregates.set(sourcePath, aggregate);
          }
          if (reachedWindowLimit || pageCount >= 10) {
            break;
          }
        }

        const workflow_metrics = {};
        for (const [path, aggregate] of aggregates.entries()) {
          workflow_metrics[path] = {
            workflow_path: path,
            workflow_invocation_count: aggregate.runs,
            success_rate: aggregate.runs ? Number((aggregate.successfulRuns / aggregate.runs).toFixed(4)) : 0,
            runtime_duration: aggregate.runtimeSamples
              ? Number((aggregate.runtimeSeconds / aggregate.runtimeSamples).toFixed(2))
              : 0,
            observed: aggregate.runs > 0,
            validated: aggregate.runs > 0,
            source: "github-actions-runs",
          };
        }

        fs.writeFileSync(
          process.env.AW_YIELD_TELEMETRY_OUT,
          JSON.stringify(
            {
              generated_at: new Date().toISOString(),
              source: "github-actions-runs",
              window_days: 90,
              workflow_metrics,
            },
            null,
            2,
          ) + "\n",
        );
  - name: Precompute workflow portfolio data
    uses: actions/github-script@v9
    env:
      AW_YIELD_WORKSPACE: ${{ github.workspace }}
      AW_YIELD_WORKFLOWS: .github/workflows
      AW_YIELD_OUT: /tmp/aw-yield-precompute.json
      AWY_OTEL_SUMMARY_JSON: /tmp/aw-yield-telemetry-summary.json
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
- Telemetry = facts. Deterministic precompute/postcompute = math. Agent = interpretation.
- Do **not** recompute raw scores, ranking, overlap values, fractions, or portfolio math from scratch.
- Do **not** invent telemetry, economics, confidence, or success evidence.
- Use the OTel queries skill only when the precompute file explicitly indicates that telemetry exists and needs brief interpretation.
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
- optional telemetry summaries already folded into the precompute payload

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
