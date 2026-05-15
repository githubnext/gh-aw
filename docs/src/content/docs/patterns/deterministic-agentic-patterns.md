---
title: Deterministic & Agentic Patterns
description: Learn how to combine deterministic computation steps with agentic reasoning in GitHub Agentic Workflows for powerful hybrid automation.
sidebar:
  order: 6
---

GitHub Agentic Workflows combine deterministic computation with AI reasoning, enabling data preprocessing, custom trigger filtering, and post-processing patterns.

## When to Use

Combine deterministic steps with AI agents to precompute data, filter triggers, preprocess inputs, post-process outputs, or build multi-stage computation and reasoning pipelines.

## Architecture

Define deterministic jobs in frontmatter alongside agentic execution:

```text
┌────────────────────────┐
│  Deterministic Jobs    │
│  - Data fetching       │
│  - Preprocessing       │
└───────────┬────────────┘
            │ artifacts/outputs
            ▼
┌────────────────────────┐
│   Agent Job (AI)       │
│   - Reasons & decides  │
└───────────┬────────────┘
            │ safe outputs
            ▼
┌────────────────────────┐
│  Safe Output Jobs      │
│  - GitHub API calls    │
└────────────────────────┘
```

## Precomputation Example

```yaml wrap title=".github/workflows/release-highlights.md"
---
on:
  push:
    tags: ['v*.*.*']
engine: copilot
safe-outputs:
  update-release:

steps:
  - run: |
      gh release view "${GITHUB_REF#refs/tags/}" --json name,tagName,body > /tmp/gh-aw/agent/release.json
      gh pr list --state merged --limit 100 --json number,title,labels > /tmp/gh-aw/agent/prs.json
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

# Release Highlights Generator

Generate release highlights for `${GITHUB_REF#refs/tags/}`. Analyze PRs in `/tmp/gh-aw/agent/prs.json`, categorize changes, and use update-release to prepend highlights to the release notes.
```

Files in `/tmp/gh-aw/agent/` are automatically uploaded as artifacts and available to the AI agent.

## Multi-Job Pattern

```yaml wrap title=".github/workflows/static-analysis.md"
---
on:
  schedule: daily
engine: claude
safe-outputs:
  create-discussion:

jobs:
  run-analysis:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - run: ./gh-aw compile --zizmor --poutine > /tmp/gh-aw/agent/analysis.txt

steps:
  - uses: actions/download-artifact@v6
    with:
      name: analysis-results
      path: /tmp/gh-aw/
---

# Static Analysis Report

Parse findings in `/tmp/gh-aw/agent/analysis.txt`, cluster by severity, and create a discussion with fix suggestions.
```

Pass data between jobs via artifacts, job outputs, or environment variables.

## Custom Trigger Filtering

### Inline Steps (`on.steps:`) — Preferred

Inject deterministic steps directly into the pre-activation job using `on.steps:`. This saves **one workflow job** compared to the multi-job pattern and is the recommended approach for lightweight filtering:

```yaml wrap title=".github/workflows/smart-responder.md"
---
on:
  issues:
    types: [opened]
  steps:
    - id: check
      env:
        LABELS: ${{ toJSON(github.event.issue.labels.*.name) }}
      run: echo "$LABELS" | grep -q '"bug"'
      # exits 0 (outcome: success) if the label is found, 1 (outcome: failure) if not
engine: copilot
safe-outputs:
  add-comment:

if: needs.pre_activation.outputs.check_result == 'success'
---

# Bug Issue Responder

Triage bug report: "${{ github.event.issue.title }}" and add-comment with a summary of the next steps.
```

Each step with an `id` gets an auto-wired output `<id>_result` set to `${{ steps.<id>.outcome }}` — `success` when the step's exit code is 0, `failure` when non-zero. Gate the workflow by checking `needs.pre_activation.outputs.<id>_result == 'success'`.

To pass an explicit value rather than relying on exit codes, set a step output and re-expose it via `jobs.pre-activation.outputs`:

```yaml wrap
jobs:
  pre-activation:
    outputs:
      has_bug_label: ${{ steps.check.outputs.has_bug_label }}

if: needs.pre_activation.outputs.has_bug_label == 'true'
```

When `on.steps:` need GitHub API access, use `on.permissions:` to grant the required scopes to the pre-activation job:

```yaml wrap
on:
  schedule: every 30m
  permissions:
    issues: read
  steps:
    - id: search
      uses: actions/github-script@v8
      with:
        script: |
          const open = await github.rest.issues.listForRepo({ ...context.repo, state: 'open' });
          core.setOutput('has_work', open.data.length > 0 ? 'true' : 'false');

jobs:
  pre-activation:
    outputs:
      has_work: ${{ steps.search.outputs.has_work }}

if: needs.pre_activation.outputs.has_work == 'true'
```

See [Pre-Activation Steps](/gh-aw/reference/triggers/#pre-activation-steps-onsteps) and [Pre-Activation Permissions](/gh-aw/reference/triggers/#pre-activation-permissions-onpermissions) for full documentation.

### Multi-Job Pattern — For Complex Cases

Use a separate `jobs:` entry when filtering requires heavy tooling (checkout, compiled tools, multiple runners):

```yaml wrap title=".github/workflows/smart-responder.md"
---
on:
  issues:
    types: [opened]
engine: copilot
safe-outputs:
  add-comment:

jobs:
  filter:
    runs-on: ubuntu-latest
    outputs:
      should-run: ${{ steps.check.outputs.result }}
    steps:
      - id: check
        env:
          LABELS: ${{ toJSON(github.event.issue.labels.*.name) }}
        run: |
          if echo "$LABELS" | grep -q '"bug"'; then
            echo "result=true" >> "$GITHUB_OUTPUT"
          else
            echo "result=false" >> "$GITHUB_OUTPUT"
          fi

if: needs.filter.outputs.should-run == 'true'
---

# Bug Issue Responder

Triage bug report: "${{ github.event.issue.title }}" and add-comment with a summary of the next steps.
```

The compiler automatically adds the filter job as a dependency of the activation job, so when the condition is false the workflow run is **skipped** (not failed), keeping the Actions tab clean.

### Simple Context Conditions

For conditions that can be expressed directly with GitHub Actions context, use `if:` without a custom job:

```yaml wrap
---
on:
  pull_request:
    types: [opened, synchronize]
engine: copilot
if: github.event.pull_request.draft == false
---
```

### Query-Based Filtering

For conditions based on GitHub search results, use [`skip-if-match:`](/gh-aw/reference/triggers/#skip-if-match-condition-skip-if-match) or [`skip-if-no-match:`](/gh-aw/reference/triggers/#skip-if-no-match-condition-skip-if-no-match) in the `on:` section — these accept standard [GitHub search query syntax](https://docs.github.com/en/search-github/searching-on-github/searching-issues-and-pull-requests) and are evaluated in the pre-activation job, producing the same skipped-not-failed behavior:

```yaml wrap
---
on:
  issues:
    types: [opened]
  # Skip if a duplicate issue already exists (GitHub search query syntax)
  skip-if-match: 'is:issue is:open label:duplicate'
engine: copilot
---
```

## Post-Processing Pattern

```yaml wrap title=".github/workflows/code-review.md"
---
on:
  pull_request:
    types: [opened]
engine: copilot

safe-outputs:
  jobs:
    format-and-notify:
      description: "Format and post review"
      runs-on: ubuntu-latest
      inputs:
        summary: {required: true, type: string}
      steps:
        - ...
---

# Code Review Agent

Review the pull request and use format-and-notify to post your summary.
```

## Importing Shared Instructions

Define reusable guidance in shared files and import them:

```yaml wrap title=".github/workflows/analysis.md"
---
on:
  schedule: daily
engine: copilot
imports:
  - shared/reporting.md
safe-outputs:
  create-discussion:
---

# Daily Analysis

Follow the report formatting guidelines from shared/reporting.md.
```

For daily discussion-based audit workflows, prefer `shared/daily-audit-base.md` to bundle discussion publishing, reporting guidance, and OTLP observability in a single import.

## Agent Data Directory

Use `/tmp/gh-aw/agent/` to share data with AI agents. Files here are automatically uploaded as artifacts and accessible to the agent:

```yaml
steps:
  - run: |
      gh api repos/${{ github.repository }}/issues > /tmp/gh-aw/agent/issues.json
      gh api repos/${{ github.repository }}/pulls > /tmp/gh-aw/agent/pulls.json
```

Reference in prompts: "Analyze issues in `/tmp/gh-aw/agent/issues.json` and PRs in `/tmp/gh-aw/agent/pulls.json`."

## Related Documentation

- [Pre-Activation Steps](/gh-aw/reference/triggers/#pre-activation-steps-onsteps) - Inline step injection into the pre-activation job
- [Pre-Activation Permissions](/gh-aw/reference/triggers/#pre-activation-permissions-onpermissions) - Grant additional scopes for `on.steps:` API calls
- [Custom Safe Outputs](/gh-aw/reference/custom-safe-outputs/) - Custom post-processing jobs
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - Configuration options
- [Compilation Process](/gh-aw/reference/compilation-process/) - How jobs are orchestrated
- [Imports](/gh-aw/reference/imports/) - Sharing configurations across workflows
- [Templating](/gh-aw/reference/templating/) - Using GitHub Actions expressions
