---
emoji: "🔍"
description: Investigates [aw] failures from the last 6 hours, correlates with open agentic-workflows issues, closes fixed issues, and opens focused fix sub-issues when needed
on:
  schedule:
    - cron: "every 6h"
  workflow_dispatch:
max-daily-ai-credits: 10000
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
tracker-id: aw-failure-investigator
engine: claude
experiments:
  tone_variant:
    variants: [clinical, assertive, narrative]
    description: "Tests whether report tone (clinical/assertive/narrative) affects output efficiency and engineer engagement on failure investigation issues"
    hypothesis: "H0: no change in output_length_chars across tone variants. H1: assertive tone produces shorter, more actionable outputs than clinical or narrative, with equivalent or better sub-issue quality."
    metric: output_length_chars
    secondary_metrics: [issue_creation_rate, sub_issue_link_count, run_duration_seconds]
    guardrail_metrics:
      - name: run_success_rate
        threshold: ">=0.85"
    min_samples: 50
    weight: [34, 33, 33]
    start_date: "2026-05-31"
    analysis_type: mann_whitney
    tags: [tone, output-quality, triage]
    issue: 36105
sandbox:
  agent:
    sudo: false
tools:
  cli-proxy: true
  github:
    mode: gh-proxy
    toolsets: [actions, issues, pull_requests]
  bash: ["*"]
cache:
  - key: aw-failure-investigator-prefetch-${{ github.run_id }}
    name: Failure investigator prefetch
    path: /tmp/gh-aw/agent/failure-investigator
safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[aw-failures] "
    labels: [agentic-workflows, automation, cookie]
    max: 2
    group: true
  update-issue:
    target: "*"
    max: 10
  link-sub-issue:
    max: 10
  noop:
timeout-minutes: 60
imports:
  - uses: shared/meta-analysis-base.md
    with:
      toolsets: [default, actions]
  - shared/reporting.md

  - shared/otlp.md
