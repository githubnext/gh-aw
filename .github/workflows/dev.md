---
on: 
  workflow_dispatch:
name: Dev
description: Find and assign documentation issues to technical doc writer agent
timeout-minutes: 10
strict: false
engine: claude
permissions:
  contents: read
  issues: write
  pull-requests: read
tools:
  github:
    toolsets: [default]
safe-outputs:
  assign-to-agent:
    max: 1
---
# Find and Assign Documentation Issues

Find issues that need documentation work and assign them to the technical documentation writer agent.

## Current Context

- **Repository**: ${{ github.repository }}
- **Actor**: @${{ github.actor }}
- **Run**: ${{ github.run_id }}

## Task

1. **Search for documentation issues**: Use the GitHub tools to search for open issues that need documentation work. Look for issues with labels like "documentation", "docs", or issues that describe documentation needs.

2. **Select a good candidate**: Pick an issue that clearly describes what documentation is needed. Good candidates:
   - Have clear requirements about what needs to be documented
   - Specify the location or scope of the documentation
   - Are not already assigned to someone

3. **Assign to technical doc writer**: Once you've identified a good documentation issue, use the `assign_to_agent` tool to assign it:

```
assign_to_agent(
  issue_number=<issue_number>,
  custom_agent="technical-doc-writer",
  custom_instructions="Read the description carefully. This assignment was made by @${{ github.actor }} in run ${{ github.run_id }}."
)
```

**Important**: The `custom_instructions` field supports GitHub expression syntax like `${{ github.actor }}` which will be rendered at runtime.