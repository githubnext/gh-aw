---
on:
  workflow_dispatch:
  workflow_run:
    workflows: ["*"]
    types: [completed]

safe-outputs:
  missing-tool:
  staged: true

engine:
  id: claude
permissions: read-all
---

Call the `missing-tool` tool and request the `draw pelican` tool, which does not exist, to trigger the `missing-tool` safe output.