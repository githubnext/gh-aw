---
description: Test safe-inputs gateway compilation with local binary
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
sandbox:
  safe-inputs:
    command: ./gh-aw mcp-gateway
    steps:
      - name: Build local gh-aw binary
        run: |
          echo "Building local gh-aw binary for testing..."
          make build
          ./gh-aw --version
safe-inputs:
  test-tool:
    description: "A test tool for gateway"
    inputs:
      message:
        type: string
        description: "A test message"
        required: true
    script: |
      return { result: message };
safe-outputs:
  create-issue:
    title-prefix: "Test Gateway"
timeout-minutes: 5
---

# Test Gateway Workflow with Local Binary

Test the safe-inputs gateway compilation with local binary.

Call the test-tool with message: "Hello Gateway!"
