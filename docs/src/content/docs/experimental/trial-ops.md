---
title: TrialOps
description: Test and validate agentic workflows in isolated trial repositories before deploying to production
sidebar:
  badge: { text: 'Testing', variant: 'tip' }
---

TrialOps uses temporary trial repositories for safely validating and iterating on workflows before deployment to target repositories. The `trial` command creates isolated private repos where workflows execute and capture safe outputs (issues, PRs, comments) without affecting your actual codebase.

## How Trial Mode Works

```bash
gh aw trial githubnext/agentics/weekly-research
```

The CLI creates a temporary private repository (default: `gh-aw-trial`), installs and executes the workflow via `workflow_dispatch`. Results are saved locally in `trials/weekly-research.DATETIME-ID.json`, in the trial repository on GitHub, and summarized in the console.

## Repository Modes

| Mode | Flag | Description |
|------|------|-------------|
| Default | (none) | `github.repository` points to your repo; outputs go to trial repo |
| Direct | `--repo myorg/test-repo` | Runs in specified repo; creates real issues/PRs there |
| Logical | `--logical-repo myorg/target-repo` | Simulates running against specified repo; outputs in trial repo |
| Clone | `--clone-repo myorg/real-repo` | Clones repo contents so workflows can analyze actual code |

## Basic Usage

### Dry-Run Mode

Preview what would happen without executing workflows or creating repositories:

```bash
gh aw trial ./my-workflow.md --dry-run
```

### Single Workflow

```bash
gh aw trial githubnext/agentics/weekly-research  # From GitHub
gh aw trial ./my-workflow.md                      # Local file
```

### Multiple Workflows

Compare workflows side-by-side with combined results:

```bash
gh aw trial githubnext/agentics/daily-plan githubnext/agentics/weekly-research
```

Outputs: individual result files plus `trials/combined-results.DATETIME.json`.

### Repeated Trials

Test consistency by running multiple times:

```bash
gh aw trial githubnext/agentics/my-workflow --repeat 3
```

### Custom Trial Repository

```bash
gh aw trial githubnext/agentics/my-workflow --host-repo my-custom-trial
gh aw trial ./my-workflow.md --host-repo .  # Use current repo
```

## Advanced Patterns

### Issue Context

Provide issue context for issue-triggered workflows:

```bash
gh aw trial githubnext/agentics/triage-workflow \
  --trigger-context "https://github.com/myorg/repo/issues/123"
```

### Append Instructions

Test workflow responses to additional constraints without modifying the source:

```bash
gh aw trial githubnext/agentics/my-workflow \
  --append "Focus on security issues and create detailed reports."
```

### Cleanup Options

```bash
gh aw trial ./my-workflow.md --delete-host-repo-after        # Delete after completion
gh aw trial ./my-workflow.md --force-delete-host-repo-before # Clean slate before running
```

## Understanding Trial Results

Results are saved in `trials/*.json` with workflow runs, issues, PRs, and comments viewable in the trial repository's Actions and Issues tabs.

**Result file structure:**

```json
{
  "workflow_name": "weekly-research",
  "run_id": "12345678",
  "safe_outputs": {
    "issues_created": [{
      "number": 5,
      "title": "Research quantum computing trends",
      "url": "https://github.com/user/gh-aw-trial/issues/5"
    }]
  },
  "agentic_run_info": {
    "duration_seconds": 45,
    "token_usage": 2500
  }
}
```

**Success indicators:** Green checkmark, expected outputs created, no errors in logs.

**Common issues:**

- **Workflow dispatch failed** - Add `workflow_dispatch` trigger
- **No safe outputs** - Configure safe outputs in workflow
- **Permission errors** - Verify API keys
- **Timeout** - Use `--timeout 60` (minutes)

## Comparing Multiple Workflows

Run multiple workflows to compare quality, quantity, performance, and consistency:

```bash
gh aw trial v1.md v2.md v3.md --repeat 2
cat trials/combined-results.*.json | jq '.results[] | {workflow: .workflow_name, issues: .safe_outputs.issues_created | length}'
```

## Related Documentation

- [SideRepoOps](/gh-aw/patterns/side-repo-ops/) - Run workflows from separate repositories
- [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/) - Coordinate across multiple repositories
- [Orchestration](/gh-aw/patterns/orchestration/) - Orchestrate multi-issue initiatives
- [CLI Commands](/gh-aw/setup/cli/) - Complete CLI reference
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Configuration options
- [Workflow Triggers](/gh-aw/reference/triggers/) - Including workflow_dispatch
- [Security Best Practices](/gh-aw/introduction/architecture/) - Authentication and security
