---
description: Test workflow for expression-controlled threat detection via workflow_call inputs
on:
  workflow_call:
    inputs:
      enable-threat-detection:
        description: 'Whether to enable threat detection'
        type: boolean
        default: true
      detection-continue-on-error:
        description: 'Whether threat detection failures are non-blocking'
        type: boolean
        default: false

permissions: read-all

engine: copilot

safe-outputs:
  create-issue:
    title-prefix: "[bot] "
    labels: [automated]
    max: 1
  # Object form: exercises both 'enabled' and 'continue-on-error' as expressions.
  # The shorthand form (threat-detection: ${{ expr }}) is covered by the
  # *-threat-detection-expression.md fixtures and the integration tests.
  threat-detection:
    enabled: ${{ inputs.enable-threat-detection }}
    continue-on-error: ${{ inputs.detection-continue-on-error }}

timeout-minutes: 10
---

# Test Expression-Controlled Threat Detection

This workflow demonstrates parameterising `safe-outputs.threat-detection` at runtime
via `workflow_call` inputs so callers can toggle detection without separate workflow files.

When called with `enable-threat-detection: true` the detection job runs normally.
When called with `enable-threat-detection: false` the detection job is skipped and
`safe_outputs` still runs (its condition accepts `detection.result == 'skipped'`).

Create a brief summary issue noting whether threat detection was enabled or skipped.
