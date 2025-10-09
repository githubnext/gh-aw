---
on: 
  workflow_dispatch:
name: Smoke GenAIScript
imports:
  - shared/genaiscript.md
tools:
  github:
safe-outputs:
    staged: true
    create-issue:
      min: 1
timeout_minutes: 10
strict: true
---

List the last 5 merged pull requests in this repository into an issue.
