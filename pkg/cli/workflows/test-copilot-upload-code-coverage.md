---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  upload-code-coverage:
    max: 1
timeout-minutes: 5
---

# Test Copilot Upload Code Coverage

Test the `upload_code_coverage` safe output type with the Copilot engine.

## Task

Create a small coverage report file named `coverage.xml` with minimal valid coverage XML content,
then stage it for upload as a code coverage report.

Output results in JSONL format using the `upload_code_coverage` tool.
