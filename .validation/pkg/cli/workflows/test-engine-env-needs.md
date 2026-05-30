---
name: Test Engine Env Needs Expression
on:
  workflow_dispatch:
permissions:
  contents: read
engine:
  id: copilot
  env:
    RECEIVED_VALUE: ${{ needs.provide_value_to_agent.outputs.provided_value }}
strict: false
jobs:
  provide_value_to_agent:
    runs-on: ubuntu-latest
    outputs:
      provided_value: ${{ steps.provide.outputs.provided_value }}
    steps:
      - id: provide
        run: echo "provided_value=hello" >> "$GITHUB_OUTPUT"
---

# Test Engine Env Needs Expression

This workflow tests that `engine.env` values containing `needs.<job>.outputs.*` expressions
cause the referenced custom job to be added as a direct dependency of the agent job.

The `provide_value_to_agent` job must appear in the agent job's `needs` list so that
`RECEIVED_VALUE` evaluates correctly at runtime.

Please echo the value of the `RECEIVED_VALUE` environment variable.
