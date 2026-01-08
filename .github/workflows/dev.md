---
on: 
  workflow_dispatch:
name: Dev
description: Say hello
timeout-minutes: 5
strict: true
engine: copilot

permissions:
  contents: read

tools:
  github:
---

{{#runtime-import .github/say-hello.md}}