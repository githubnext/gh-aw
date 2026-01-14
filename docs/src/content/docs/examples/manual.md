---
title: Manual Workflows
description: On-demand workflows triggered manually via workflow_dispatch - research, analysis, and tasks you run when needed
sidebar:
  order: 4
---

Manual workflows run only when explicitly triggered via the GitHub Actions UI or CLI. They're perfect for on-demand tasks like research, analysis, or operations that need human judgment about timing.

## When to Use Manual Workflows

- **On-demand research**: Search and analyze topics as needed
- **Manual operations**: Tasks requiring human judgment on timing
- **Testing and debugging**: Run workflows with custom inputs during development
- **One-time tasks**: Operations that don't fit a schedule or event trigger
- **Interactive workflows**: Tasks that require parameters or choices at runtime

## Basic Manual Trigger

The simplest manual workflow uses `workflow_dispatch` without inputs:

```aw wrap
---
on:
  workflow_dispatch:
permissions:
  contents: read
safe-outputs:
  create-issue:
    title-prefix: "[analysis] "
---

# Repository Analysis

Analyze the repository structure and create an issue with findings.
```

**Trigger this workflow:**
- CLI: `gh aw run workflow-name`
- GitHub UI: Actions tab → Select workflow → "Run workflow" button

## Example Manual Triggers

### Text Input

Use `type: string` for free-form text inputs:

```yaml
on:
  workflow_dispatch:
    inputs:
      topic:
        description: 'Research topic'
        required: true
        type: string
      depth:
        description: 'Analysis depth (brief or detailed)'
        required: false
        default: 'brief'
        type: string
```

### Choice Input

Use `type: choice` for predefined options:

```yaml
on:
  workflow_dispatch:
    inputs:
      severity:
        description: 'Issue severity level'
        required: false
        type: choice
        options:
          - low
          - medium
          - high
        default: medium
```

### Boolean Input

Use `type: boolean` for yes/no options:

```yaml
on:
  workflow_dispatch:
    inputs:
      create_issue:
        description: 'Create an issue with results'
        required: false
        type: boolean
        default: true
```

### Number Input

Use `type: number` for numeric inputs:

```yaml
on:
  workflow_dispatch:
    inputs:
      max_results:
        description: 'Maximum number of results'
        required: false
        type: number
        default: 10
```

## Complete Working Example

Here's a complete manual workflow with multiple input types:

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
          - comprehensive
        default: brief
      create_issue:
        description: 'Create an issue with findings'
        type: boolean
        default: true
      max_sources:
        description: 'Maximum number of sources to analyze'
        type: number
        default: 5
permissions:
  contents: read
network:
  allowed:
    - defaults
safe-outputs:
  create-discussion:
    title-prefix: "[research] "
    labels: [research, documentation]
---

# Research Assistant

Research the topic: "${{ github.event.inputs.topic }}"

## Parameters

- **Analysis depth**: ${{ github.event.inputs.depth }}
- **Maximum sources**: ${{ github.event.inputs.max_sources }}
- **Create issue**: ${{ github.event.inputs.create_issue }}

## Instructions

1. Search for information about the research topic
2. Analyze findings based on the specified depth level:
   - **brief**: Quick overview with key points (1-2 paragraphs)
   - **detailed**: Comprehensive analysis with examples (3-5 paragraphs)
   - **comprehensive**: In-depth research with citations and comparisons (full report)
3. Limit analysis to the specified maximum number of sources
4. If create_issue is true, create a GitHub discussion with findings
5. Include references and citations for all sources used

## Output Format

