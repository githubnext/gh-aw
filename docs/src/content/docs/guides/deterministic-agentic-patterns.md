---
title: Deterministic & Agentic Patterns
description: Learn how to combine deterministic computation steps with agentic reasoning in GitHub Agentic Workflows for powerful hybrid automation.
sidebar:
  order: 6
---

GitHub Agentic Workflows enable you to mix deterministic computation steps with agentic AI reasoning in a single workflow. This hybrid approach combines the reliability of traditional GitHub Actions with the flexibility of AI agents, enabling sophisticated automation patterns like data preprocessing, custom trigger filtering, and post-processing of AI results.

## When to Use This Pattern

Combine deterministic and agentic steps when you need to:

- **Precompute data** to ground AI agents with structured context
- **Filter triggers** with custom logic before invoking AI
- **Preprocess inputs** to normalize or validate data for AI consumption
- **Post-process AI output** to perform deterministic operations on agentic results
- **Build multi-stage pipelines** with computation and reasoning phases
- **Integrate external systems** through deterministic API calls around agentic steps

## Architecture

The hybrid pattern works by defining deterministic jobs in the frontmatter alongside the agentic execution:

```text
┌────────────────────────┐
│  Deterministic Jobs    │
│  - Data fetching       │
│  - Preprocessing       │
│  - Custom validation   │
└───────────┬────────────┘
            │ artifacts/outputs
            ▼
┌────────────────────────┐
│   Agent Job (AI)       │
│   - Receives context   │
│   - Reasons & decides  │
│   - Generates output   │
└───────────┬────────────┘
            │ safe outputs
            ▼
┌────────────────────────┐
│  Safe Output Jobs      │
│  - GitHub API calls    │
│  - Custom actions      │
└────────────────────────┘
```

## Basic Example: Precomputation

Define deterministic steps in the frontmatter to prepare data for the AI agent:

```yaml wrap title=".github/workflows/release-highlights.md"
---
on:
  push:
    tags:
      - 'v*.*.*'
engine: copilot
safe-outputs:
  update-release:

steps:
  - name: Fetch release data
    env:
      GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    run: |
      set -e
      mkdir -p /tmp/gh-aw/release-data
      
      # Get release tag from push event
      RELEASE_TAG="${GITHUB_REF#refs/tags/}"
      echo "RELEASE_TAG=$RELEASE_TAG" >> "$GITHUB_ENV"
      
      # Fetch current release information
      gh release view "$RELEASE_TAG" \
        --json name,tagName,body \
        > /tmp/gh-aw/release-data/current_release.json
      
      # Get previous release for comparison
      PREV_TAG=$(gh release list --limit 2 \
        --json tagName --jq '.[1].tagName // empty')
      
      if [ -n "$PREV_TAG" ]; then
        # Fetch merged PRs between releases
        gh pr list --state merged --limit 1000 \
          --json number,title,author,labels \
          > /tmp/gh-aw/release-data/pull_requests.json
      fi
---

# Release Highlights Generator

Generate engaging release highlights for version `${RELEASE_TAG}`.

## Available Data

All data is precomputed in `/tmp/gh-aw/release-data/`:
- `current_release.json` - Release metadata
- `pull_requests.json` - Merged PRs since last release

Analyze the PRs, categorize changes, and use the update-release tool
to prepend highlights to the release notes.
```

The deterministic `steps:` execute first, preparing structured data that the AI agent can reliably access and analyze.

## Multi-Job Pattern

For complex workflows, define multiple deterministic jobs with dependencies:

```yaml wrap title=".github/workflows/static-analysis.md"
---
on:
  schedule: daily
engine: claude
safe-outputs:
  create-discussion:

jobs:
  pull-images:
    runs-on: ubuntu-latest
    steps:
      - name: Pull static analysis Docker images
        run: |
          docker pull ghcr.io/zizmorcore/zizmor:latest
          docker pull ghcr.io/boostsecurityio/poutine:latest
  
  run-analysis:
    needs: [pull-images]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      
      - name: Run security scanners
        run: |
          ./gh-aw compile --zizmor --poutine \
            2>&1 | tee /tmp/gh-aw/compile-output.txt
      
      - name: Upload analysis results
        uses: actions/upload-artifact@v6
        with:
          name: analysis-results
          path: /tmp/gh-aw/compile-output.txt

steps:
  - name: Download analysis results
    uses: actions/download-artifact@v6
    with:
      name: analysis-results
      path: /tmp/gh-aw/
---

# Static Analysis Report

Analyze the security scan results in `/tmp/gh-aw/compile-output.txt`.

Parse the findings, cluster by severity and type, generate fix suggestions,
and create a discussion with your comprehensive analysis.
```

Custom jobs run **before** the agent job and can pass data through:
- **Artifacts** - Files uploaded with `actions/upload-artifact`
- **Job outputs** - Variables defined with `$GITHUB_OUTPUT`
- **Environment variables** - Set in `$GITHUB_ENV` for downstream steps

## Custom Trigger Filtering

Use deterministic `steps:` to implement custom trigger logic:

