---
description: Test runtime-import with GitHub Actions expressions
on: workflow_dispatch
engine: copilot
---

# Test Runtime Import with Expressions

This workflow tests that runtime-import can handle GitHub Actions expressions safely.

## Test 1: Import file with safe expressions

Content from imported file:
{{#runtime-import test-expressions.md}}

## Test 2: Verify expressions are rendered

The actor who triggered this workflow is: ${{ github.actor }}
The repository is: ${{ github.repository }}
The run ID is: ${{ github.run_id }}

## Instructions

Please verify that:
1. The imported file content appears above with expressions rendered
2. All safe expressions show actual values, not the raw expression syntax
3. The test passes successfully
