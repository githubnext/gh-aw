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
- reacting to workflow completion/conclusion

Incident-triage pattern:

- trigger: `on.workflow_run` for the named deployment/CI workflow
- permissions: include `actions: read`; main job read-only
- reads: failed job logs/artifacts via GitHub tools
- output: `create-issue` with impact/root cause; `noop` when no action needed

Compact `workflow_run` examples:

- **Deploy workflow failure triage**: trigger on `workflow_run` for `Deploy`, read failed jobs/logs/artifacts, create one incident issue, `noop` when rerun succeeds.
- **CI regression watcher**: trigger on `workflow_run` for `CI`, compare current failure against recent runs, create issue only for new regressions, `noop` for known flakes.

### Use `deployment_status` when

- monitoring an external deployment service reporting back to GitHub

Rule of thumb:

- `workflow_run` → GitHub Actions outcomes in this repo
- `deployment_status` → external platform outcomes via Deployments API

Single-job limits apply:

- triage, evidence collection, and summary in one agent job
- no multi-job fan-out/fan-in
- no cross-workflow waits or chaining

See also: [deployment-status.md](deployment-status.md)

## High-Volume Triage and Escalation Pattern

For workflows receiving many similar events (issues, PR comments, CI failures, security alerts, dependency events):

- start with a cheap triage/classification pass
- detect known/duplicate/stale/low-value cases first
- emit `noop` or a safe output when triage is confident
- escalate to the main agent only when uncertain or genuinely new/high-value

Decision flow:

```text
IF cheap triage is confident (known/duplicate/stale/low-value) THEN
  emit safe output or noop
ELSE
  escalate to the main agent
END IF
```

Use with pull-context workflows: fetch targeted evidence on demand instead of pushing raw logs into the initial prompt.

## Large-Repository Improvement Pattern

For recurring maintenance in large repos:

- use `cache-memory`
- process one package/module/directory per run
- store last-processed item; round-robin
- prefer small focused PRs over wide sweeps

See also: [memory.md](memory.md)

## Step Authoring Guidance

When writing `steps:`, `pre-steps:`, and `post-steps:`, choose the implementation type in this order of preference:

### 1. Preferred: `actions/github-script`

Use `actions/github-script` for GitHub API interactions and general scripting. The workflow compiler handles action pinning automatically; specify a recent major version tag (`@v7`) without a SHA.

- Provides typed access to the GitHub REST API via `github.rest.*`
- Exposes `context`, `core`, `github`, `io`, and `exec` helpers
- Eliminates shell injection risks for untrusted input
- Example:

  ```yaml
  steps:
    - name: Fetch issue data
      uses: actions/github-script@v7
      with:
        script: |
          const issue = await github.rest.issues.get({
            owner: context.repo.owner,
            repo: context.repo.repo,
            issue_number: context.issue.number,
          });
          core.setOutput('title', issue.data.title);
  ```

### 2. Shell scripts

Use `run:` steps when `actions/github-script` is not suitable. To prevent shell injection, never interpolate untrusted values directly into the script body. Any value that originates from user input — including `github.event.issue.title`, `github.event.issue.body`, `github.event.comment.body`, `github.event.pull_request.title`, `github.event.pull_request.body`, and `github.head_ref` — must be passed through environment variables:

```yaml
# ❌ Unsafe: direct expression interpolation into the shell script
- name: Unsafe comment
  run: gh issue comment ${{ github.event.issue.number }} --body "${{ github.event.issue.title }}"

# ✅ Safe: pass untrusted values through env vars and reference them as $VAR_NAME
- name: Safe comment
  env:
    ISSUE_NUMBER: ${{ github.event.issue.number }}
    TITLE: ${{ github.event.issue.title }}
  run: gh issue comment "$ISSUE_NUMBER" --body "$TITLE"
```

### 3. Python (last resort)

Use Python only when the task genuinely requires data science or numeric libraries (for example `pandas`, `numpy`, `matplotlib`). Prefer `actions/github-script` or a shell step for everything else.

## Pre-Step Data Fetching Pattern

Use deterministic `steps:` when the workflow needs large external data before the agent runs.

Rules:

- write prepared files to `/tmp/gh-aw/agent/`
- trim large outputs before handing to the agent
- set `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}` on every `gh` step
- add `permissions: actions: read` for downloading workflow logs/artifacts
- use `jq` to reduce JSON payload size

## PR Visual Regression Pattern

For PR UI validation and screenshot diffs:

- trigger: `pull_request`
- tools: `playwright` plus `cache-memory` for baseline metadata
- permissions: read-only repo/PR access
- output: `add-comment` with pass/fail summary and artifact links
- fallback: `noop` when no UI changes detected

## QA Coverage Report Pattern

For PR QA coverage summaries (gaps, risks, suggested test focus):

- trigger: `pull_request` (optionally scoped with `paths:`)
- tools: `github` (`gh-proxy`) for changed files, PR metadata, labels, checks
- permissions: `contents: read`, `pull-requests: read`; agent job read-only
- output: `add-comment` with coverage matrix and untested/high-risk areas
- fallback: `noop` for non-testable changes (e.g. docs-only)

## PM Stakeholder Digest Pattern

For recurring product/stakeholder digests:

- trigger: fuzzy `schedule` (e.g. `weekly on mondays`)
- tools: `github` (`gh-proxy`), optional `cache-memory` for period-over-period continuity
- permissions: read-only
- output: `create-issue` by default; `create-discussion` only when requested
- prompt: audience-aware language (summary first, details second)

## Database Migration Safety Pattern

For PRs adding/modifying migration files:

- trigger: `pull_request` with `paths:` scoped to migration dirs (e.g. `db/migrate/**`, `migrations/**`, `*.sql`)
- permissions: `contents: read`, `pull-requests: read`; agent job read-only
- reads: changed migration content via GitHub tools
- output: `add-comment` flagging risky operations; `noop` when clean
- prompt: include migration best practices

## Cross-Repository Pattern

For cross-repo reads and writes:

- enable GitHub toolsets needed for external repos
- configure PAT or GitHub App auth in `safe-outputs:` for cross-repo writes
- tell the agent to set `target-repo` explicitly
- document required token scopes in the prompt or instructions

Cross-repo workflows inherit single-job constraints from [workflow-constraints.md](workflow-constraints.md).
