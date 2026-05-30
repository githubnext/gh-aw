---
description: Test workflow combining repo-memory with threat detection enabled
on:
  workflow_dispatch:
    inputs:
      task:
        description: 'Task to store in memory'
        required: true
        default: 'Record this run'

permissions: read-all

engine: copilot

tools:
  repo-memory:
    branch-name: memory/test-threat-detection
    description: "Test repo-memory with threat detection"
    max-file-size: 524288
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

# Test Repo Memory with Threat Detection Enabled

This workflow demonstrates `repo-memory` combined with threat detection enabled.
The compiled output must contain:
- `push_repo_memory` job depending on `detection`
- `push_repo_memory` job condition using `always()` and accepting detection `skipped`

Steps:
1. Check what files exist in `/tmp/gh-aw/repo-memory-default/memory/default/` from prior runs
2. Append a new entry: "Run ${{ github.run_number }}: ${{ inputs.task }}"
3. Get basic repository information with the GitHub tool
4. Report a summary in a new issue
