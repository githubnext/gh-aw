---
on: 
  workflow_dispatch:
  push:
    branches:
      - copilot/*
      - compiler*
name: Dev
engine: copilot
safe-outputs:
    staged: true
    create-issue:
timeout_minutes: 10
strict: true
---

Write a poem.