---
on:
  workflow_dispatch:
  stop-after: "+48h"
engine: claude
roles: [admin, maintainer]
jobs:
  pre-activation:
    runs-on: ubuntu-latest
    steps:
      - name: Custom check
        run: echo "ok"
---

Test workflow with unsupported field in pre-activation
