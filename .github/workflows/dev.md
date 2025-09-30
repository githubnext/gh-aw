---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
      - collect-guards
engine: copilot
tools:
  github:
    allowed:
      - list_pull_requests
      - get_pull_request
safe-outputs:
    staged: true
    create-issue:
---
# Available Tools Report

Generate a comprehensive table of all accessible tools grouped by MCP server. 

## Instructions

1. List all MCP servers available in this workflow configuration
2. For each MCP server, list all tools that are accessible
3. Present the information in a clear markdown table format
4. Include tool descriptions where available

## Expected Format

Create a table like this:

### GitHub MCP Server
| Tool Name | Description |
|-----------|-------------|
| list_pull_requests | List all pull requests in the repository |
| get_pull_request | Get details of a specific pull request |
| ... | ... |

### Safe Outputs MCP Server
| Tool Name | Description |
|-----------|-------------|
| create_issue | Create a new GitHub issue from agent output |
| ... | ... |

## Additional Information

- Organize tools by MCP server
- Include brief descriptions
- Note any permission restrictions
- Highlight which tools are currently allowed in this workflow

Post the complete table as a new issue titled "Available MCP Tools - Workflow Configuration".