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
    create-issue:
---
# Dev

Generate a comprehensive table of all accessible tools in the current chat session, detailing their names, descriptions, and usage examples. Post result in an issue.
