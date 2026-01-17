---
on:
  issues:
    types: [opened]

# Powerless-by-default compiler story:
# - workflow permissions are read-only
# - any mutation must go through safe-outputs executor jobs
permissions:
  contents: read
  issues: read
  pull-requests: read

engine: copilot

safe-outputs:
  add-comment:
    max: 1
    target: issue

---

# Triage new issues

Read the issue and post a single helpful comment asking for missing details.
