---
on:
  workflow_dispatch:

engine: 
  id: claude

safe-outputs:
  create-pull-request:
    title-prefix: "[claude-test] "
    labels: [claude, automation, bot]
---

Add a file "TEST.md" with content "Hello, World!"

Create a pull request with title "Hello" and body "World"

Add a haiku about GitHub Actions and AI to the PR body.