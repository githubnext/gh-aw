---
description: Test workflow combining cache-memory with threat detection enabled
on:
  workflow_dispatch:
    inputs:
      task:
        description: 'Task to store in cache'
        required: true
        default: 'Cache this result'

permissions: read-all

engine: copilot

tools:
  cache-memory:
    retention-days: 7
  github:
    allowed: [get_repository]

safe-outputs:
  create-issue:
    title-prefix: "[bot] "
    labels: [automated]
    max: 1
  threat-detection: true

timeout-minutes: 10
---

# Test Cache Memory with Threat Detection Enabled

This workflow demonstrates `cache-memory` combined with standard threat detection.
When detection is enabled the compiled output must contain:
- `actions/cache/restore` (instead of `actions/cache`) in the agent job
- An `update_cache_memory` job that depends on `detection`
- `update_cache_memory` condition using `always()` and requiring detection `success`

Steps:
1. Check what files exist in `/tmp/gh-aw/cache-memory/` from previous runs
2. Store a new entry: "Run ${{ github.run_number }}: ${{ inputs.task }}"
3. Get basic repository information with the GitHub tool
4. Report a summary in a new issue
