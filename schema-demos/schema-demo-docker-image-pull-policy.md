---
description: Demonstrates the `docker-image-pull-policy` schema field
on:
  workflow_dispatch:
permissions:
  contents: read
engine: codex
docker-image-pull-policy: always
timeout-minutes: 5
---

# Schema Demo: `docker-image-pull-policy`

This workflow demonstrates usage of the `docker-image-pull-policy` field in the
gh-aw frontmatter schema. It exists solely to achieve schema feature coverage.

## What `docker-image-pull-policy` Does

Policy for pre-start Docker images in agent and detection jobs. 'always' (default) pulls images; 'never' validates exact local RepoDigests and does not contact a registry.

This demo uses the default `always` policy.

## Task

Call `noop` — this is a coverage-only demo workflow.

**Important**: Always call the `noop` safe-output tool.

```json
{"noop": {"message": "Coverage demo for `docker-image-pull-policy` — no action needed."}}
```
