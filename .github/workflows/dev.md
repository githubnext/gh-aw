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

Generate a comprehensive table of all accessible tools in the current chat session, detailing their names, descriptions, and usage examples.
