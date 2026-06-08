---
description: Shared design patterns for command workflows, monitoring workflows, large-repository workflows, database migration reviews, and cross-repository operations.
---

# Workflow Patterns

## Command Workflows

### Prefer `slash_command` when

- the action is conversational
- the user may pass arguments in the comment body
- the workflow should work across issues, pull requests, and discussions

### Prefer `label_command` when

- the action is one-shot and argument-free
- discoverability in the GitHub UI matters
- the workflow fits a label-driven process

### Combine both when

- the action is common enough to justify both invocation styles
- you want UI discoverability plus comment-based flexibility

See also: [triggers.md](triggers.md)

## Monitoring Workflows

### Use `workflow_run` when

- monitoring another GitHub Actions workflow in the **same repository**
- reacting to workflow completion or conclusion

Reusable incident-triage pattern:

- trigger: `on.workflow_run` for the named deployment or CI workflow
- permissions: include `actions: read`; keep main job read-only
- reads: fetch failed job logs/artifacts via GitHub tools
- output: summarize impact/root cause in `create-issue`; use `noop` when no incident action is needed

### Use `deployment_status` when

- monitoring an external deployment service that reports status back to GitHub

See also: [deployment-status.md](deployment-status.md)

## Large-Repository Improvement Pattern

For recurring maintenance in large repositories:

- use `cache-memory`
- process one package, module, or directory per run
- store the last processed item and rotate in round-robin order
- prefer smaller focused PRs over wide repository sweeps

See also: [memory.md](memory.md)

## Pre-Step Data Fetching Pattern

Use deterministic `steps:` when the workflow needs large external data before the agent runs.

Rules:

- write prepared files to `/tmp/gh-aw/agent/`
- trim large outputs before handing them to the agent
- set `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}` on every `gh` step
- add `permissions: actions: read` when downloading workflow logs or artifacts
- use `jq` to reduce JSON payload size before writing files

## PR Visual Regression Pattern

For pull-request UI validation and screenshot diffs:

- trigger: `pull_request`
- tools: `playwright` plus `cache-memory` for baseline metadata
- permissions: read-only repo/PR access in agent job
- output: `add-comment` with pass/fail summary and links to captured artifacts
- fallback: use `noop` when no UI-relevant changes are detected

## Database Migration Safety Pattern

For pull requests that add or modify database migration files:

- trigger: `pull_request` with `paths:` scoped to migration directories (e.g. `db/migrate/**`, `migrations/**`, `*.sql`)
- permissions: `contents: read`, `pull-requests: read`; keep agent job read-only
- reads: changed migration file content via GitHub tools
- output: `add-comment` with a safety summary flagging risky operations; use `noop` when no concerns are detected
- prompt: suggest migration best practices in the agent prompt

## Cross-Repository Pattern

For cross-repository reads and writes:

- enable the GitHub toolsets needed for external repos
- configure PAT or GitHub App auth in `safe-outputs:` when writing to another repo
- tell the agent to set `target-repo` explicitly for cross-repo outputs
- document the required token scopes in the workflow prompt or surrounding instructions

Cross-repository workflows still inherit the single-job constraints in [workflow-constraints.md](workflow-constraints.md).
