---
on:
  workflow_dispatch:
imports:
  - ./shared/safe-outputs-needs-import.md
safe-outputs:
  needs:
    - main_job
    - imported_job
jobs:
  main_job:
    runs-on: ubuntu-latest
    steps:
      - run: echo "main"
  imported_job:
    runs-on: ubuntu-latest
    steps:
      - run: echo "imported"
  shared_job:
    runs-on: ubuntu-latest
    steps:
      - run: echo "shared"
---

# Safe outputs needs imports fixture

Verify that safe-outputs.needs from imports is merged with top-level needs.
