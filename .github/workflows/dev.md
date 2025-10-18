---
on: 
  workflow_dispatch:
concurrency:
  group: dev-workflow-${{ github.ref }}
  cancel-in-progress: true
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
tools:
  edit:
  github:
safe-outputs:
  create-issue:
    assignees: copilot
imports:
  - shared/mcp/drain3.md
---

# Go Source Code Pattern Analysis

Use the drain3 tool to analyze Go source files in this repository and extract log templates.

## Task

1. **Find Go Source Files**: Locate all `.go` files in the repository
2. **Index Files with Drain3**: Use the `index_file` tool to analyze each Go file and extract templates
3. **List Extracted Templates**: Use the `list_templates` tool to display all templates found
4. **Create Summary Issue**: Create an issue with:
   - Total number of Go files analyzed
   - Number of templates extracted
   - Top 10 most common templates
   - Insights about code patterns

Assign the issue to copilot.
