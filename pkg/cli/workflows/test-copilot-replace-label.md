---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
engine: copilot
safe-outputs:
  replace-label:
    max: 5
---

# Test Copilot Replace Label

Test the `replace_label` safe output type with the Copilot engine.

## Task

On issue #1, replace the label "in-progress" with the label "done".

Output results in JSONL format using the `replace_label` tool.
