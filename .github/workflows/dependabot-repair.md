---
description: Repair safe Dependabot PR failures locally inside a product repository.
on:
  pull_request:
    types: [opened, synchronize, reopened]

permissions: read-all

tools:
  github:
    toolsets: [default]

network:
  allowed: [defaults, node, python, go, java, ruby, rust]

#observability:
#  otlp:
#    endpoint: ${{ secrets.OTEL_EXPORTER_OTLP_ENDPOINT }}
#    headers: ${{ secrets.OTEL_EXPORTER_OTLP_HEADERS }}

safe-outputs:
  add-comment:
    max: 5
  update-issue:
    max: 5
  create-pull-request:
    max: 1
  noop:
    max: 1
source: githubnext/dependabot-campaign/.github/workflows/dependabot-repair.md@ff91842c074b23270d3a0e26a7206a251232a374
---

# Dependabot Local Repair

## Scope

Only act on PRs authored by `dependabot[bot]`.

If not, use `noop`.

## Mission

Inspect failing checks and attempt one minimal safe repair.

## Allowed Repairs

- lockfile updates
- import fixes
- config updates
- test/snapshot updates

## Forbidden

- business logic changes
- auth/crypto/db behavior
- deployment config
- secrets

## Safe-Out

If risky:
- label `agent:safe-out`
- label `needs-human-review`
- comment explanation

## Repair Rules

- max one attempt
- minimal diff
- no test deletion

## Repair PR

Title:
[dependabot-repair] Fix CI for PR #<number>

## Comment

### Dependabot Repair Review

Action: repaired | safe-out | no-op  
Checks reviewed: yes | no  
Repair PR: <link>

Summary: <explanation>  
Next Step: <action>
