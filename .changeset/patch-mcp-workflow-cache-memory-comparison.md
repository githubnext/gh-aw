---
"gh-aw": patch
---

Update MCP server workflow for toolset comparison with cache-memory

Enhanced the github-mcp-tools-report workflow to track and compare changes to the GitHub MCP toolset over time. Added cache-memory configuration to enable persistent storage across workflow runs, allowing the workflow to detect new and removed tools since the last report. The workflow now loads previous tools data, compares it with the current toolset, and includes a changes section in the generated report.
