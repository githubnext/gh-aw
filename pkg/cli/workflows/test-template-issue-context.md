---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
engine:
  id: copilot
tools:
  github:
    allowed: [get_issue, add_issue_comment]
---

# Test Template with Issue Context

Analyze issue #${{ github.event.issue.number }} in repository ${{ github.repository }}.

Created by: ${{ github.actor }}

{{#if ${{ github.event.issue.number }}}}
## Standard Analysis

Always perform this basic analysis:
- Review the issue description
- Identify the issue type
- Suggest next steps
{{/if}}

{{#if false}}
## Optional Advanced Analysis (Disabled)

This section is hidden and won't be included in the prompt.
{{/if}}

{{#if 1}}
## Additional Context

Truthy number condition - this section is included.
Provide comprehensive analysis with context.
{{/if}}

{{#if 0}}
## Debug Mode (Disabled)

Falsy number condition - this section is excluded.
{{/if}}

Add a comment to the issue with your analysis.
