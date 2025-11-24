---
description: Example workflow demonstrating git-memory feature for persistent storage using git branches
on: workflow_dispatch
permissions:
  contents: write
engine: copilot
tools:
  git-memory:
    branch: memory/example
    description: Example persistent storage branch
---

# Git Memory Example Workflow

This workflow demonstrates the git-memory feature which provides persistent storage using git branches.

## What to do

1. Check if a file called `counter.txt` exists in the working directory
2. If it exists, read the number, increment it, and write it back
3. If it doesn't exist, create it with the value `1`
4. Create a log entry in `history.log` with the timestamp and counter value

This demonstrates that data persists across workflow runs using the git-memory branch.
