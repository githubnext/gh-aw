---
title: How to Add the Agentic Observability Kit
description: Add a drop-in workflow that turns gh aw logs and audit signals into recurring observability reports and warning issues.
---

Use this guide when a repository already has agentic workflows and needs a supported starter workflow for run-behavior reporting.

The kit reviews recent runs, publishes a recurring discussion report, and opens warning issues only when a workflow shows repeated risk. Use [Projects & Monitoring](/gh-aw/patterns/monitoring/) instead when building a custom project board or status-update workflow.

There are two variants:

- `agentic-observability-kit` publishes into the same repository it analyzes
- `agentic-observability-central-kit` publishes into a central reporting repository

## Add the workflow

Run:

```bash wrap
gh aw add github/gh-aw/agentic-observability-kit
```

This adds the workflow source file to `.github/workflows` so it can be reviewed and customized like any other workflow.

## Add the central variant

Use the central variant when a platform or workflow-operations repository should collect reports from many repositories.

Run:

```bash wrap
gh aw add github/gh-aw/agentic-observability-central-kit
```

Then set the `REPORT_REPOSITORY` repository variable to the destination repository in `owner/repo` format.

Example:

```text
acme/workflow-operations
```

If `REPORT_REPOSITORY` is not set, the workflow falls back to the current repository.

## Review the default outputs

By default, the workflow creates:

- one discussion report per run in the `audits` category
- up to five warning issues when a workflow shows repeated risky behavior

The default issue labels are `agentics` and `warning`.

If the repository uses a different discussion category or labeling convention, edit the `safe-outputs` section after adding the workflow.

## Compile the workflow

After reviewing the file, compile it:

```bash wrap
gh aw compile .github/workflows/agentic-observability-kit.md
```

If the repository already uses a bulk compile step, run that instead.

For the central variant:

```bash wrap
gh aw compile .github/workflows/agentic-observability-central-kit.md
```

## What counts as a warning

The kit opens issues only for repeated, actionable patterns in the last 14 days. By default, that means one workflow crossed the same threshold in at least two runs.

The default warning conditions are:

- repeated `risky` comparison classifications
- repeated `new_mcp_failure` or `blocked_requests_increase` comparison reasons
- repeated medium or high `resource_heavy_for_domain`
- repeated medium or high `poor_agentic_control`

## What stays in the report instead of opening an issue

Some findings stay in the discussion report instead of opening an issue because they are usually optimization candidates rather than incidents:

- repeated `overkill_for_agentic`
- workflows that remain `lean`, `directed`, and `narrow` across successful runs
- workflows that can only be compared to `latest_success` and never find a meaningful cohort match

## Customizing the kit

The starter workflow is designed to be modified after import.

Common changes are:

- widen the analysis window from 14 days to 30 days
- change labels to match internal triage processes
- route discussions to a central reporting repository
- route warning issues to a platform or workflow-operations repository
- tighten or relax warning thresholds depending on run volume

If the organization wants one central place for reports, update the `create-discussion` and `create-issue` safe outputs to target that repository.

If a central platform repository is already the operating model, prefer `agentic-observability-central-kit` instead of manually rewriting the single-repo starter.

## Related documentation

- [Debugging Workflows](/gh-aw/troubleshooting/debugging/)
- [GH-AW as an MCP Server](/gh-aw/reference/gh-aw-as-mcp-server/)
- [Projects & Monitoring](/gh-aw/patterns/monitoring/)
- [CentralRepoOps](/gh-aw/patterns/central-repo-ops/)
