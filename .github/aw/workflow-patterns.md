---
description: Shared design patterns for command workflows, monitoring workflows, large-repository workflows, and cross-repository operations.
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

## Cross-Repository Pattern

For cross-repository reads and writes:

- enable the GitHub toolsets needed for external repos
- configure PAT or GitHub App auth in `safe-outputs:` when writing to another repo
- tell the agent to set `target-repo` explicitly for cross-repo outputs
- document the required token scopes in the workflow prompt or surrounding instructions

Cross-repository workflows still inherit the single-job constraints in [workflow-constraints.md](workflow-constraints.md).
