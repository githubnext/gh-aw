---
title: DispatchOps
description: Manually trigger and test agentic workflows with custom inputs using workflow_dispatch
sidebar:
  badge: { text: 'Manual', variant: 'tip' }
---

DispatchOps enables manual workflow execution via the GitHub Actions UI or CLI, perfect for on-demand tasks, testing, and workflows that need human judgment about timing. The `workflow_dispatch` trigger lets you run workflows with custom inputs whenever needed.

Use DispatchOps for research tasks, operational commands, testing workflows during development, debugging production issues, or any task that doesn't fit a schedule or event trigger.

## How Workflow Dispatch Works

Workflows with `workflow_dispatch` can be triggered manually rather than waiting for events like issues, pull requests, or schedules.

### Basic Syntax

Add `workflow_dispatch:` to the `on:` section in your workflow frontmatter:

```yaml
on:
  workflow_dispatch:
```

### With Input Parameters

Define inputs to customize workflow behavior at runtime:

```yaml
on:
  workflow_dispatch:
    inputs:
      topic:
        description: 'Research topic'
        required: true
        type: string
      priority:
        description: 'Task priority'
        required: false
        type: choice
        options:
          - low
          - medium
          - high
        default: medium
      deploy_target:
        description: 'Deployment environment'
        required: false
        type: environment
        default: staging
```

Supported input types: `string` (text), `boolean` (checkbox), `choice` (dropdown), `environment` (GitHub environments dropdown).

### Environment Input Type

The `environment` type populates automatically from repository Settings → Environments, returning the selected name as a string. A `default` may be specified (must match an existing environment name). Unlike `choice`, no `options` list is needed.

```yaml
on:
  workflow_dispatch:
    inputs:
      target_env:
        description: 'Deployment target'
        required: true
        type: environment
        default: staging
```

```markdown
Deploy to the ${{ github.event.inputs.target_env }} environment.
```

