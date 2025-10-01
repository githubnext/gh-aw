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
# Dev
List tools defined in the current chat session (do not run commands, I am asking about tools defined in the LLM). Just the names in a table, nothing else.
Post result in an issue.
