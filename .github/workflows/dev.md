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
post-steps:
  - name: Upload MCP logs for debugging
    if: always()
    uses: actions/upload-artifact@v4
    with:
      name: mcp-logs
      path: /tmp/gh-aw/mcp-logs/
      retention-days: 7
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

## Error Handling

**IMPORTANT**: If the drain3 MCP server is not available or fails to start:
1. Check the MCP server logs at `/tmp/gh-aw/mcp-logs/drain3/server.log` to understand the failure
2. Check the curl test logs at `/tmp/gh-aw/mcp-logs/drain3/curl-test.log` for connection issues
3. **Give up gracefully** - Do NOT attempt to use drain3 tools if they are unavailable
4. Create an issue explaining:
   - That drain3 MCP server was not available
   - The failure reason from the logs (if accessible)
   - That the Go source code analysis could not be completed
   - Suggestions for fixing the drain3 server setup

## Task

**Step 1: Verify drain3 Availability**
- Check if drain3 tools are available by attempting to list them
- If drain3 is not available, follow the error handling steps above and exit gracefully

**Step 2: Analyze Go Source Files (only if drain3 is available)**
1. **Find Go Source Files**: Locate all `.go` files in the repository
2. **Index Files with Drain3**: Use the `index_file` tool to analyze each Go file and extract templates
3. **List Extracted Templates**: Use the `list_templates` tool to display all templates found
4. **Create Summary Issue**: Create an issue with:
   - Total number of Go files analyzed
   - Number of templates extracted
   - Top 10 most common templates
   - Insights about code patterns

Assign the issue to copilot.
