---
title: TrialOps
description: Test and validate agentic workflows in isolated trial repositories before deploying to production
sidebar:
  badge: { text: 'Testing', variant: 'tip' }
---

TrialOps is a testing pattern that runs agentic workflows in isolated trial repositories to safely validate behavior, compare approaches, and iterate on prompts without affecting your production environment. The `trial` command creates temporary private repositories where workflows execute in "trial mode"—capturing all safe outputs (issues, PRs, comments) without modifying your actual codebase.

## When to Use TrialOps

Use TrialOps when you need to test workflow behavior before production deployment, compare multiple workflow implementations side-by-side, validate prompts changes across iterations, debug workflow logic in isolation, or demonstrate workflow capabilities to stakeholders with real execution results.

## How Trial Mode Works

When you run a trial, the CLI:

1. **Creates a trial repository** - A temporary private repository in your GitHub account (default: `gh-aw-trial`)
2. **Installs workflow(s)** - Downloads specified workflows and compiles them in the trial repo
3. **Executes workflows** - Triggers workflow runs via `workflow_dispatch`
4. **Captures outputs** - All safe outputs (issues, PRs, comments) are created in the trial repository
5. **Saves results** - Captures safe output metadata locally (`trials/` directory) and in the trial repo

```bash
gh aw trial githubnext/agentics/weekly-research
```

**Output locations:**
- **Local**: `trials/weekly-research.DATETIME-ID.json` (safe output metadata)
- **GitHub**: Trial repository at `<username>/gh-aw-trial` (actual issues/PRs/comments)
- **Console**: Summary of execution results and links

## Repository Modes

Trial mode supports four different repository modes for different testing scenarios:

### Default Mode (Logical Repository Simulation)

Simulates running the workflow against your current repository. The workflow sees `github.repository` pointing to your current repo, but all outputs go to the trial repository.

```bash
gh aw trial githubnext/agentics/my-workflow
# Simulates running as if installed in current repo
# github.repository = "myorg/current-repo"
# Safe outputs go to trial repo
```

**Use when**: Testing how a workflow would behave in your main repository without creating actual issues/PRs there.

### Direct Mode (`--repo`)

Runs the workflow directly in a specified repository without simulation. The workflow is installed and executed in that repository, and all outputs are created there.

```bash
gh aw trial githubnext/agentics/my-workflow --repo myorg/test-repo
# Installs and runs workflow in myorg/test-repo
# All outputs created in myorg/test-repo
```

**Use when**: Testing in a dedicated sandbox repository you control.

:::caution[Production Risk]
Direct mode creates real issues and PRs in the target repository. Only use with repositories intended for testing.
:::

### Logical Repository Mode (`--logical-repo`)

Explicitly simulates running against a specified repository. Similar to default mode but targets a specific repository instead of the current directory.

```bash
gh aw trial githubnext/agentics/my-workflow --logical-repo myorg/target-repo
# Simulates running in myorg/target-repo
# github.repository = "myorg/target-repo"
# Safe outputs go to trial repo
```

**Use when**: Testing cross-repository workflows or workflows designed for specific repositories.

### Clone Mode (`--clone-repo`)

Clones the contents of a specified repository into the trial repository before running the workflow. The workflow runs with access to the actual file structure and code.

```bash
gh aw trial githubnext/agentics/code-review --clone-repo myorg/real-repo
# Clones myorg/real-repo contents into trial repo
# Workflow can read actual files and code structure
# Safe outputs created in trial repo
```

**Use when**: Testing workflows that need to analyze actual code, file structures, or repository content (e.g., code quality analysis, documentation updates).

## Basic Usage

### Single Workflow Trial

Test a single workflow with default settings:

```bash
gh aw trial githubnext/agentics/weekly-research
```

This creates a trial repository, runs the workflow once, and displays results.

### Local Workflow Trial

Test a workflow file from your local filesystem:

```bash
gh aw trial ./my-workflow.md
```

```bash
gh aw trial ./.github/workflows/triage.md
```

**Use when**: Developing and testing workflows before committing them to a repository.

### Multiple Workflow Comparison

Run multiple workflows simultaneously to compare their approaches:

```bash
gh aw trial \
  githubnext/agentics/daily-plan \
  githubnext/agentics/weekly-research
```

