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

## High-Volume Triage and Escalation Pattern

For workflows that receive many similar events (issues, PR comments, CI failures, security alerts, dependency events):

- start with a narrow, cheap triage/classification pass
- detect known, duplicate, stale, or low-value cases first
- emit explicit `noop` or a safe output when triage is confident
- escalate to the main/frontier agent only when triage is uncertain or the case is genuinely new/high-value

Decision flow:

```text
IF cheap triage is confident that the event is known, duplicate, stale, or low-value THEN
  stop and emit the configured safe output or noop
ELSE
  escalate to the main agent
END IF
```

Use this with pull-context workflows: fetch targeted evidence on demand instead of pushing large raw logs or payloads into the initial prompt.

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

## QA Coverage Report Pattern

For pull-request QA coverage summaries (gaps, risks, and suggested test focus):

- trigger: `pull_request` (optionally scoped with `paths:` for product areas under test)
- tools: `github` (`gh-proxy`) for changed files, PR metadata, labels, and linked checks
- permissions: `contents: read`, `pull-requests: read`; keep agent job read-only
- output: `add-comment` with a concise coverage matrix and explicit untested/high-risk areas
- fallback: use `noop` when the change is non-testable (for example docs-only PRs)

## PM Stakeholder Digest Pattern

For recurring product/stakeholder digests (status, risks, and notable changes):

- trigger: fuzzy `schedule` (for example `weekly on mondays`)
- tools: `github` (`gh-proxy`), optional `cache-memory` for period-over-period continuity
- permissions: read-only in the agent job
- output: `create-issue` by default; use `create-discussion` only when explicitly requested
- prompt: require audience-aware language (PM/stakeholder-friendly summary first, details second)

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