```yaml wrap title=".github/workflows/smart-responder.md"
---
on:
  issues:
    types: [opened, edited]
engine: copilot
safe-outputs:
  add-comment:

steps:
  - name: Filter issues by criteria
    id: filter
    env:
      ISSUE_BODY: ${{ github.event.issue.body }}
      ISSUE_LABELS: ${{ toJson(github.event.issue.labels.*.name) }}
    run: |
      # Custom filtering logic
      if echo "$ISSUE_BODY" | grep -q "urgent"; then
        echo "should_respond=true" >> "$GITHUB_OUTPUT"
        echo "priority=high" >> "$GITHUB_OUTPUT"
      elif echo "$ISSUE_LABELS" | grep -q "question"; then
        echo "should_respond=true" >> "$GITHUB_OUTPUT"
        echo "priority=normal" >> "$GITHUB_OUTPUT"
      else
        echo "should_respond=false" >> "$GITHUB_OUTPUT"
      fi
  
  - name: Skip if filtered out
    if: steps.filter.outputs.should_respond != 'true'
    run: |
      echo "Issue does not meet response criteria"
      exit 1
---

# Smart Issue Responder

Respond to the issue: "${{ github.event.issue.title }}"

Priority level: ${{ steps.filter.outputs.priority }}

Provide a helpful response appropriate to the priority level.
```

## Post-Processing Pattern

Combine agentic reasoning with deterministic post-processing using custom safe output jobs:

```yaml wrap title=".github/workflows/code-review.md"
---
on:
  pull_request:
    types: [opened]
engine: copilot
tools:
  github:
    toolsets: [default]

safe-outputs:
  jobs:
    format-and-notify:
      description: "Format review results and send notifications"
      runs-on: ubuntu-latest
      inputs:
        review_summary:
          description: "Agent's review summary"
          required: true
          type: string
      steps:
        - name: Format review report
          env:
            SUMMARY: "${{ inputs.review_summary }}"
          run: |
            # Deterministic formatting
            echo "## 🤖 AI Code Review" > /tmp/report.md
            echo "" >> /tmp/report.md
            echo "$SUMMARY" >> /tmp/report.md
            echo "" >> /tmp/report.md
            echo "---" >> /tmp/report.md
            echo "*Reviewed by GitHub Agentic Workflows*" >> /tmp/report.md
        
        - name: Post as PR comment
          env:
            GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          run: |
            gh pr comment ${{ github.event.pull_request.number }} \
              --body-file /tmp/report.md
---

# Code Review Agent

Review the pull request changes and provide feedback.

After analysis, use the format-and-notify tool to post your review summary.
```

## Importing Shared Steps

Define reusable deterministic steps in shared files and import them:

```yaml wrap title=".github/workflows/shared/reporting.md"
---
# No engine or on: configuration - this is just shared steps
---

## Report Formatting

Structure your report with an overview followed by detailed content:

1. **Content Overview**: Start with 1-2 paragraphs summarizing key findings.

2. **Detailed Content**: Place details inside HTML `<details>` tags.

**Example format:**

```markdown
Brief overview paragraph introducing the report.

<details>
<summary><b>Full Report Details</b></summary>

## Detailed Analysis

Full report content with sections and tables.

</details>
```
```

Import and use in workflows:

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

Analyze the repository and create a discussion report.

Follow the report formatting guidelines from the imported instructions.
```

## Pattern Examples

Real-world examples from the gh-aw repository:

### Release Workflow
**File**: `.github/workflows/release.md`

**Pattern**: Multi-job deterministic pipeline with agentic highlights generation

```yaml
jobs:
  release:         # Build and publish binaries
  generate-sbom:   # Generate security manifests
  # Agent job runs after, generates release highlights
```

### Static Analysis Report
**File**: `.github/workflows/static-analysis-report.md`

**Pattern**: Pull Docker images → Run scanners → AI analyzes results

```yaml
steps:
  - Pull zizmor and poutine Docker images
  - Run ./gh-aw compile with security tools
  - Save output to /tmp/gh-aw/compile-output.txt
# Agent reads output, clusters findings, creates discussion
```

## Best Practices

### Data Preparation

**Store precomputed data in standard locations:**

```bash
mkdir -p /tmp/gh-aw/data
# Save structured data for agent
echo "$RESULT" > /tmp/gh-aw/data/analysis.json
```

**Use artifacts for large datasets:**

```yaml
- uses: actions/upload-artifact@v6
  with:
    name: dataset
    path: /tmp/dataset.json
```

### Job Dependencies

**Define explicit dependencies with `needs:`:**

```yaml
jobs:
  fetch-data:
    steps: [...]
  
  process-data:
    needs: [fetch-data]
    steps: [...]

# Agent job automatically depends on custom jobs
```

### Error Handling

**Validate data before agent execution:**

```yaml
steps:
  - name: Validate prerequisites
    run: |
      if [ ! -f /tmp/data.json ]; then
        echo "Error: Required data not found"
        exit 1
      fi
```

### Environment Variables

**Pass data to agents via environment variables:**

```yaml
steps:
  - name: Set context
    run: |
      echo "RELEASE_TAG=v1.0.0" >> "$GITHUB_ENV"
      echo "PR_COUNT=42" >> "$GITHUB_ENV"
```

**Reference in prompt:**

```markdown
Analyze release `${RELEASE_TAG}` which includes ${PR_COUNT} pull requests.
```

## Related Documentation

- [Custom Safe Outputs](/gh-aw/guides/custom-safe-outputs/) - Custom post-processing jobs
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) - Configuration options
- [Compilation Process](/gh-aw/reference/compilation-process/) - How jobs are orchestrated
- [Imports](/gh-aw/reference/imports/) - Sharing configurations across workflows
- [Templating](/gh-aw/reference/templating/) - Using GitHub Actions expressions
