---
description: Test workflow combining safe-jobs with expression-controlled threat detection
on:
  workflow_call:
    inputs:
      enable-threat-detection:
        description: 'Whether to enable threat detection at runtime'
        type: boolean
        default: true

permissions:
  contents: read
  actions: read

engine: copilot

safe-outputs:
  threat-detection: ${{ inputs.enable-threat-detection }}
  jobs:
    summarize:
      runs-on: ubuntu-latest
      steps:
        - name: Print summary
          run: |
            if [ -f "$GH_AW_AGENT_OUTPUT" ]; then
              echo "Agent output available"
              cat "$GH_AW_AGENT_OUTPUT"
            else
              echo "No agent output found"
            fi

timeout-minutes: 10
---

# Test Safe Jobs with Expression-Controlled Threat Detection

This workflow demonstrates `safe-outputs.jobs` combined with expression-controlled threat detection.

When `enable-threat-detection` is `true` (default) the detection job runs before `safe_jobs`.
When `enable-threat-detection` is `false` the detection job is skipped and `safe_jobs`
still runs because its condition accepts detection `skipped`.

The compiled output must contain:
- `detection` job with `if:` referencing `inputs.enable-threat-detection`
- `safe_jobs` (the custom `summarize` job) depending on `detection`
- `safe_jobs` condition using `always()` and accepting detection `skipped`

Produce a short text output (one sentence describing the repository) using the summarize tool.
