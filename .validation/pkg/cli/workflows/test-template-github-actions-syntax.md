---
on:
  workflow_dispatch:
permissions:
  contents: read
engine:
  id: claude
---

# Test Template Rendering with GitHub Actions Syntax

This workflow tests template rendering with GitHub Actions expressions in conditions.

Repository: ${{ github.repository }}

{{#if true}}
## Standard Analysis

Always perform this analysis:
- Review the repository structure
- Identify key components
- Provide actionable insights
{{/if}}

{{#if false}}
## Optional Advanced Analysis (Disabled)

This section is hidden and won't be included in the prompt.
{{/if}}

## Workflow Information

- Run ID: ${{ github.run_id }}
- Run Number: ${{ github.run_number }}

Analyze the repository and provide insights.
