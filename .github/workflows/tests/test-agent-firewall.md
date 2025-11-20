---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
network:
  firewall: true
  allowed:
    - defaults
imports:
  - ../../agents/technical-doc-writer.md
---

# Test Custom Agent with Firewall

This is a test workflow to verify that custom agents work correctly when AWF firewall is enabled.

Please analyze this repository's README.md file and provide a brief summary of what this project does.
