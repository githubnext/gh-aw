---
name: Bulk-Close Resolved Failure Issues
description: Closes open [aw] * failed issues where the referenced workflow has since run successfully, leaving genuinely broken workflows open
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  actions: read
engine: codex
tools:
  github:
    toolsets: [issues, actions]
safe-outputs:
  close-issue:
    required-title-prefix: "[aw]"
    target: "*"
    max: 100
  add-comment:
    target: "*"
    max: 1
  noop:
timeout-minutes: 30
---

# Bulk-Close Resolved Failure Issues

You are a cleanup agent whose job is to reduce issue noise by closing `[aw] * failed` issues for workflows that have since recovered and run successfully.

## Background

Between April 8–9, 2026 a silent startup crash in Copilot CLI v1.0.21 caused ~63 workflows to fail with `exit code 1` and zero output.  With v1.0.22 now deployed and workflows recovering, most of these failure issues are no longer actionable.

## Repository

${{ github.repository }}

## Your Task

### Phase 1 — Enumerate All Open Failure Issues

Use the GitHub MCP server to search for every open issue that:
- Has the label `agentic-workflows`
- Has a title matching the pattern `[aw] * failed`

```
search_issues(query="repo:${{ github.repository }} is:issue is:open label:agentic-workflows \"[aw]\" \"failed\" in:title")
```

Collect all matching issues.  If there are more than 30, paginate until you have the full list.

### Phase 2 — Determine the Workflow Name for Each Issue

The issue title always has the form:

```
[aw] <workflow-name> failed
```

Strip the `[aw] ` prefix and the ` failed` suffix to obtain `<workflow-name>`.

For example:
- `[aw] sub-issue-closer failed` → workflow name `sub-issue-closer`
- `[aw] deep-report failed` → workflow name `deep-report`

### Phase 3 — Check the Most Recent Workflow Run

For each `<workflow-name>`, use the GitHub MCP actions tools to retrieve the most recent workflow run:

```
list_workflow_runs(workflow_id="<workflow-name>.lock.yml", per_page=1)
```

Evaluate the outcome:

| Condition | Action |
|-----------|--------|
| Most recent run `conclusion == "success"` | Close the issue (see Phase 4) |
| Most recent run `conclusion` is `failure`, `cancelled`, or `timed_out` | Leave the issue open |
| No runs found, or run is still in progress | Leave the issue open |

> **Important**: Only close an issue when you can confirm the workflow's most recent **completed** run was successful.  When in doubt, leave it open.

### Phase 4 — Close Resolved Issues

For each issue identified as resolved in Phase 3, emit a `close_issue` safe-output call:

```json
{"type": "close_issue", "issue_number": <N>, "body": "Resolved — workflow ran successfully after Copilot CLI v1.0.22 upgrade."}
```

Do **not** add any extra commentary to the closing comment.  Use exactly that body text.

### Phase 5 — Post a Summary Comment

After processing all issues, post a single summary comment on **the first issue that is still open** after the bulk-close run (i.e. the first issue from Phase 1 whose most recent run was still failing).  If every issue was successfully closed and no open issues remain, skip this step entirely.

The comment should include:
- Total issues inspected
- Issues closed (list them by number and title)
- Issues left open (list them by number and title, with a one-line reason)

Emit the summary using `add_comment`:

```json
{"type": "add_comment", "issue_number": <summary-issue-number>, "body": "<your summary>"}
```

### NOOP Fallback

If no `[aw] * failed` issues are found, or all issues were already closed before this run, you **must** emit a `noop`:

```json
{"noop": {"message": "No open [aw] * failed issues found — nothing to close."}}
```
