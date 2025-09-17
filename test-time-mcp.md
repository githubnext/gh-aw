---
engine: claude
on:
  workflow_dispatch:
tools:
  time:
    mcp-ref: "vscode"
    allowed: [current_time, get_timezone, set_timezone]
  github:
    allowed: [list_issues, create_issue]
---

# Time MCP Test Workflow

This workflow tests the mcp/time MCP server using VSCode configuration import.

Please:
1. Get the current time
2. Check the current timezone 
3. Create a summary of the current date and time information
4. Create a simple issue documenting the time information