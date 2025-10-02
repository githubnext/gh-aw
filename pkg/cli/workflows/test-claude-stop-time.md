---
on:
  workflow_dispatch:
  stop-after: "+24h"
permissions:
  contents: read
engine: claude
---

# Test Claude Stop-Time

This is a test workflow to verify stop-time safety checks with Claude engine.

The workflow has a stop-after configuration that should create a dedicated stop_time_check job
with actions:write permission to disable the workflow if the deadline is reached.

Please analyze the current repository state and provide a brief summary.
