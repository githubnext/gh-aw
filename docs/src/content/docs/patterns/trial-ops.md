---
title: TrialOps
description: Test and validate agentic workflows in isolated trial repositories before deploying to production
sidebar:
  badge: { text: 'Testing', variant: 'tip' }
---

TrialOps extends [SideRepoOps](/gh-aw/patterns/side-repo-ops/) with temporary trial repositories for safely validating and iterating on workflows before production deployment. The `trial` command creates isolated private repos where workflows execute and capture safe outputs (issues, PRs, comments) without affecting your actual codebase.

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

> [!WARNING]
> Direct mode creates real issues and PRs in the target repository. Only use with test repositories.

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

> [!TIP]
> Trial repositories persist between runs. Reuse the same `--host-repo` name across test sessions.

## Advanced Patterns

### Issue Context

Provide issue context for issue-triggered workflows:

```bash
gh aw trial githubnext/agentics/triage-workflow \
  --trigger-context "https://github.com/myorg/repo/issues/123"
```

### Auto-merge PRs

Automatically merge created PRs (useful for testing multi-step workflows):

```bash
gh aw trial githubnext/agentics/feature-workflow --auto-merge-prs
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

## Trial Mode Limitations

- **Requires `workflow_dispatch` trigger** - Add to workflows that only trigger on issues/PRs/schedules
- **Safe outputs needed** - Workflows without safe outputs execute but create no visible results
- **No simulated events** - Use `--trigger-context` to provide event context like issue payloads
- **Private repositories** - Trial repos count toward your private repository quota
- **API rate limits** - Space out large runs or use `--repeat` instead of separate invocations

## Best Practices

### Development Workflow

Iterate locally: write the workflow, preview with `--dry-run`, run a trial, adjust, compare variants side-by-side, validate consistency with `--repeat`, then deploy.

### Testing Strategy

```bash
# Unit testing - individual workflows
gh aw trial ./workflows/triage.md --delete-host-repo-after

# Integration testing - with actual content
gh aw trial ./workflows/code-review.md --clone-repo myorg/real-repo

# Regression testing - before/after comparison
gh aw trial ./workflow.md --host-repo regression-baseline
gh aw trial ./workflow.md --host-repo regression-test

# Performance testing - execution time and tokens
gh aw trial ./workflow.md --repeat 5
```

### Prompt Engineering

Iteratively refine prompts: run baseline → modify prompt → test variant → compare outputs → repeat.

### CI/CD Integration

```yaml
name: Test Workflows
on: [pull_request]
jobs:
  trial:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: Install gh-aw
        run: gh extension install github/gh-aw
      - name: Trial workflow
        env:
          COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
        run: gh aw trial ./.github/workflows/my-workflow.md --delete-host-repo-after --yes
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `workflow not found` | Use correct format: `owner/repo/workflow-name`, `owner/repo/.github/workflows/workflow.md`, or `./local-workflow.md` |
| `workflow_dispatch not supported` | Add `workflow_dispatch:` to workflow frontmatter `on:` section |
| `authentication failed` | See [Authentication](/gh-aw/reference/auth/). Trial automatically prompts for missing secrets and uploads them to the trial repo |
| `failed to create trial repository` | Check `gh auth status`, verify quota with `gh api user \| jq .plan`, try explicit `--host-repo name` |
| `execution timed out` | Increase with `--timeout 60` (minutes, default: 30) |
| No issues/PRs created | Configure `safe-outputs` in workflow frontmatter, check Actions logs for errors |

## Related Documentation

- [SideRepoOps](/gh-aw/patterns/side-repo-ops/) - Run workflows from separate repositories
- [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/) - Coordinate across multiple repositories
- [Orchestration](/gh-aw/patterns/orchestration/) - Orchestrate multi-issue initiatives
- [CLI Commands](/gh-aw/setup/cli/) - Complete CLI reference
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Configuration options
- [Workflow Triggers](/gh-aw/reference/triggers/) - Including workflow_dispatch
- [Security Best Practices](/gh-aw/introduction/architecture/) - Authentication and security
