---
name: basic-codex-test
description: Basic Codex engine workflow for wasm golden testing
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
timeout-minutes: 10
network:
  allowed:
    - defaults
---

# Mission

Say hello to the world! This is a basic Codex engine workflow test.
