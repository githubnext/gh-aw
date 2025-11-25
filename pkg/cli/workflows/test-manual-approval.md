---
description: Test workflow for validating manual approval functionality in agentic workflows
on:
  workflow_dispatch:
  manual-approval: production
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
---

# Test Manual Approval Workflow

This workflow tests the manual-approval field in the on section.
