---
on:
  workflow_dispatch:
permissions:
  contents: read
  actions: read
engine: copilot
safe-outputs:
  upload-artifact:
    max-uploads: 1
    allowed-paths:
      - "output/**"
---

# Test Copilot Upload Artifact

Test the `upload_artifact` safe output type with the Copilot engine.

## Task

Create a small text file at `output/result.txt` with the content "Hello from upload-artifact test" and upload it as a GitHub Actions artifact named "test-artifact".

Output results in JSONL format using the `upload_artifact` tool.
