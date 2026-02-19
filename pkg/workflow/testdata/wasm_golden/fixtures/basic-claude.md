---
name: basic-claude-test
description: Basic Claude engine workflow for wasm golden testing
on:
  workflow_dispatch:
permissions:
  contents: read
engine: claude
timeout-minutes: 10
network:
  allowed:
    - defaults
---

# Mission

Say hello to the world! This is a basic Claude engine workflow test.
