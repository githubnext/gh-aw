---
on: 
  workflow_dispatch:
name: Dev - Trace Capture Test
description: Test trace capture and replay functionality
timeout-minutes: 5
strict: false
engine: copilot

permissions:
  contents: read
  pull-requests: read

tools:
  edit:
  bash: ["*"]
imports:
  - shared/gh.md
safe-outputs:
  create-pull-request:
    allow-empty: true
---

# Trace Capture Test Workflow

This workflow tests the universal checkpoint capture and replay system.

## Test Steps

1. List recent workflow runs (tool call checkpoint)
2. Check repository status (tool call checkpoint)
3. Analyze a file (edit/read checkpoint)
4. Create a test PR (safe-output checkpoint)

## Instructions

Execute the following steps to generate checkpoints:

1. Run `gh run list --limit 3` to list recent runs
2. Run `git status` to check repo state
3. Read and analyze the AGENTS.md file structure
4. Create a PR with title "Test: Trace capture validation" on branch "test/trace-capture"

After completion, check the Job Summary for the checkpoint timeline with replay commands.
