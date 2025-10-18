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
  staged: true
  create-issue:
    assignees: copilot
imports:
  - shared/mcp/drain3.md
---

# Go Source Code Pattern Analysis

Use the drain3 tool to analyze Go source files in this repository and extract log templates.

## Diagnostic Information

To diagnose drain3 MCP server issues locally, use:

```bash
# Inspect all MCP servers in this workflow
gh aw mcp inspect dev

# Inspect only the drain3 server
gh aw mcp inspect dev --server drain3

# View detailed information about the index_file tool
gh aw mcp inspect dev --server drain3 --tool index_file

# Verbose output with connection details
gh aw mcp inspect dev --server drain3 -v
```

The drain3 MCP server provides three tools:
- **index_file**: Stream-mine templates from log files and persist snapshots
- **query_file**: Match log lines against previously indexed templates
- **list_templates**: List all extracted templates from indexed files

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
