---
name: "dev"
on:
  workflow_dispatch: # do not remove this trigger
  push:
    branches:
      - copilot/*
safe-outputs:
  staged: true
  create-issue:
engine: codex
network:
  allowed: ["*"]
permissions: read-all
---

Summarize the description of https://microsoft.com/