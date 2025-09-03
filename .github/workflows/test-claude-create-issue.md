---
on:
  workflow_dispatch:

engine: 
  id: claude
strict: true
safe-outputs:
  create-issue:
    title-prefix: "[claude-test] "
    labels: [claude, automation, haiku]
---

Create an issue with title "Hello from Claude" and body "World"

Add a haiku about GitHub Actions and AI to the issue body.