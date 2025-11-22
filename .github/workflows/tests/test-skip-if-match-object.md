---
on:
  workflow_dispatch:
  skip-if-match:
    query: "is:pr is:open label:urgent"
    max: 3
engine: claude
description: Test workflow for skip-if-match object format with max threshold
---

# Skip-If-Match Object Format Test

This workflow demonstrates the object format of skip-if-match with a max threshold.

The workflow will be skipped if there are 3 or more open PRs with the "urgent" label.

This allows you to express conditions like "skip if 3 PRs match this request".
