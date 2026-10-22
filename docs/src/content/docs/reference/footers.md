---
title: Footer Control
description: Learn how to control AI-generated footers in safe output operations and customize footer messages for GitHub issues, pull requests, discussions, and releases.
sidebar:
  order: 805
---

Control whether AI-generated footers are added to created and updated GitHub items (issues, pull requests, discussions, releases). Footers provide attribution and links to workflow runs, but you may want to omit them for cleaner content or when using custom branding.

## Global Footer Control

Set `footer: false` at the safe-outputs level to hide footers for all output types:

```yaml wrap
safe-outputs:
  footer: false                      # hide footers globally
  create-issue:
    title-prefix: "[ai] "
  create-pull-request:
    title-prefix: "[ai] "
```

When `footer: false` is set, visible attribution text is omitted from item bodies but hidden XML markers remain for searchability:
- `<!-- gh-aw-workflow-id: WORKFLOW_NAME -->` — for search and tracking
- `<!-- gh-aw-tracker-id: unique-id -->` — for issue/discussion tracking

Applies to all output types: create-issue, create-pull-request, create-discussion, update-issue, update-pull-request, update-discussion, and update-release.

### Work Queue Attribution

Safe outputs published under a work queue Claim include the full Work and Claim IDs on a separate blockquote line in the generated footer:

```markdown
> Work `work-id` · Claim `claim-id`
```

The IDs come from the trusted per-Claim execution context, not agent-supplied content. In multi-Claim runs, each output references only its own Claim. This line is also appended to custom `messages.footer` templates; `footer: false` hides it along with the rest of the visible footer. Runs without a Claim retain their existing footer.

### Execution Provenance

Issue, pull request, comment, and discussion bodies also contain an always-on `<!-- gh-aw-execution: JSON -->` marker. Disabling the visible footer or customizing `messages.footer` and `body-footer` does not remove this marker or the existing combined metadata marker. Reserved execution markers in supplied content are replaced with trusted runtime metadata. Publishing fails explicitly if required runtime metadata is missing or invalid.

The version 1 JSON payload contains:

| Field | Meaning |
|---|---|
| `version` | Payload schema version, currently `1` |
| `repo_id`, `repository` | Execution repository ID (`GITHUB_REPOSITORY_ID`) and `owner/repo` (`GITHUB_REPOSITORY`), not the destination repository |
| `aw_workflow_id` | AW workflow filename without its extension (`GH_AW_WORKFLOW_ID`), not the numeric Actions workflow ID |
| `run_id`, `run_attempt` | Execution run and attempt (`GITHUB_RUN_ID`, `GITHUB_RUN_ATTEMPT`); a rerun retains the run ID but changes the attempt |
| `run_url` | Canonical run URL in the execution repository |
| `publishing_job.key` | Logical job key (`GITHUB_JOB`), not a unique execution ID |
| `publishing_job.check_run_id` | Numeric publishing check-run ID, supplied through step-level `${{ job.check_run_id }}` |

Parse the comment payload with a JSON parser; HTML-sensitive characters are JSON-escaped. Existing standalone workflow and caller markers retain their grouping and deduplication semantics.

To resolve the numeric Actions job ID, list **all pages** of `GET /repos/{repository}/actions/runs/{run_id}/attempts/{run_attempt}/jobs` in the execution repository. Select the job whose `check_run_url` points to that repository's `/check-runs/{publishing_job.check_run_id}` on the expected GitHub API host. Require exactly one match and verify its `run_id` and, when returned, `run_attempt` against the payload before using its numeric `id`. Never resolve by display name or logical job key: matrix legs and reusable workflow invocations can share those names but have distinct check-run IDs. The run API's `workflow_id` supplies the numeric Actions workflow ID; it is separate from `aw_workflow_id`. API resolution requires permission to read Actions runs in the execution repository, even when publishing uses a different destination repository or App token.

The runtime's `execution_provenance.cjs` exports `resolvePublishingJobId(github, metadata)` to perform this paginated lookup and validation using an Octokit client and a parsed marker. It returns the Actions job ID as a string and rejects missing, ambiguous, or mismatched executions. Publishing itself does not require this lookup or additional Actions permissions.

The marker describes execution, not authorship. The triggering `github.actor` is not the publishing bot or App; use the artifact's GitHub author identity for the publisher. Recompile older workflows to provide the required publishing check-run context.

Maintenance replays record the publishing maintenance workflow's identifier, repository, run, attempt, and check run—not the earlier run whose agent output is being replayed.

### Searching for Workflow-Created Items

Use the `gh-aw-workflow-id` marker (the workflow filename without `.md`) to find items in GitHub search:

