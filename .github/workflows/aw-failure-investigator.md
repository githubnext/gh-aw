---
emoji: "🔍"
description: Investigates [aw] failures from the last 6 hours, correlates with open agentic-workflows issues, closes fixed issues, and opens focused fix sub-issues when needed
on:
  schedule:
    - cron: "every 6h"
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
tracker-id: aw-failure-investigator
engine: claude
tools:
  bash: ["*"]
cache:
  - key: aw-failure-investigator-prefetch-${{ github.run_id }}
    name: Failure investigator prefetch
    path: /tmp/gh-aw/failure-investigator
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
      mkdir -p /tmp/gh-aw/failure-investigator
      python3 - <<'PY'
      import json
      import os
      import subprocess
      from datetime import datetime, timezone
      
      REPO = os.environ["GITHUB_REPOSITORY"]
      OUT = "/tmp/gh-aw/failure-investigator/prefetch.json"
      TRACKER_ID = "aw-failure-investigator"
      LOOKBACK = "-6h"
      MAX_FAILED_RUNS = 20
      MAX_RUNS_TO_FETCH = 200
      MAX_LOG_TAIL_LINES = 200
      
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
      
      logs = run_json(["gh", "aw", "logs", "--start-date", LOOKBACK, "--json", "-c", str(MAX_RUNS_TO_FETCH)]) or {"runs": []}
      failed_runs = []
      for run in logs.get("runs", []):
          if (run.get("conclusion") or "").lower() != "failure":
              continue
          failed_runs.append(
              {
                  "run_id": run.get("run_id"),
                  "workflow_name": run.get("workflow_name"),
                  "workflow_path": run.get("workflow_path"),
                  "created_at": run.get("created_at"),
                  "status": run.get("status"),
                  "conclusion": run.get("conclusion"),
                  "url": run.get("url"),
              }
          )
          if len(failed_runs) >= MAX_FAILED_RUNS:
              break
      
      failure_details = []
      for run in failed_runs:
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
      
          failed_steps = []
          truncated_error_logs = []
          for job in run_view.get("jobs", []):
              if (job.get("conclusion") or "").lower() == "failure":
                  for step in job.get("steps", []):
                      if (step.get("conclusion") or "").lower() == "failure":
                          failed_steps.append(
                              {
                                  "job_id": job.get("databaseId"),
                                  "job_name": job.get("name"),
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
                                  "job_name": job.get("name"),
                                  "line_count": len(tail_lines),
                                  "tail_200_lines": "\n".join(tail_lines),
                              }
                          )
      
          failure_details.append(
              {
                  "run_id": run_id,
                  "workflow_name": run_view.get("workflowName") or run_view.get("name"),
                  "url": run_view.get("url"),
                  "created_at": run_view.get("createdAt"),
                  "status": run_view.get("status"),
                  "conclusion": run_view.get("conclusion"),
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
          "lookback_window": "6h",
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
---

# [aw] Failure Investigator (6h)

Investigate agentic workflow failures from the last 6 hours and produce actionable issue tracking with sub-issues.

## Scope

- **Repository**: `${{ github.repository }}`
- **Lookback window**: last 6 hours
- **Issue query to inspect first**: <https://github.com/github/gh-aw/issues?q=is%3Aissue%20state%3Aopen%20label%3Aagentic-workflows>
- **Deterministic pre-fetch payload**: `/tmp/gh-aw/failure-investigator/prefetch.json`

## Mission

1. Find recent failures from agentic workflows in the last 6 hours.
2. Correlate findings with currently open `agentic-workflows` issues.
3. Perform large-scale failure analysis using logs + audit + audit-diff.
4. Close fixed/stale issues first, then create only the minimum necessary linked fix sub-issues.

## Required Investigation Steps

### 0) Use deterministic pre-fetch payload first (required)

Read `/tmp/gh-aw/failure-investigator/prefetch.json` first. It already includes:
- recent failed run IDs for the 6-hour window
- failed step names
- truncated error logs (up to last 200 lines per failed job)
- existing open tracking issues filtered by `gh-aw-tracker-id: aw-failure-investigator`

Use this payload as the primary discovery dataset. Only call additional logs/list APIs when a field is missing or stale.

### 1) Fetch and review existing issue context

Use the `issue-context-fetcher` agent to retrieve open `agentic-workflows` issues grouped into clusters, gaps, and potential duplicates. Merge that with `existing_tracking_issues` from the pre-fetch payload when correlating failures.

### 2) Collect workflow runs and isolate failures (last 6h)

Start from `failed_run_ids` and `failures` in the pre-fetch payload to build clustered failure rows with representative + comparator run IDs.
Only run additional logs queries if the pre-fetch payload cannot support a cluster decision.

### 3) Deep-dive each failure cluster with `audit`

Use the `cluster-evidence-extractor` agent, passing the clusters from step 2, to retrieve per-cluster evidence (dominant error, tool-failure pattern, anomalies, failure class).

### 4) Compare behavior with `audit-diff`

Use `agentic-workflows` MCP `audit-diff` to compare:
- failed run vs nearest successful run of the same workflow, or
- failed run vs prior failed run to detect drift

Identify regressions and deltas (metrics/tooling/firewall/MCP behavior) that support fix recommendations.

### 5) Close fixed issues first, then add focused sub-issues

First, identify currently open `agentic-workflows` issues that are now fixed, stale, or no longer actionable based on fresh evidence, and close them using `update-issue`.

Then, if new uncovered work remains, add **sub-issues** for concrete fixes to the **most recent open parent report issue** instead of creating a new parent by default.

Only create a new parent report issue (temporary ID format `aw_` + 3-8 alphanumeric characters) when **P0 failures have no existing tracking coverage**.

Each new sub-issue must include:
- clear problem statement
- affected workflows and run IDs
- probable root cause
- specific proposed remediation
- success criteria / verification

## Output Requirements

**Report Formatting**: Use `###` or lower for all headers in the issue body. Wrap evidence/log excerpts and verbose tables in `<details><summary>Section Name</summary>` tags.

### Parent report issue structure

Include these sections:
1. Executive summary
2. Failure clusters (table)
3. Evidence (logs/audit/audit-diff)
4. Existing issue correlation
5. Proposed fix roadmap (P0/P1/P2)
6. Sub-issues created

### Sub-issue quality bar

- Prefer a few high-quality, actionable sub-issues over many weak ones.
- Avoid duplicates of already-open issues unless new evidence materially changes scope.
- Reference the parent issue and the concrete run IDs analyzed.

## Decision Rules

- If there are **no failures** in the last 6h, or no actionable delta vs existing issues, call `noop` with a concise reason.
- If failures exist but are already fully tracked, prefer closing stale/fixed issues and avoid creating new issues.
- Only create a new parent report issue when P0 failures have no existing tracking coverage.
- Prefer closing stale/fixed issues over creating new issues when issue volume is high.
- Always be explicit about confidence and unknowns.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```

## agent: `issue-context-fetcher`
---
description: Fetches open agentic-workflows issues and groups them into clusters, gaps, and duplicate candidates
model: small
---
Find open issues labeled `agentic-workflows` for `${{ github.repository }}`.
Group findings into existing tracked clusters, tracking gaps, and potential duplicates.

Return only JSON:
```json
{
  "clusters": [{"name":"", "issue_numbers":[]}],
  "gaps": [{"failure_signature":"", "reason":""}],
  "potential_duplicates": [{"issue_numbers":[], "reason":""}]
}
```

## agent: `failure-dataset-builder`
---
description: Fetches the last 6h workflow logs and builds clustered failure rows with representative and comparator run IDs
model: small
---
Use `agentic-workflows` MCP `logs` for the last 6 hours (for example `start_date: "-6h"`), including enough runs to cover the window.
Cluster failures by signature and include representative and comparator run IDs.

Return only JSON:
```json
{
  "failure_rows": [{"cluster_id":"", "workflow":"", "engine":"", "failure_signature":"", "representative_failed_run_id":"", "comparator_success_run_id":"", "run_ids":[]}]
}
```

## agent: `cluster-evidence-extractor`
---
description: Extracts per-cluster audit evidence including dominant errors, tool patterns, anomalies, and failure class
model: small
---
Given failure clusters from step 2, call `agentic-workflows` MCP `audit` for each cluster's representative failed run and a successful comparator when available.
Extract dominant error, tool-failure pattern, anomalies, and failure class.

Return only JSON:
```json
{
  "cluster_evidence": [{"cluster_id":"", "dominant_error":"", "tool_failure_pattern":"", "anomalies":[],"failure_class":"","evidence_run_ids":[]}]
}
```
