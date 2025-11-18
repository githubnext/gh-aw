---
description: Smoke test workflow for Sandbox Runtime (SRT) - validates SRT functionality with Copilot
on:
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["test-srt"]
permissions:
  contents: read
  issues: read
name: Smoke SRT
engine: copilot
network:
  allowed:
    - defaults
    - github
imports:
  - shared/sandbox-runtime.md
tools:
  bash:
  github:
timeout-minutes: 5
strict: true
---

You are testing the Sandbox Runtime (SRT) integration. Perform the following tasks:

1. Run `echo "Hello from SRT!"` using bash
2. Check the current directory with `pwd`
3. List files in the current directory with `ls -la`

Report the results in a friendly summary. This is just a smoke test to validate that SRT is working correctly.
