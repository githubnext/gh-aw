---
name: Test Experiments Bare Array Form
description: Integration test workflow — validates bare-array experiment form compiles correctly
on:
  schedule:
    - cron: daily
permissions:
  contents: read
engine: copilot
experiments:
  prompt_style: [concise, verbose]
  model_temp: [low, high]
---

Bare-array experiment test.
