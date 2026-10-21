---
name: Engine Conformance DeepSeek Harness
description: Run the shared configuration conformance suite with DeepSeek Harness.
on:
  workflow_dispatch:
permissions:
  contents: read
  copilot-requests: write
strict: true
inlined-imports: true
timeout-minutes: 10
# Include headroom for the engine's initial context and subsequent tool turns.
max-ai-credits: 50
features:
  gh-aw-detection: false
model: copilot/claude-haiku-5.5
engine:
  id: deepseek-harness
  env:
    ENGINE_CONFORMANCE_SENTINEL: conformance-deepseek-harness
imports:
  - shared/deepseek-harness.md
  - uses: shared/engine-conformance.md
    with:
      engine-id: deepseek-harness
  - shared/engine-conformance-worker.md
tools:
  work-queue:
    worker: true
    require-assignment: true
  github:
    mode: gh-proxy
    toolsets: [repos]
---

Execute the imported engine configuration conformance suite once.
