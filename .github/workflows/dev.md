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
  - shared/mcp/gh-aw.md
tools:
  agentic-workflows:
safe-outputs:
  staged: true
  create-issue:
---

# Agentic Workflow Log Analysis with Drain3

This workflow demonstrates analyzing GitHub Actions workflow logs using Drain3 for pattern extraction.

## Task

1. Use the agentic-workflows tool to download logs from the last 24 hours:
   - Use the `logs` command with `--start-date -1d` to get logs from the last 24 hours
   - Save the logs to a file for analysis

2. Analyze the downloaded logs with drain3:
   - Use the `index_file` tool to extract log patterns from the downloaded logs
   - Use the `list_templates` tool to see all extracted templates
   - Use the `query_file` tool to match specific error patterns

3. Provide insights based on the analysis:
   - Identify the most common log patterns
   - Highlight any error or warning patterns
   - Suggest potential improvements or areas of concern
   - Write a brief summary and publish it as an issue

The agentic-workflows tool provides workflow introspection capabilities, while drain3 extracts meaningful patterns from the log data.
