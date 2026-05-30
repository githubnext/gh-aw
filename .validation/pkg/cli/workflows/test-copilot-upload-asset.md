---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
safe-outputs:
  upload-asset:
    max: 1
timeout-minutes: 5
---

# Test Copilot Upload Asset

Test the `upload_asset` safe output type with the Copilot engine.

## Task

Find the latest release in this repository and upload a small test asset to it.
Create a file named `test-asset.txt` with the content `Test asset uploaded by automated test workflow.`
Then upload it as an asset to the latest release.

Output results in JSONL format using the `upload_asset` tool.
