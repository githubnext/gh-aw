---
on:
  pull_request:
    branches: [ "main" ]
  workflow_dispatch:

permissions:
  issues: write # needed to write the output report to an issue

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
      - "create_issue"
      - "create_comment"
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

Create a GitHub issue with the test results, documenting:
- Which domains were successfully accessed vs blocked
- Error messages received for blocked domains  
- Confirmation that network isolation is working correctly
- Any security observations or recommendations

The test should demonstrate that MCP containers are properly isolated and can only access explicitly allowed domains through the network proxy.
