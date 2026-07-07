---
on:
  workflow_dispatch:
permissions:
  contents: read
  security-events: read
engine: copilot
safe-outputs:
  create-code-scanning-alert:
    max: 5
---

# Test Copilot Create Code Scanning Alert

Test the `create_code_scanning_alert` safe output type with the Copilot engine.

## Task

Create a code scanning alert (SARIF report) for the current repository identifying a test vulnerability in `src/main.go` at line 1 with rule ID "test-rule" and message "Test code scanning alert created by automated test workflow."

Output results in JSONL format using the `create_code_scanning_alert` tool.
