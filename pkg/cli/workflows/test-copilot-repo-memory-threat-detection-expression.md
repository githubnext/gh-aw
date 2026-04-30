---
description: Test workflow combining repo-memory with expression-controlled threat detection
on:
  workflow_call:
    inputs:
      enable-threat-detection:
        description: 'Whether to enable threat detection at runtime'
        type: boolean
        default: true
      task:
        description: 'Task to store in memory'
        type: string
        default: 'Record this run'

permissions: read-all

engine: copilot

tools:
  repo-memory:
    branch-name: memory/test-threat-detection-expr
    description: "Test repo-memory with expression-controlled threat detection"
    max-file-size: 524288
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

# Test Repo Memory with Expression-Controlled Threat Detection

This workflow demonstrates `repo-memory` combined with expression-controlled threat detection.
The caller controls whether detection runs by passing `enable-threat-detection`.

The compiled output must contain:
- `detection` job with `if:` referencing `inputs.enable-threat-detection`
- `push_repo_memory` job depending on `detection`
- `push_repo_memory` condition using `always()` and accepting detection `skipped`
  so that the memory is persisted even when detection is skipped at runtime

Steps:
1. Check existing files in `/tmp/gh-aw/repo-memory-default/memory/default/`
2. Append a new entry: "Run ${{ github.run_number }}: ${{ inputs.task }}"
3. Report a summary in a new issue
