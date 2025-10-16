---
on: 
  command:
    name: dev
  stop-after: "2025-11-16 00:00:00"
name: Dev
engine: copilot
permissions:
  contents: read
  actions: read
imports:
  - shared/mcp/drain3.md
safe-outputs:
  staged: true
  create-issue:
---

# Drain3 Log Pattern Mining Test

Test the drain3 MCP server by:
1. Creating a sample log file with test log entries
2. Using the index_file tool to analyze the log file
3. Using the list_templates tool to see the extracted patterns
4. Using the query_file tool to match a specific log line

Provide a summary of the templates found and demonstrate the query functionality.
Then write a brief poem about log pattern mining and publish it as an issue.
