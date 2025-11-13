---
on:
  push:
    branches: [main]
permissions:
  contents: read
engine: copilot
tools:
  bash: ["echo", "ls"]
  github:
    toolsets: [default]
jobs:
  prepare:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Setup
        run: echo "Setup complete"
timeout-minutes: 10
---

# Multi-Job Workflow

A workflow with custom jobs and agent processing.
