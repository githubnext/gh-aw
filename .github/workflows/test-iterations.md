---
on:
  workflow_dispatch:
    inputs:
      number:
        description: 'A number to work with'
        required: true
        type: number

engine:
  id: copilot
  iterations: 3

timeout-minutes: 5
---

# Test Iterative Loop

Count from 1 to {{ inputs.number }}, showing one number per iteration.

On each iteration, continue from where the previous iteration left off.