**Outputs:**
- Individual result files for each workflow
- Combined result file: `trials/combined-results.DATETIME.json`
- Side-by-side comparison in the trial repository

**Use when**: Evaluating different implementations, comparing AI engines, or A/B testing prompt variations.

### Repeated Trials

Run a workflow multiple times to test consistency:

```bash
gh aw trial githubnext/agentics/my-workflow --repeat 3
```

This executes the workflow 3 times total (not 3 additional times), useful for evaluating non-deterministic behavior or testing rate limits.

### Custom Trial Repository

Specify a custom trial repository name:

```bash
gh aw trial githubnext/agentics/my-workflow --host-repo my-custom-trial
```

Or use the current repository as the trial host:

```bash
gh aw trial ./my-workflow.md --host-repo .
```

:::tip[Reusing Trial Repositories]
By default, the trial repository persists between runs. Specify the same `--host-repo` name to reuse a trial repository across multiple test sessions.
:::

## Advanced Patterns

### Testing with Issue Context

Test issue-triggered workflows by providing issue context:

```bash
gh aw trial githubnext/agentics/triage-workflow \
  --trigger-context "https://github.com/myorg/repo/issues/123"
```

The workflow receives the issue context and can access issue metadata through GitHub tools.

### Auto-merge Testing

Test PR creation workflows with automatic merging:

```bash
gh aw trial githubnext/agentics/feature-workflow \
  --auto-merge-prs
```

Any pull requests created during the trial are automatically merged, useful for testing multi-step workflows that depend on merged PRs.

### Engine Comparison

Compare behavior across different AI engines:

```bash
# Test with Claude
gh aw trial ./my-workflow.md --engine claude

# Test with Copilot
gh aw trial ./my-workflow.md --engine copilot

# Test with OpenAI
gh aw trial ./my-workflow.md --engine codex
```

:::note[Engine Requirements]
Ensure appropriate API keys are set in your environment:
- Copilot: `COPILOT_GITHUB_TOKEN`
- Claude: `CLAUDE_CODE_OAUTH_TOKEN` or `ANTHROPIC_API_KEY`
- OpenAI: `OPENAI_API_KEY`
:::

### Appending Test Instructions

Add extra instructions to a workflow during trial:

```bash
gh aw trial githubnext/agentics/my-workflow \
  --append "Focus specifically on security issues and create detailed reports."
```

**Use when**: Testing how workflows respond to additional constraints or specific scenarios without modifying the source workflow.

### Cleanup After Trial

Delete the trial repository after execution completes:

```bash
gh aw trial githubnext/agentics/my-workflow --delete-host-repo-after
```

**Use when**: Running one-off tests or in CI/CD pipelines where persistence isn't needed.

### Force Recreate Trial Repository

Delete and recreate the trial repository before running:

```bash
gh aw trial githubnext/agentics/my-workflow --force-delete-host-repo-before
```

**Use when**: Ensuring a completely clean slate for testing.

## Understanding Trial Results

### Local Result Files

Trial results are saved in the `trials/` directory with timestamps:

```bash
trials/
├── weekly-research.20250109-143022-abc123.json
├── daily-plan.20250109-143045-def456.json
└── combined-results.20250109-143100.json
```

**Result file structure:**

```json
{
  "workflow_name": "weekly-research",
  "run_id": "12345678",
  "safe_outputs": {
    "issues_created": [
      {
        "number": 5,
        "title": "Research quantum computing trends",
        "url": "https://github.com/user/gh-aw-trial/issues/5",
        "labels": ["research", "weekly"]
      }
    ],
    "prs_created": []
  },
  "agentic_run_info": {
    "duration_seconds": 45,
    "token_usage": 2500
  },
  "timestamp": "2025-01-09T14:30:22Z"
}
```

### GitHub Trial Repository

The trial repository contains:

- **Workflow runs** - View in Actions tab to see execution logs
- **Issues** - All issues created by the workflow
- **Pull requests** - All PRs created by the workflow  
- **Comments** - Comments added to issues/PRs
- **trials/ directory** - Result JSON files committed for reference

### Interpreting Results

**Success indicators:**
- Workflow run completes successfully (green checkmark)
- Expected issues/PRs are created
- Safe output metadata matches expectations
- No error messages in workflow logs

