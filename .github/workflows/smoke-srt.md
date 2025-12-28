---
description: Smoke test workflow for Sandbox Runtime (SRT) - validates SRT functionality with Copilot
features:
  sandbox-runtime: true
on:
  workflow_dispatch:
  pull_request:
    types: [labeled]
    names: ["test-srt"]
permissions:
  contents: read
  issues: read
  pull-requests: read
name: Smoke SRT
engine: copilot
network:
  allowed:
    - defaults
    - github
    - node
    - "*.githubcopilot.com"
    - "example.com"
sandbox:
  type: sandbox-runtime
  config:
    filesystem:
      denyRead: []
      allowWrite:
        - "."
        - "/tmp"
        - "/home/runner/.copilot"
        - "/home/runner"
      denyWrite: []
    enableWeakerNestedSandbox: true
tools:
  bash:
  github:
safe-outputs:
  messages:
    footer: "> ⚓ *Logged in the captain's journal by [{workflow_name}]({run_url})*"
    run-started: "⚓ Ahoy! [{workflow_name}]({run_url}) sets sail on this {event_type}! All hands on deck, me hearties! 🏴‍☠️"
    run-success: "🏴‍☠️ Yo ho ho! [{workflow_name}]({run_url}) has claimed the treasure! The voyage be a SUCCESS! ⚓"
    run-failure: "🏴‍☠️ Blimey! [{workflow_name}]({run_url}) {status}! We've hit rough waters, mateys..."
timeout-minutes: 5
strict: true
---

**IMPORTANT: Keep all outputs extremely short and concise. Use single-line responses where possible. No verbose explanations.**

Test the Sandbox Runtime (SRT) integration:

1. Run `echo "Hello from SRT!"` using bash
2. Check the current directory with `pwd`
3. List files in the current directory with `find . -maxdepth 1 -ls`

Output a **very brief** summary (max 3-5 lines): ✅ or ❌ for each test, overall status.
