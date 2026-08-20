---
description: Maps every compiler-generated job to its purpose, dependencies, and GitHub token or GitHub App configuration so agents grant credentials only to the job that needs them.
---

# Compiler-Generated Jobs

The compiler owns the job IDs in this reference. Do not define a custom
`jobs.<id>` entry using a generated name. Built-in job configuration can add
supported `setup-steps`, `pre-steps`, `needs`, and `if` values, but must not
replace compiler-managed permissions or authentication.

## Authentication configuration

Configure credentials at the feature that consumes them, not by adding a token
to an unrelated job:

| Consumer jobs | Configure a GitHub token | Configure a GitHub App |
|---|---|---|
| `pre_activation`, `activation` | `on.github-token: ${{ secrets.MY_TOKEN }}` | `on.github-app` with `client-id`, `private-key`, and optional `owner` or `repositories` |
| `agent` | `tools.github.github-token: ${{ secrets.MY_TOKEN }}` | `tools.github.github-app` |
| `safe_outputs`, `upload_assets`, `upload_code_scanning_sarif`, and `call-*` | `safe-outputs.github-token: ${{ secrets.MY_TOKEN }}`, or the handler's `github-token` override | `safe-outputs.github-app`, or the handler's `github-app` override |
| Maintenance jobs | Do not add workflow frontmatter. Configure the generated maintenance workflow's repository `GITHUB_TOKEN` permissions, or use the supported maintenance configuration. |

Use a repository secret expression for every custom token; never put a token or
private key in workflow source. Prefer the least-privileged token or App
installation. The `agent` and `detection` jobs intentionally do not receive
write permissions: use safe outputs for writes. `permissions:
{ copilot-requests: write }` lets the Copilot agent use `${{ github.token }}`
for inference and does not require a PAT or `COPILOT_GITHUB_TOKEN`.

## Agentic workflow jobs

| Job | Created when | Depends on | Credential owner and purpose |
|---|---|---|---|
| `pre_activation` | Trigger validation, role checks, skip checks, or `on.github-app` is configured | — | `on.github-token` or `on.github-app`; performs trigger-time reactions, status comments, and queries. |
| `activation` | Most agentic workflows | `pre_activation` when present | Uses the activation credential established by `on.github-token` or `on.github-app` only for activation work. |
| `agent` | Every agentic workflow | `activation` when present | `tools.github.github-token` or `tools.github.github-app` for GitHub tool access. Keep this job read-only. |
| `detection` | Threat detection is enabled for safe outputs | `agent` | No separate credential configuration; preserve its read-only isolation. |
| `safe_outputs` | At least one built-in safe output, script, action, or safe-output job is configured | `agent`, and `detection` when present | `safe-outputs.github-token` or `safe-outputs.github-app`; performs approved GitHub writes. |
| `upload_assets` | A safe output uploads release assets | `safe_outputs` | Inherits safe-output authentication; grant only the release permissions required. |
| `upload_code_scanning_sarif` | A safe output creates code-scanning alerts | `safe_outputs` | Inherits safe-output authentication; requires `security-events: write`. |
| `unlock` | `lock-for-agent` is enabled | `agent` | Uses the workflow token for lock cleanup; do not give the agent job write access. |
| `evals` | `evals` is configured | `agent` | No custom GitHub credential; evaluates agent output in parallel with safe outputs. |
| `conclusion` | The compiler creates a workflow conclusion | All preceding generated and custom jobs | No custom GitHub credential; aggregates workflow results and usage. |

## State and fan-out jobs

| Job | Created when | Credential owner and purpose |
|---|---|---|
| `push_repo_memory` | Repository-backed memory is configured | Framework-managed repository credential; persists memory after threat detection. |
| `update_cache_memory` | Cache memory is configured | Framework-managed repository credential; updates cache state after threat detection. |
| `push_experiments_state` | Repository-backed experiments are configured | Framework-managed repository credential; persists experiment allocation state. |
| `push_evals_state` | Evals persist state | Framework-managed repository credential; persists evaluation results. |
| `call-<sanitized-worker-name>` | A `safe-outputs.call-workflow` worker is configured | Configure the called workflow's token input through that worker's `github-token` or `github-app`; each worker name is sanitized and emitted with the `call-` prefix. |

## Maintenance workflow jobs

The maintenance compiler emits `agentics-maintenance.yml`. These jobs use the
generated workflow's `GITHUB_TOKEN` and explicit, job-level permissions; do not
attempt to configure them through an agentic workflow's frontmatter.

| Job | Purpose |
|---|---|
| `close-expired-discussions` | Closes expired discussions. |
| `close-expired-issues` | Closes expired issues. |
| `close-expired-pull-requests` | Closes expired pull requests. |
| `cleanup-cache-memory` | Removes stale cache-memory entries. |
| `run_operation` | Runs selected maintenance operations. |
| `update_pull_request_branches` | Updates eligible pull request branches. |
| `apply_safe_outputs` | Replays approved safe outputs. |
| `create_labels` | Creates configured labels. |
| `activity_report` | Produces the activity report. |
| `forecast_report` | Produces the forecast report. |
| `close_agentic_workflows_issues` | Closes resolved agentic-workflow tracking issues. |
| `validate_workflows` | Validates configured workflows. |
| `label_disable_agentic_workflow` | Applies the disable-workflow label operation. |
| `label_apply_safe_outputs` | Applies the safe-output label operation. |
| `compile-workflows` | Compiles tracked workflow sources. |
| `secret-validation` | Validates secrets in development-mode maintenance workflows. |

## Compatibility aliases

`pre-activation` and `safe-outputs` are reserved compatibility aliases for
`pre_activation` and `safe_outputs`. The compiler emits the underscore forms;
use those forms in `needs` and built-in job configuration.
