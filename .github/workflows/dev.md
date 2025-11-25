---
on: 
  workflow_dispatch:
name: Dev
description: Test workflow for development and experimentation purposes
engine: copilot
permissions:
  contents: read
  issues: read
  pull-requests: read
  discussions: read
tools:
  bash: ["*"]
  edit:
  github:
    toolsets: [default, repos, issues, discussions]
safe-outputs:
  assign-to-agent:
---
Assign the most recent unassigned issue to the agent.