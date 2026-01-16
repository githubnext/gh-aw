---
title: Deterministic & Agentic Patterns
description: Learn how to combine deterministic computation steps with agentic reasoning in GitHub Agentic Workflows for powerful hybrid automation.
sidebar:
  order: 6
---

GitHub Agentic Workflows combine deterministic computation with AI reasoning. This enables data preprocessing, custom trigger filtering, and post-processing patterns.

## When to Use This Pattern

Use deterministic steps with AI agents to:

- Precompute data to ground AI with structured context
- Filter triggers with custom logic
- Preprocess inputs before AI consumption
- Post-process AI output deterministically
- Build multi-stage computation and reasoning pipelines

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

## Basic Example: Precomputation

Prepare data for the AI agent:

```yaml wrap title=".github/workflows/release-highlights.md"
---
on:
  push:
    tags:
      - 'v*.*.*'
engine: <your-engine>  # See /gh-aw/reference/engines/ for options
safe-outputs:
  update-release:

steps:
  - name: Fetch release data
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      gh release view "${GITHUB_REF#refs/tags/}" --json name,tagName,body > /tmp/gh-aw/agent/release.json
      gh pr list --state merged --limit 100 --json number,title,labels > /tmp/gh-aw/agent/prs.json
---

# Release Highlights Generator

Generate engaging release highlights for version `${GITHUB_REF#refs/tags/}`.

The agent has access to precomputed data in `/tmp/gh-aw/agent/`:
- `release.json` - Release metadata
- `prs.json` - Merged PRs

Analyze the PRs, categorize changes, and use the update-release tool
to prepend highlights to the release notes.
```

Files in `/tmp/gh-aw/agent/` are automatically uploaded as workflow artifacts, making them available to the AI agent and subsequent jobs.

## Multi-Job Pattern

Define multiple deterministic jobs with dependencies:

```yaml wrap title=".github/workflows/static-analysis.md"
---
on:
  schedule: daily
engine: <your-engine>  # See /gh-aw/reference/engines/ for options
safe-outputs:
  create-discussion:

jobs:
  run-analysis:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - run: ./gh-aw compile --zizmor --poutine > /tmp/gh-aw/agent/analysis.txt

steps:
  - name: Download analysis
    uses: actions/download-artifact@v6
    with:
      name: analysis-results
      path: /tmp/gh-aw/
---

# Static Analysis Report

Parse the findings in `/tmp/gh-aw/agent/analysis.txt`, cluster by severity, 
and create a discussion with fix suggestions.
```

Custom jobs pass data through artifacts, job outputs, or environment variables.

## Custom Trigger Filtering

Use deterministic `steps:` for custom trigger logic:

```yaml wrap title=".github/workflows/smart-responder.md"
---
on:
  issues:
    types: [opened, edited]
engine: <your-engine>  # See /gh-aw/reference/engines/ for options
safe-outputs:
  add-comment:

steps:
  - name: Filter issues
    id: filter
    run: |
      if echo "${{ github.event.issue.body }}" | grep -q "urgent"; then
        echo "priority=high" >> "$GITHUB_OUTPUT"
      else
        exit 1
      fi
---

# Smart Issue Responder

Respond to urgent issue: "${{ github.event.issue.title }}"

Priority: ${{ steps.filter.outputs.priority }}
```

## Post-Processing Pattern

Use custom safe output jobs for deterministic post-processing:

```yaml wrap title=".github/workflows/code-review.md"
---
on:
  pull_request:
    types: [opened]
engine: <your-engine>  # See /gh-aw/reference/engines/ for options

safe-outputs:
  jobs:
    format-and-notify:
      description: "Format and post review"
      runs-on: ubuntu-latest
      inputs:
        summary:
          required: true
          type: string
      steps:
        - run: |
            echo "## 🤖 AI Code Review\n\n${{ inputs.summary }}" > /tmp/report.md
            gh pr comment ${{ github.event.pull_request.number }} --body-file /tmp/report.md
          env:
            GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

# Code Review Agent

Review the pull request and use the format-and-notify tool to post your summary.
```

## Importing Shared Steps

Define reusable steps in shared files:

```yaml wrap title=".github/workflows/shared/reporting.md"
---
---

## Report Formatting

Structure reports with an overview followed by expandable details:

```markdown
Brief overview paragraph.

<details>
<summary><b>Full Details</b></summary>

Detailed content here.

</details>
```
```

Import in workflows:

```yaml wrap title=".github/workflows/analysis.md"
---
on:
  schedule: daily
engine: <your-engine>  # See /gh-aw/reference/engines/ for options
imports:
  - shared/reporting.md
safe-outputs:
  create-discussion:
---

# Daily Analysis

Follow the report formatting guidelines from the imported instructions.
```

## Pattern Examples

### Release Workflow
`.github/workflows/release.md` - Multi-job pipeline with AI highlights generation

```yaml
jobs:
  release:         # Build binaries
  generate-sbom:   # Security manifests
  # Agent generates release highlights
```

### Static Analysis Report
`.github/workflows/static-analysis-report.md` - Run scanners then AI analysis

```yaml
steps:
  - Run ./gh-aw compile with security tools
  - Save to /tmp/gh-aw/agent/analysis.txt
# Agent clusters findings, creates discussion
```

## Agent Data Directory

The `/tmp/gh-aw/agent/` directory is the standard location for sharing data with AI agents:

```yaml
steps:
  - name: Prepare data
    run: |
      gh api repos/${{ github.repository }}/issues > /tmp/gh-aw/agent/issues.json
      gh api repos/${{ github.repository }}/pulls > /tmp/gh-aw/agent/pulls.json
```

**Key features:**
- Files in this directory are automatically uploaded as workflow artifacts
- The agent has read access to all files in `/tmp/gh-aw/agent/`
- Use for JSON data, text files, or any structured content the agent needs
- Directory is created automatically by the workflow runtime

**Example prompt reference:**

```markdown
Analyze the issues in `/tmp/gh-aw/agent/issues.json` and pull requests 
in `/tmp/gh-aw/agent/pulls.json`. Summarize the top 5 most active threads.
```

## Best Practices

Store data in `/tmp/gh-aw/agent/` for automatic artifact upload:

```bash
gh api repos/${{ github.repository }}/issues > /tmp/gh-aw/agent/issues.json
```

Define job dependencies with `needs:`:

```yaml
jobs:
  fetch-data:
    steps: [...]
  process-data:
    needs: [fetch-data]
    steps: [...]
```

Pass data via environment variables:

```yaml
steps:
  - run: echo "RELEASE_TAG=v1.0.0" >> "$GITHUB_ENV"
```

Reference in prompts: `Analyze release ${RELEASE_TAG}`.

## Related Documentation

- [Custom Safe Outputs](/gh-aw/guides/custom-safe-outputs/) - Custom post-processing jobs
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - Configuration options
- [Compilation Process](/gh-aw/reference/compilation-process/) - How jobs are orchestrated
- [Imports](/gh-aw/reference/imports/) - Sharing configurations across workflows
- [Templating](/gh-aw/reference/templating/) - Using GitHub Actions expressions
