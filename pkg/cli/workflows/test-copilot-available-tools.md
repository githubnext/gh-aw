---
name: Test Copilot Available Tools
description: Test workflow to verify --available-tools flag behavior
on: workflow_dispatch

engine: copilot

tools:
  github:
    mode: remote
    allowed:
      - get_file_contents
      - list_commits
  bash:
    - echo
    - ls
  edit: null
---

# Test Available Tools Flag

This workflow tests the `--available-tools` flag introduced in Copilot CLI v0.0.370.

The workflow should only have access to specified GitHub tools, bash commands, and edit functionality.

Your task: List the available tools and confirm you can only access:
- github(get_file_contents)
- github(list_commits)
- shell(echo)
- shell(ls)
- write

Then attempt to use an unavailable tool and confirm it's blocked.
