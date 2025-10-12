---
on: 
  workflow_dispatch:
name: Smoke GenAIScript
imports:
  - shared/genaiscript.md
tools:
  github:
    toolset: [pull_requests]
    allowed:
      - list_pull_requests
      - get_pull_request
  safety-prompt: false
safe-outputs:
    staged: true
    create-issue:
      min: 1
timeout_minutes: 5
strict: true
---

Review the last 5 merged pull requests in this repository and post summary in an issue.