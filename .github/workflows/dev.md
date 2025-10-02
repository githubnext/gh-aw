---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot*
  stop-after: "2026-01-01 00:00:00"
engine: copilot
tools:
  github:
    allowed:
      - list_pull_requests
      - get_pull_request
  web-fetch:
safe-outputs:
    staged: true
    create-issue:
---
# Dev
1. List tools defined in the current chat session (do not run commands, I am asking about tools defined in the LLM). Just the names in a table, nothing else.
2. Fetch the content of https://example.com and show the first 200 characters of the response.
3. Post the results in an issue.
