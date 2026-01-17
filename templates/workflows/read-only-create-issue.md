---
on:
  workflow_dispatch:

permissions:
  contents: read
  issues: read
  pull-requests: read

engine: copilot

safe-outputs:
  create-issue:
    max: 1
    title-prefix: "[agentic] "

---

# Create an issue from a prompt

Ask the agent to draft a single issue from the workflow_dispatch inputs.
