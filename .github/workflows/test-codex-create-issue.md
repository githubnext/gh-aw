---
on:
  workflow_dispatch:

engine: 
  id: codex

safe-outputs:
  create-issue:
    title-prefix: "[codex-test] "
    labels: [codex, automation, haiku]
---

Create an issue with title "Hello from Codex" and body "World"

Add a haiku about GitHub Actions and AI to the issue body.