**Note:** The `environment` type does not enforce protection rules. To require approval gates, reviewers, or wait timers, use the `manual-approval:` field (see [Environment Approval Gates](#environment-approval-gates) below).

## Security Model

### Permission Requirements

Manual workflow execution respects the same security model as other triggers:

- **Repository permissions** - User must have write access or higher to trigger workflows
- **Role-based access** - Use the `roles:` field to restrict who can run workflows:

```yaml
on:
  workflow_dispatch:
roles: [admin, maintainer]
```

- **Bot authorization** - Use the `bots:` field to allow specific bot accounts:

```yaml
on:
  workflow_dispatch:
bots: ["dependabot[bot]", "github-actions[bot]"]
```

### Fork Protection

Unlike issue/PR triggers, `workflow_dispatch` only executes in the repository where it's defined-forks cannot trigger workflows in the parent repository. This provides inherent protection against fork-based attacks.

### Environment Approval Gates

Require manual approval before execution using GitHub environment protection rules:

```yaml
on:
  workflow_dispatch:
manual-approval: production
```

Configure approval rules, required reviewers, and wait timers in repository Settings → Environments. See [GitHub's environment documentation](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment) for setup details.

## Running Workflows from GitHub.com

### Via Actions Tab

1. Navigate to your repository on GitHub.com
2. Click the **Actions** tab
3. Select the workflow from the left sidebar
4. Click the **Run workflow** dropdown button
5. Select the branch to run from (default: main)
6. Fill in any required inputs
7. Click the **Run workflow** button

The workflow will execute immediately, and you can watch progress in the Actions tab.

### Finding Runnable Workflows

Only workflows with `workflow_dispatch:` appear in the "Run workflow" dropdown. If your workflow isn't listed:
- Verify `workflow_dispatch:` exists in the `on:` section
- Ensure the workflow has been compiled and pushed to GitHub
- Check that the `.lock.yml` file exists in `.github/workflows/`

## Running Workflows with CLI

The `gh aw run` command provides a faster way to trigger workflows from the command line.

### Basic Usage

```bash
gh aw run workflow
```

This matches workflows by filename prefix, validates `workflow_dispatch:`, and returns the run URL immediately.

### With Input Parameters

Pass inputs using the `--raw-field` or `-f` flag in `key=value` format:

```bash
gh aw run research --raw-field topic="quantum computing"
```

```bash
gh aw run scout \
  --raw-field topic="AI safety research" \
  --raw-field priority=high
```

### Wait for Completion

Monitor workflow execution and wait for results:

```bash
gh aw run research --raw-field topic="AI agents" --wait
```

`--wait` monitors progress in real-time and exits with a success/failure code on completion.

### Additional Options

```bash
gh aw run research --ref feature-branch              # Run from specific branch
gh aw run workflow --repo owner/repository           # Run in another repository
gh aw run research --raw-field topic="AI" --verbose  # Verbose output
```

## Declaring and Referencing Inputs

### Declaring Inputs in Frontmatter

```yaml
on:
  workflow_dispatch:
    inputs:
      analysis_depth:
        description: 'How deep should the analysis go?'
        required: true
        type: choice
        options:
          - surface
          - detailed
          - comprehensive
        default: detailed
      
      include_examples:
        description: 'Include code examples in the report'
        required: false
        type: boolean
        default: true
      
      max_results:
        description: 'Maximum number of results to return'
        required: false
        type: string
        default: '10'
```

### Referencing Inputs in Markdown

Access input values using GitHub Actions expression syntax:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      topic:
        description: 'Research topic'
        required: true
        type: string
      depth:
        description: 'Analysis depth'
        type: choice
        options:
          - brief
          - detailed
        default: brief
permissions:
  contents: read
safe-outputs:
  create-discussion:
---

# Research Assistant

Research the following topic: "${{ github.event.inputs.topic }}"

Analysis depth requested: ${{ github.event.inputs.depth }}

Provide a ${{ github.event.inputs.depth }} analysis with key findings and recommendations.
```

Reference inputs with `${{ github.event.inputs.INPUT_NAME }}`—values are interpolated at compile time throughout the workflow.

### Conditional Logic Based on Inputs

Use Handlebars conditionals to change behavior based on input values:

```markdown
{{#if (eq github.event.inputs.include_code "true")}}
Include actual code snippets in your analysis.
{{else}}
Describe code patterns without including actual code.
{{/if}}

{{#if (eq github.event.inputs.priority "high")}}
URGENT: Prioritize speed over completeness.
{{/if}}
```

## Development Pattern: Branch Testing

### Testing Workflow Changes

When developing workflows in a feature branch, add `workflow_dispatch:` for testing before merging to main:

```bash
# 1. Develop in feature branch
git checkout -b feature/improve-workflow
# Edit .github/workflows/research.md and add workflow_dispatch

# 2. Test in isolation first
gh aw trial ./research.md --raw-field topic="test query"

# 3. For in-repo testing, temporarily push to main
git checkout main
git cherry-pick <commit-sha>
git push origin main

# 4. Test from your branch
git checkout feature/improve-workflow
gh aw run research --ref feature/improve-workflow

# 5. Iterate, then create PR when satisfied
gh pr create --title "Improve workflow"
```

The workflow runs with your branch's code and state. Safe outputs (issues, PRs, comments) are created in your branch context. Use [trial mode](/gh-aw/patterns/trial-ops/) for completely isolated testing without affecting the production repository.

## Common Use Cases

**On-demand research:** Add a `topic` string input and trigger with `gh aw run research --raw-field topic="AI safety"` when needed.

**Manual operations:** Use a `choice` input with predefined operations (cleanup, sync, audit) to execute specific tasks on demand.

**Testing and debugging:** Add `workflow_dispatch` to event-triggered workflows (issues, PRs) with optional test URL inputs to test without creating real events.

**Scheduled workflow testing:** Combine `schedule` with `workflow_dispatch` to test scheduled workflows immediately rather than waiting for the cron schedule.

## Troubleshooting

**Workflow not listed in GitHub UI:** Verify `workflow_dispatch:` exists in the `on:` section, compile the workflow (`gh aw compile workflow`), and push both `.md` and `.lock.yml` files. The Actions page may need a refresh.

**"Workflow not found" error:** Use the filename without `.md` extension (`research` not `research.md`). Ensure the workflow exists in `.github/workflows/` and has been compiled.

**"Workflow cannot be run" error:** Add `workflow_dispatch:` to the `on:` section, recompile, and verify the `.lock.yml` includes the trigger before pushing.

**Permission denied:** Verify write access to the repository and check the `roles:` field in workflow frontmatter. For organization repos, confirm your org role.

**Inputs not appearing:** Check YAML syntax and indentation (2 spaces) in `workflow_dispatch.inputs`. Ensure input types are valid (`string`, `boolean`, `choice`, `environment`), then recompile and push.

**Wrong branch context:** Specify the branch explicitly with `--ref branch-name` in CLI or select the correct branch in the GitHub UI dropdown before running.

## Related Documentation

- [Manual Workflows Example](/gh-aw/examples/manual/) - Example manual workflows
- [Triggers Reference](/gh-aw/reference/triggers/) - Complete trigger syntax including workflow_dispatch
- [TrialOps](/gh-aw/patterns/trial-ops/) - Testing workflows in isolation
- [CLI Commands](/gh-aw/setup/cli/) - Complete gh aw run command reference
- [Templating](/gh-aw/reference/templating/) - Using expressions and conditionals
- [Security Best Practices](/gh-aw/introduction/architecture/) - Securing workflow execution
- [Quick Start](/gh-aw/setup/quick-start/) - Getting started with agentic workflows
