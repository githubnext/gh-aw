---
name: PR Triage Agent
description: Labels pull requests based on change type when opened or updated
on:
  pull_request:
    types: [opened, reopened, edited, synchronize, ready_for_review]
permissions: read-all
tools:
  github:
    toolsets: [default]
safe-outputs:
  add-labels:
    allowed: [bug, enhancement, documentation, refactor, dependencies, maintenance, automation, code-quality, ci, security, performance]
    max: 3
---

<!-- Edit the file linked below to modify the agent without recompilation. Feel free to move the entire markdown body to that file. -->
@./agentics/pr-triage-agent.md
