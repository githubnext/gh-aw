---
on: workflow_dispatch
permissions:
  contents: write
engine: copilot
tools:
  git-memory: true
---

# Test Git Memory

This is a test workflow to verify git-memory functionality.

Create a file called `test.txt` with the current date and time, then verify it persists.