steps:
  - name: Deterministic pre-fetch for failure analysis
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -euo pipefail
      mkdir -p /tmp/gh-aw/agent/failure-investigator
      python3 - <<'PY'
      import json
      import os
      import subprocess
      from datetime import datetime, timedelta, timezone
      from pathlib import Path
      from urllib.parse import urlencode
      
      REPO = os.environ["GITHUB_REPOSITORY"]
      OUT = "/tmp/gh-aw/agent/failure-investigator/prefetch.json"
      TRACKER_ID = "aw-failure-investigator"
      LOOKBACK_HOURS = 6
      FAILURE_CONCLUSIONS = {"failure", "timed_out", "startup_failure", "cancelled"}
      MAX_DISCOVERY_PAGES = 20
      # Most dominant signatures appear in the final 30-60 lines.
      MAX_LOG_TAIL_LINES = 50
      # Deep-dive budget: investigate at most this many distinct failed runs.
      MAX_FAILURES_TO_DETAIL = 5
      AGENTIC_WORKFLOW_PATHS = {
          f".github/workflows/{path.name}"
          for path in Path(".github/workflows").glob("*.lock.yml")
      }
      
      def cmd_display(args):
          return " ".join(args)
      
      def run_json(args):
          try:
              out = subprocess.check_output(args, text=True, stderr=subprocess.STDOUT)
              return json.loads(out)
          except subprocess.CalledProcessError as error:
              print(f"Warning: command failed: {cmd_display(args)}")
              print(error.output)
              return None
          except json.JSONDecodeError as error:
              print(f"Warning: non-JSON output from command: {cmd_display(args)} ({error})")
              return None
          except OSError as error:
              print(f"Warning: could not execute command: {cmd_display(args)} ({error})")
              return None
      
      def run_text(args):
          try:
              return subprocess.check_output(args, text=True, stderr=subprocess.STDOUT)
          except subprocess.CalledProcessError as error:
              print(f"Warning: command failed: {cmd_display(args)}")
              print(error.output)
              return ""
          except OSError as error:
              print(f"Warning: could not execute command: {cmd_display(args)} ({error})")
              return ""
      
      def run_api_json(endpoint, params):
          query = urlencode(params)
          return run_json(["gh", "api", f"{endpoint}?{query}"])
      
      def is_failure_conclusion(conclusion):
          return (conclusion or "").lower() in FAILURE_CONCLUSIONS
      
      def normalize_workflow_path(path):
          return (path or "").split("@", 1)[0]
      
      def is_agentic_workflow_path(path):
          workflow_path = normalize_workflow_path(path)
          if AGENTIC_WORKFLOW_PATHS:
              return workflow_path in AGENTIC_WORKFLOW_PATHS
          print("Warning: no local .lock.yml workflows found; falling back to workflow path suffix matching")
          return workflow_path.endswith(".lock.yml")
      
      def isoformat_z(dt):
          return dt.astimezone(timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")
      
      def list_failed_agentic_runs():
          created_since = isoformat_z(datetime.now(timezone.utc) - timedelta(hours=LOOKBACK_HOURS))
          page = 1
          failed_runs = []
      
          while True:
              response = run_api_json(
                  f"repos/{REPO}/actions/runs",
                  {
                      "exclude_pull_requests": "true",
                      "status": "completed",
                      "created": f">={created_since}",
                      "per_page": "100",
                      "page": str(page),
                  },
              ) or {}
              workflow_runs = response.get("workflow_runs") or []
              if not workflow_runs:
                  break
      
              for run in workflow_runs:
                  workflow_path = normalize_workflow_path(run.get("path"))
                  if not is_agentic_workflow_path(workflow_path):
                      continue
                  if not is_failure_conclusion(run.get("conclusion")):
                      continue
      
                  failed_runs.append(
                      {
                          "run_id": run.get("id"),
                          "workflow_name": run.get("name"),
                          "workflow_path": workflow_path,
                          "created_at": run.get("created_at"),
                          "status": run.get("status"),
                          "conclusion": run.get("conclusion"),
                          "url": run.get("html_url"),
                      }
                  )
      
              if len(workflow_runs) < 100:
                  break
              if page >= MAX_DISCOVERY_PAGES:
                  print(f"Warning: reached pagination cap ({MAX_DISCOVERY_PAGES} pages) while listing workflow runs")
                  break
              page += 1
      
          failed_runs.sort(key=lambda run: run.get("created_at") or "", reverse=True)
          return failed_runs
      
      failed_runs = list_failed_agentic_runs()
      
      # Cap the number of runs to detail so the payload stays compact.
      failure_details = []
      for run in failed_runs[:MAX_FAILURES_TO_DETAIL]:
          run_id = run.get("run_id")
          if not run_id:
              continue
      
          run_view = run_json(
              [
                  "gh",
                  "run",
                  "view",
                  str(run_id),
                  "--repo",
                  REPO,
                  "--json",
                  "databaseId,url,name,workflowName,createdAt,conclusion,status,jobs",
              ]
          )
          if not run_view:
              continue
      
          failed_job_names = []
          failed_steps = []
          truncated_error_logs = []
          agent_job_conclusion = None
          for job in run_view.get("jobs", []):
              job_name = job.get("name")
              job_conclusion = (job.get("conclusion") or "").lower()
              if (job_name or "").lower() == "agent":
                  agent_job_conclusion = job_conclusion or None
      
              if is_failure_conclusion(job_conclusion):
                  if job_name:
                      failed_job_names.append(job_name)
                  for step in job.get("steps", []):
                      if is_failure_conclusion(step.get("conclusion")):
                          failed_steps.append(
                              {
                                  "job_id": job.get("databaseId"),
                                  "job_name": job_name,
                                  "step_name": step.get("name"),
                              }
                          )
      
                  job_id = job.get("databaseId")
                  if job_id:
                      log_text = run_text(
                          [
                              "gh",
                              "run",
                              "view",
                              str(run_id),
                              "--repo",
                              REPO,
                              "--job",
                              str(job_id),
                              "--log-failed",
                          ]
                      )
                      if log_text:
                          tail_lines = log_text.splitlines()[-MAX_LOG_TAIL_LINES:]
                          truncated_error_logs.append(
                              {
                                  "job_id": job_id,
                                  "job_name": job_name,
                                  "line_count": len(tail_lines),
                                  "tail_lines": "\n".join(tail_lines),
                              }
                          )
      
          failure_details.append(
              {
                  "run_id": run_id,
                  "workflow_name": run_view.get("workflowName") or run_view.get("name"),
                  "workflow_path": run.get("workflow_path"),
                  "url": run_view.get("url"),
                  "created_at": run_view.get("createdAt"),
                  "status": run_view.get("status"),
                  "conclusion": run_view.get("conclusion"),
                  "failed_job_names": sorted(set(failed_job_names)),
                  "agent_job_conclusion": agent_job_conclusion,
                  "failed_steps": failed_steps,
                  "truncated_error_logs": truncated_error_logs,
              }
          )
      
      existing_tracking_issues = run_json(
          [
              "gh",
              "issue",
              "list",
              "--repo",
              REPO,
              "--state",
              "open",
              "--search",
              f"gh-aw-tracker-id: {TRACKER_ID}",
              "--limit",
              "100",
              "--json",
              "number,title,state,url,labels,createdAt,updatedAt",
          ]
      ) or []
      
      payload = {
          "generated_at": datetime.now(timezone.utc).isoformat(),
          "repository": REPO,
          "lookback_window": f"{LOOKBACK_HOURS}h",
          "failed_run_ids": [run.get("run_id") for run in failed_runs if run.get("run_id")],
          "failures": failure_details,
          "existing_tracking_issues": existing_tracking_issues,
      }
      
      with open(OUT, "w", encoding="utf-8") as f:
          json.dump(payload, f, indent=2)
          f.write("\n")
      
      print(f"Wrote deterministic prefetch payload to {OUT}")
      print(f"Failed runs in payload: {len(payload['failed_run_ids'])}")
      print(f"Existing tracking issues in payload: {len(existing_tracking_issues)}")
      PY
features:
  gh-aw-detection: true
---

# [aw] Failure Investigator (6h)

Investigate agentic workflow failures from the last 6 hours and produce actionable issue tracking with sub-issues.

## Scope

- **Repository**: `${{ github.repository }}`
- **Lookback window**: last 6 hours
- **Issue query to inspect first**: <https://github.com/github/gh-aw/issues?q=is%3Aissue%20state%3Aopen%20label%3Aagentic-workflows>
- **Deterministic pre-fetch payload**: `/tmp/gh-aw/agent/failure-investigator/prefetch.json`

## Mission

1. Find recent failures from agentic workflows in the last 6 hours.
2. Correlate findings with currently open `agentic-workflows` issues.
3. Perform large-scale failure analysis using logs + audit + audit-diff.
4. Close fixed/stale issues first, then create only the minimum necessary linked fix sub-issues.

## Required Investigation Steps

### 0) Read deterministic pre-fetch payload first (required)

Read `failed_run_ids`, `failures`, and `existing_tracking_issues` **once** from `/tmp/gh-aw/agent/failure-investigator/prefetch.json`.
Do not re-read this file; keep the parsed data in context for all subsequent steps.
Use this payload as the primary discovery dataset and build clustered failure rows with representative + comparator run IDs.
Definitions for step 0 clustering:
- representative run ID: failed run that best captures the dominant signature in a cluster
- comparator run ID: nearest successful run of the same workflow when available, otherwise nearest prior failed run
Only call additional logs/list APIs when a required field is missing or stale.

**Early exit**: If `failed_run_ids` is empty, or every failure signature is already covered by an open issue in `existing_tracking_issues`, call `noop` immediately with a brief explanation and stop.

### 1) Classify failures and correlate with existing issues

Use the `failure-classifier` agent, passing the full `failures` array (including `truncated_error_logs`) from the prefetch payload.
It returns compact JSON with severity-ranked clusters (id, severity, representative_run_id, comparator_run_id, workflows, error_signature).

Then use the `issue-matcher` agent, passing the cluster summaries and `existing_tracking_issues` from the prefetch payload.
It returns which clusters are already tracked (matched) and which are new gaps.

Keep the combined cluster + tracking mapping in context for steps 2-4.

**Early exit**: If all untracked clusters from `issue-matcher` are P2 severity (no P0 or P1 gaps), call `noop` with a brief explanation and stop.

### 2) Deepen evidence for untracked clusters

Use the `cluster-evidence-extractor` agent for untracked P0 and P1 clusters identified in step 1 (at most 3 clusters). The agent prefers pre-fetched `truncated_error_logs` and only calls `audit` for clusters whose logs are too sparse — capping total `audit` MCP calls at 2 across all clusters.

### 3) Compare behavior with `audit-diff`

Use `agentic-workflows` MCP `audit-diff` to compare **the single highest-severity cluster only** (1 comparison maximum):
- failed run vs nearest successful run of the same workflow, or
- failed run vs prior failed run to detect drift

Identify regressions and deltas (metrics/tooling/firewall/MCP behavior) that support fix recommendations.

### 4) Close fixed issues first, then add focused sub-issues

First, identify currently open `agentic-workflows` issues that are now fixed, stale, or no longer actionable based on fresh evidence, and close them using `update-issue`.

Then, if new uncovered work remains, add **sub-issues** for concrete fixes to the **most recent open parent report issue** instead of creating a new parent by default.

Only create a new parent report issue when **P0 failures have no existing tracking coverage**.

Each new sub-issue must include:
- clear problem statement
- affected workflows and run IDs
- probable root cause
- specific proposed remediation
- success criteria / verification

## Tone Variant Instructions

{{#if experiments.tone_variant == 'assertive'}}
Tone instruction: Write in assertive, action-first style. Open every section with a direct imperative recommendation (e.g., "Fix the retry loop in workflow X — it causes 40% of P0 failures"). Keep rationale to one sentence. Prioritize brevity and actionability over completeness.
{{#elseif experiments.tone_variant == 'narrative'}}
Tone instruction: Write in narrative style. Use flowing prose paragraphs to explain what happened, why it matters, and what the broader context is. Readers should finish each section with a clear mental model of the failure, not just a list of facts.
{{#else}}
Tone instruction: Write in clinical, neutral style. Use numbered lists, avoid editorializing, and anchor every claim to a metric or log reference. This is the baseline behavior.
{{/if}}

## Output Requirements

Follow `shared/reporting.md` for header levels and progressive disclosure formatting.
When creating a parent report issue, include: executive summary, failure cluster table, evidence, existing issue correlation, fix roadmap (P0/P1/P2), and sub-issues created.
For sub-issues, prioritize high-quality actionable items, avoid duplicates unless scope changed, and reference the parent issue and analyzed run IDs.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation.

## agent: `failure-classifier`
---
description: Groups pre-fetched failure runs into severity-ranked clusters by error signature and workflow
model: small
---
You receive a JSON array of `failures` from the pre-fetch payload. Each entry has `run_id`, `workflow_name`, `workflow_path`, `conclusion`, `failed_job_names`, `failed_steps`, and `truncated_error_logs`.

Group failures into clusters:
1. Cluster by dominant error signature extracted from `truncated_error_logs[].tail_lines`; group failures from the same workflow with matching signatures together.
2. Assign severity — P0: agent/infra crash, data loss risk, or startup_failure; P1: persistent failure pattern across ≥2 runs; P2: isolated or transient.
3. Pick `representative_run_id` (run that best illustrates the cluster) and `comparator_run_id` (nearest run_id not in the cluster, for diff).
4. Copy `truncated_error_logs` from the representative run only into the output cluster.

Return only JSON — no prose:
```json
{
  "clusters": [
    {
      "id": "cluster-1",
      "severity": "P0",
      "representative_run_id": 123,
      "comparator_run_id": 456,
      "workflows": ["workflow-name"],
      "error_signature": "one-line dominant error",
      "run_ids": [123, 789],
      "truncated_error_logs": []
    }
  ]
}
```

## agent: `issue-matcher`
---
description: Matches failure clusters to existing open tracking issues to identify coverage gaps
model: small
---
You receive:
- `clusters`: array of failure clusters (id, severity, workflows, error_signature)
- `existing_tracking_issues`: array of open issues (number, title, labels, url)

For each cluster, determine whether an existing issue already tracks it. Match by error_signature similarity and workflow name overlap.

Return only JSON — no prose:
```json
{
  "matched": [
    {"cluster_id": "cluster-1", "issue_number": 42, "confidence": "high"}
  ],
  "gaps": [
    {"cluster_id": "cluster-2", "reason": "no existing issue covers this signature"}
  ]
}
```

## agent: `cluster-evidence-extractor`
---
description: Extracts per-cluster audit evidence including dominant errors, tool patterns, anomalies, and failure class
model: small
---
Given failure clusters with their `truncated_error_logs` from the prefetch payload:
1. If a cluster has ≥10 lines of pre-fetched error logs, extract evidence directly from those logs — do **not** call `audit`.
2. Only call `agentic-workflows` MCP `audit` when pre-fetched logs are missing or fewer than 5 lines. Cap total `audit` calls at **2** across all clusters.
3. When calling `audit`, request only `artifacts: ["usage", "agent"]` to limit download size.

Extract dominant error, tool-failure pattern, anomalies, and failure class.

Return only JSON:
```json
{
  "cluster_evidence": [{"cluster_id":"", "dominant_error":"", "tool_failure_pattern":"", "anomalies":[],"failure_class":"","evidence_run_ids":[]}]
}
```
