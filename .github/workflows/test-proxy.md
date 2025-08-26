---
on:
  pull_request:
    branches: [ "main" ]
  workflow_dispatch:

output:
  issue_comment: {}

tools:
  fetch:
    mcp:
      type: stdio
      container: mcp/fetch
      permissions:
        network:
          allowed: 
            - "example.com"
    allowed: 
      - "fetch"
  
  github:
    allowed:
      - "get_issue"

engine: claude
runs-on: ubuntu-latest
---

# Test Network Permissions

## Task Description

Test the MCP network permissions feature to validate that domain restrictions are properly enforced.

- Use the fetch tool to successfully retrieve content from `https://example.com/` (the only allowed domain)
- Attempt to access blocked domains and verify they fail with network errors:
  - `https://httpbin.org/json` 
  - `https://api.github.com/user`
  - `https://www.google.com/`
  - `http://malicious-example.com/`
- Verify that all blocked requests fail at the network level (proxy enforcement)
- Confirm that only example.com is accessible through the Squid proxy

## Reporting Instructions

If there are any failures, security issues, or unexpected behaviors:

- Write a detailed report to ${{ env.GITHUB_AW_OUTPUT }} documenting:
- Which domains were successfully accessed vs blocked
- Error messages received for blocked domains  
- Any security observations or recommendations
- Specific failure details that need attention

The test results will be automatically posted as a comment on PR #${{ github.event.pull_request.number }}.