Structure your findings with:
- Executive summary
- Key findings
- Detailed analysis (based on depth parameter)
- Sources and references
- Recommendations or next steps
```

**To use this workflow:**

1. Save it as `.github/workflows/research.md`
2. Compile: `gh aw compile research`
3. Trigger with the GitHub UI or CLI:

```bash wrap
gh aw run research
```

Then fill in the input values in the GitHub Actions UI.

## Running Manual Workflows

### Via CLI

Run a workflow and provide inputs interactively:

```bash wrap
gh aw run research
```

The GitHub CLI will prompt you for input values if the workflow requires them.

### Via GitHub Actions UI

1. Navigate to your repository on GitHub
2. Click the **Actions** tab
3. Select your workflow from the left sidebar
4. Click the **"Run workflow"** dropdown button
5. Fill in any required inputs
6. Click **"Run workflow"** button to start execution

**Expected result:**
- Workflow run appears in the Actions tab
- Status updates as the workflow progresses
- Outputs appear in the workflow logs and as safe-outputs (issues, PRs, comments)

### Triggering with Inputs via CLI

For advanced usage, trigger workflows with specific inputs using GitHub CLI:

```bash wrap
gh workflow run research.yml \
  -f topic="GitHub Actions Security" \
  -f depth=detailed \
  -f create_issue=true \
  -f max_sources=10
```

> [!TIP]
> Workflow Not Listed
>
> If your workflow doesn't appear in the Actions UI:
> - Verify the `.lock.yml` file exists in `.github/workflows/`
> - Check that the workflow is committed to the default branch
> - Ensure GitHub Actions is enabled in repository settings
> - Try manually enabling the workflow in the Actions tab

## Common Use Cases

### Code Quality Analysis

Run on-demand code quality checks:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      directory:
        description: 'Directory to analyze'
        required: false
        default: 'src'
        type: string
      include_tests:
        description: 'Include test files in analysis'
        type: boolean
        default: false
permissions:
  contents: read
safe-outputs:
  create-issue:
    title-prefix: "[code-quality] "
    labels: [quality, analysis]
---

# Code Quality Analysis

Analyze code quality in the ${{ github.event.inputs.directory }} directory.

Include test files: ${{ github.event.inputs.include_tests }}

Report on:
- Code complexity
- Duplicate code
- Best practice violations
- Security concerns
- Suggested improvements
```

### Database Migration Dry Run

Test database migrations before applying:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      migration_version:
        description: 'Migration version to test'
        required: true
        type: string
      environment:
        description: 'Environment to test against'
        type: choice
        options:
          - staging
          - production
        default: staging
permissions:
  contents: read
safe-outputs:
  add-comment:
---

# Database Migration Dry Run

Test migration version ${{ github.event.inputs.migration_version }} 
against ${{ github.event.inputs.environment }} environment.

Perform a dry-run analysis and report:
- Schema changes
- Data transformations
- Potential issues
- Rollback plan
- Estimated duration
```

### Security Audit

Trigger security audits on-demand:

```aw wrap
---
on:
  workflow_dispatch:
    inputs:
      scope:
        description: 'Audit scope'
        type: choice
        options:
          - dependencies
          - secrets
          - permissions
          - all
        default: dependencies
      create_report:
        description: 'Create detailed report issue'
        type: boolean
        default: true
permissions:
  contents: read
  security-events: read
safe-outputs:
  create-issue:
    title-prefix: "[security] "
    labels: [security, audit]
---

# Security Audit

Perform a security audit with scope: ${{ github.event.inputs.scope }}

Create detailed report: ${{ github.event.inputs.create_report }}

Audit and report on:
- Vulnerable dependencies
- Exposed secrets or credentials
- Permission misconfigurations
- Security best practices
- Remediation recommendations
```

## Tips and Best Practices

### Input Validation

While GitHub validates input types, add instructions for the AI to validate inputs:

```aw wrap
# Research Assistant

Topic: "${{ github.event.inputs.topic }}"

**Before proceeding, validate:**
- Topic is not empty
- Topic is relevant to the repository context
- If validation fails, create a comment explaining the issue
```

### Default Values

Always provide sensible defaults for optional inputs:

```yaml
inputs:
  max_results:
    description: 'Maximum results'
    type: number
    default: 10    # Reasonable default
```

### Clear Descriptions

Write input descriptions that clearly explain what the input does and any format requirements:

```yaml
inputs:
  date_range:
    description: 'Date range to analyze (format: YYYY-MM-DD to YYYY-MM-DD)'
    required: true
    type: string
```

### Input Documentation

Document all inputs in your workflow markdown:

```aw wrap
# Workflow Name

## Parameters

- **topic** (required): The research topic to analyze
- **depth** (optional): Analysis depth - brief, detailed, or comprehensive (default: brief)
- **create_issue** (optional): Whether to create an issue with results (default: true)
```