**Common issues:**
- **Workflow dispatch failed** - Check that workflow has `workflow_dispatch` trigger
- **No safe outputs** - Workflow may need adjustments to create outputs
- **Permission errors** - Verify API keys are set correctly
- **Timeout** - Increase timeout with `--timeout 60` (in minutes)

## Comparing Multiple Workflows

When running multiple workflows, use the combined results file to compare:

```bash
gh aw trial \
  ./workflow-v1.md \
  ./workflow-v2.md \
  ./workflow-v3.md
```

**Comparison strategies:**

1. **Quality comparison** - Review issues created by each workflow for detail, accuracy, and usefulness
2. **Quantity comparison** - Compare number of outputs (more isn't always better)
3. **Performance comparison** - Check execution time from `agentic_run_info`
4. **Consistency comparison** - Use `--repeat 3` to test reproducibility

**Example comparison workflow:**

```bash
# Run 3 variants, 2 times each
gh aw trial v1.md v2.md v3.md --repeat 2

# Review results
cat trials/combined-results.*.json | jq '.results[] | {workflow: .workflow_name, issues: .safe_outputs.issues_created | length}'
```

## Trial Mode Limitations

**Workflows must support workflow_dispatch:**
```yaml
on:
  workflow_dispatch:  # Required for trial mode
```

If your workflow only triggers on issues, pull requests, or schedules, add a `workflow_dispatch` trigger for trial testing.

**Safe outputs are required:**

Trial mode captures safe outputs. Workflows without safe outputs will execute but won't create visible results in the trial repository.

**No simulated events:**

Trial mode triggers workflows with `workflow_dispatch`. Event-specific context (like issue payloads) isn't available unless you use `--trigger-context` to provide it.

**Private trial repositories:**

Trial repositories are created as private by default. They won't appear in your public repository list but count toward your private repository quota.

**API rate limits:**

Multiple workflow runs can consume API rate limits. Space out large trial runs or use `--repeat` instead of separate invocations.

## Best Practices

### Development Workflow

1. **Develop locally** - Write and edit workflows in your editor
2. **Trial locally** - Test with `gh aw trial ./my-workflow.md`
3. **Iterate** - Adjust prompts and safe outputs based on trial results
4. **Compare variants** - Run multiple versions side-by-side
5. **Validate** - Use `--repeat` to test consistency
6. **Deploy** - Once satisfied, commit to your repository

### Testing Strategy

**Unit testing** - Test individual workflows in isolation:
```bash
gh aw trial ./workflows/triage.md --delete-host-repo-after
```

**Integration testing** - Test workflows with actual repository content:
```bash
gh aw trial ./workflows/code-review.md --clone-repo myorg/real-repo
```

**Regression testing** - Verify changes don't break existing behavior:
```bash
# Before changes
gh aw trial ./workflow.md --host-repo regression-baseline

# After changes  
gh aw trial ./workflow.md --host-repo regression-test

# Compare results manually
```

**Performance testing** - Measure execution time and token usage:
```bash
gh aw trial ./workflow.md --repeat 5
# Check agentic_run_info in result files
```

### Prompt Engineering

Use trials to refine prompts iteratively:

1. **Baseline** - Run initial version: `gh aw trial v1.md`
2. **Hypothesis** - Modify prompt (e.g., add "be concise")
3. **Test** - Run modified version: `gh aw trial v2.md`
4. **Compare** - Review outputs side-by-side in trial repo
5. **Iterate** - Repeat until satisfied

### CI/CD Integration

Run trials in continuous integration:

```yaml
name: Test Workflows
on: [pull_request]

jobs:
  trial:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install gh-aw
        run: gh extension install githubnext/gh-aw
      - name: Trial workflow
        env:
          COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
        run: |
          gh aw trial ./.github/workflows/my-workflow.md \
            --delete-host-repo-after \
            --yes
```

### Cleanup Strategy

**Keep for analysis:**
```bash
gh aw trial ./workflow.md --host-repo analysis-2025-01
# Repository persists for later review
```

**Delete immediately:**
```bash
gh aw trial ./workflow.md --delete-host-repo-after
# Repository deleted after completion
```

**Periodic cleanup:**
```bash
# List trial repositories
gh repo list --limit 100 | grep gh-aw-trial

# Delete old trials manually
gh repo delete user/gh-aw-trial-old --yes
```

## Troubleshooting

### Workflow Not Found

**Error**: `workflow not found in repository`

**Solution**: Verify the workflow specification format:
```bash
# Correct formats
gh aw trial owner/repo/workflow-name
gh aw trial owner/repo/.github/workflows/workflow.md
gh aw trial ./local-workflow.md
```

### Workflow Dispatch Not Supported

**Error**: `workflow does not support workflow_dispatch trigger`

**Solution**: Add `workflow_dispatch` to the workflow frontmatter:
```yaml
on:
  workflow_dispatch:
  issues:
    types: [opened]
```

### Authentication Failures

**Error**: `authentication failed` or `API rate limit exceeded`

**Solution**: Ensure appropriate API keys are set:
```bash
# For Copilot
export COPILOT_GITHUB_TOKEN="your-token"

# For Claude
export CLAUDE_CODE_OAUTH_TOKEN="your-token"

# For OpenAI
export OPENAI_API_KEY="your-key"
```

Use `--use-local-secrets` to automatically push these secrets to the trial repository:
```bash
gh aw trial ./workflow.md --use-local-secrets
```

### Trial Repository Creation Failed

**Error**: `failed to create trial repository`

**Solution**: Check repository quotas and permissions:
```bash
# Verify GitHub authentication
gh auth status

# Check repository quota
gh api user | jq '{plan: .plan.name, private_repos: .plan.private_repos}'

# Try with explicit repository name
gh aw trial ./workflow.md --host-repo my-unique-trial-name
```

### Timeout Issues

**Error**: `workflow execution timed out`

**Solution**: Increase timeout duration:
```bash
gh aw trial ./workflow.md --timeout 60
# Timeout in minutes (default: 30)
```

### No Results Captured

**Issue**: Workflow completes but no issues/PRs created

**Solution**: Verify safe outputs are configured in the workflow:
```yaml
safe-outputs:
  create-issue:
    max: 10
  create-pull-request:
    max: 5
```

Check workflow logs in the trial repository's Actions tab for errors.

## Common Trial Patterns

### Pre-deployment Validation

Test workflows before deploying to production:

```bash
# Trial with production-like data
gh aw trial ./new-feature.md \
  --clone-repo myorg/production-repo \
  --host-repo pre-deployment-test

# Review results before deploying
gh repo view user/pre-deployment-test --web
```

### Prompt Optimization

A/B test different prompt strategies:

```bash
# Create variants
cp workflow.md workflow-detailed.md
cp workflow.md workflow-concise.md

# Edit prompts in each variant

# Compare side-by-side
gh aw trial \
  ./workflow-detailed.md \
  ./workflow-concise.md

# Review combined results
cat trials/combined-results.*.json | jq
```

### Workflow Documentation

Generate examples for documentation:

```bash
# Run workflow with clean repository
gh aw trial ./workflow.md \
  --force-delete-host-repo-before \
  --host-repo workflow-demo

# Share trial repository as live example
echo "Demo: https://github.com/user/workflow-demo"
```

### Debugging Production Issues

Reproduce issues in isolation:

```bash
# Clone production repository
gh aw trial ./workflow.md \
  --clone-repo myorg/production \
  --trigger-context "https://github.com/myorg/production/issues/456" \
  --host-repo debug-session

# Debug in trial repository
cd debug-session
# Make fixes, test locally
```

## Related Patterns

- **[SideRepoOps](/gh-aw/guides/siderepoops/)** - Run workflows from separate repositories targeting main codebases
- **[MultiRepoOps](/gh-aw/guides/multirepoops/)** - Coordinate workflows across multiple repositories
- **[Campaigns](/gh-aw/guides/campaigns/)** - Orchestrate multi-issue initiatives with AI-powered planning

## Related Documentation

- [CLI Commands](/gh-aw/setup/cli/) - Complete CLI reference including trial command
- [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) - Safe output configuration options
- [Workflow Triggers](/gh-aw/reference/triggers/) - Understanding workflow triggers including workflow_dispatch
- [Security Best Practices](/gh-aw/guides/security/) - Authentication and security for trials
