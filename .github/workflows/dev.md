---
on: 
  workflow_dispatch:
name: Dev
description: Test workflow for development and experimentation purposes
timeout-minutes: 5
strict: false
# Using Codex engine for better error messages
engine: codex
permissions:
  contents: read
  issues: read
  pull-requests: read
  discussions: read
imports:
  - shared/pr-data-safe-input.md
tools:
  bash: ["*"]
  edit:
  github:
    toolsets: [default, repos, issues, discussions]
safe-outputs:
  assign-to-agent:
safe-inputs:
  test-js-math:
    description: "Test JavaScript math operations"
    inputs:
      a:
        type: number
        description: "First number"
        required: true
      b:
        type: number
        description: "Second number"
        required: true
    script: |
      // Users can write simple code without exports
      const sum = a + b;
      const product = a * b;
      return { sum, product, inputs: { a, b } };
  test-js-string:
    description: "Test JavaScript string operations"
    inputs:
      text:
        type: string
        description: "Input text"
        required: true
    script: |
      // Simple string manipulation
      return {
        original: text,
        uppercase: text.toUpperCase(),
        length: text.length
      };
---
Use the `fetch-pr-data` tool to fetch Copilot agent PRs from this repository using `search: "head:copilot/"`. Then compute basic PR statistics:
- Total number of Copilot PRs in the last 30 days
- Number of merged vs closed vs open PRs
- Average time from PR creation to merge (for merged PRs)
- Most active day of the week for PR creation

Also test the JavaScript safe-inputs tools:
1. Call `test-js-math` with a=5 and b=3 to verify math operations work
2. Call `test-js-string` with text="Hello World" to verify string operations work

Present the statistics and test results in a clear summary.