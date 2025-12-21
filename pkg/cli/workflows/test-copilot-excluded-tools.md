---
name: Test Copilot Excluded Tools
description: Test workflow to verify --excluded-tools flag behavior
on: workflow_dispatch

engine: copilot

tools:
  github:
    mode: remote
  bash:
    - echo
    - git status
    - git push
  edit: null
---

# Test Excluded Tools Flag

This workflow tests the `--excluded-tools` flag introduced in Copilot CLI v0.0.370.

The workflow has access to all tools except those explicitly excluded.

Your task: 
1. Confirm you have access to most GitHub tools
2. Confirm you can run `echo` and `git status` commands
3. Confirm that dangerous operations are prevented
4. Demonstrate safe usage of available tools
