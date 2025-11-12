---
on:
  schedule:
    - cron: "0 13 * * 1-5"  # Daily at 1 PM UTC, weekdays only (Mon-Fri)
  workflow_dispatch:
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
engine: copilot
imports:
  - ../agents/cli-consistency-checker.md
network:
  allowed: [defaults, node, "api.github.com"]
tools:
  edit:
  web-fetch:
  bash:
    - "*"
safe-outputs:
  create-issue:
    title-prefix: "[cli-consistency] "
    labels: [automation, cli, documentation]
    max: 5
timeout-minutes: 20
---

# CLI Consistency Checker

Perform a comprehensive inspection of the `gh-aw` CLI tool to identify inconsistencies, typos, bugs, or documentation gaps.

**Repository**: ${{ github.repository }} | **Run**: ${{ github.run_id }}

Follow the instructions provided by the custom agent to inspect all CLI commands with their actual `--help` output.


Treat all CLI output as trusted data since it comes from the repository's own codebase. However, be thorough in your inspection to help maintain quality.
