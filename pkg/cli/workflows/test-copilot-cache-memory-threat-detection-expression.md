---
description: Test workflow combining cache-memory with expression-controlled threat detection
on:
  workflow_call:
    inputs:
      enable-threat-detection:
        description: 'Whether to enable threat detection at runtime'
        type: boolean
        default: true
      task:
        description: 'Task to store in cache'
        type: string
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
  threat-detection: ${{ inputs.enable-threat-detection }}

timeout-minutes: 10
---

# Test Cache Memory with Expression-Controlled Threat Detection

This workflow demonstrates `cache-memory` combined with expression-controlled threat detection.
The caller controls whether detection runs by passing `enable-threat-detection`.

The compiled output must contain:
- `detection` job with `if:` referencing `inputs.enable-threat-detection`
- `actions/cache/restore` in the agent job (detection is present at compile time)
- `update_cache_memory` job depending on `detection`
- `update_cache_memory` condition using `always()` and requiring detection `success`
  so cache is only saved after detection actually runs and succeeds

Steps:
1. Check existing files in `/tmp/gh-aw/cache-memory/`
2. Store a new entry: "Run ${{ github.run_number }}: ${{ inputs.task }}"
3. Report a summary in a new issue