```
repo:owner/repo is:issue is:open "gh-aw-workflow-id: daily-team-status" in:body
repo:owner/repo "gh-aw-workflow-id: bot-responder" in:comments
```

Combine with `is:open`, `created:>2024-01-01`, or `org:your-org` filters. See [GitHub advanced search](https://docs.github.com/en/search-github/searching-on-github/searching-issues-and-pull-requests).

## Per-Handler Footer Control

Override the global setting for specific output types by setting `footer` at the handler level:

```yaml wrap
safe-outputs:
  footer: false                      # global default: no footers
  create-issue:
    title-prefix: "[issue] "
    # inherits footer: false
  create-pull-request:
    title-prefix: "[pr] "
    footer: true                     # override: show footer for PRs only
```

Individual handler settings always take precedence over the global setting.

## Deterministic Body Footers

Use `body-footer` at the `safe-outputs` level to append workflow-defined content to every safe output that produces a body:

```yaml wrap
safe-outputs:
  body-footer: |
    ---
    Generated by [{workflow_name}]({run_url})
  create-issue:
    body-footer: |
      ---
      Issue-specific content.
  create-pull-request:
    body-footer: |
      ---
      Review the generated changes before merging.
```

The global body footer is added to any handler-specific `body-footer`; both are appended even when `footer: false` and remain separate from the generated attribution footer. Body footers from imported agentic workflows are additive, in import order. The template supports `{workflow_name}` and `{run_url}` placeholders.

## PR Review Footer Control

For PR reviews (`submit-pull-request-review`), the `footer` field supports conditional control over when the footer is added to the review body:

```yaml wrap
safe-outputs:
  create-pull-request-review-comment:
  submit-pull-request-review:
    footer: "if-body"         # conditional footer based on review body
```

The `footer` field accepts `"always"` (default), `"none"`, or `"if-body"` (footer only when the review has body text). Booleans are accepted: `true` → `"always"`, `false` → `"none"`. Use `"if-body"` for clean approval reviews — approvals without body text appear without a footer, while reviews with comments include it.

## Backward Compatibility

The default value for `footer` is `true`. To hide footers, explicitly set `footer: false`.

## Customizing Footer Messages

Instead of hiding footers entirely, you can customize the footer message text using the `messages.footer` template. This allows you to maintain attribution while using custom branding:

```yaml wrap
safe-outputs:
  messages:
    footer: "> 🤖 Powered by [{workflow_name}]({agentic_workflow_url})"
  create-issue:
    title-prefix: "[bot] "
```

The `messages.footer` template supports variables like `{workflow_name}`, `{agentic_workflow_url}`, `{run_url}`, `{triggering_number}`, `{triggering_type}`, `{ai_credits_suffix}`, and more. `{triggering_type}` is `issue`, `PR`, or `discussion`, matching the type of item that triggered the run. `{agentic_workflow_url}` links directly to the agentic workflow file view for the run (equivalent to `{run_url}/agentic_workflow`), while `{run_url}` links to the plain Actions run page. `{ai_credits_suffix}` is a pre-formatted, always-safe suffix (e.g. `" · sonnet46 · ~13 AIC"` or `""`) that you can place directly before `{history_link}`. When the run's engine model is known, the suffix is prefixed with a deterministic compact model identifier (`sonnet46`, `gpt55`, `opus47`, `haiku45`, `gem25`, …), and direct short aliases like `opus`, `sonnet`, and `haiku` are preserved. Individual cost components are also exposed: `{ai_model}` (full model name, e.g. `claude-sonnet-4.6`), `{ai_model_short}` (compact identifier, e.g. `sonnet46`), `{ai_credits}` (raw numeric cost), `{ai_credits_formatted}` (formatted cost), `{ai_credits_unit}` (always `AIC`), `{agent_ai_credits_formatted}`, `{evals_ai_credits_formatted}`, `{threat_detection_ai_credits_formatted}`, `{detection_conclusion}`, and `{detection_reason}`. See [Custom Messages](/gh-aw/reference/safe-outputs/#custom-messages-messages) for complete documentation on message templates and available variables.

Footer costs are estimates: formatted AIC values include a `~` prefix and round up to whole credits without decimals (`12.4` becomes `~13 AIC`). This applies to the total and each agent, threat-detection, and evals component, including no-op footers. Estimates intentionally omit compact K/M notation to preserve whole-credit precision (`1000.1` becomes `~1001 AIC`, not `~1.0K AIC`). Floating-point arithmetic noise near whole credits is normalized only for display. Raw numeric template variables retain their original precision.

## Learn More

- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Complete safe outputs reference
- [Custom Messages](/gh-aw/reference/safe-outputs/#custom-messages-messages) - Message templates and variables
- [Frontmatter](/gh-aw/reference/frontmatter/) - All configuration options for workflows
