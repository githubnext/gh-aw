# ADR-53576: Pre-create Pull Request During Activation

**Date**: 2026-08-18
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Agentic workflows compiled by gh-aw run for extended periods — agent execution can take minutes to hours. During this window no GitHub artifact (branch or pull request) is visible, so users and reviewers have no link to the work-in-progress and no way to monitor progress from the Pull Requests tab. The existing model defers PR creation to the `safe_outputs` job, which only runs after the agent finishes and its patch has been validated.

### Decision

We will add an opt-in `pre-create: true` flag under `safe-outputs.create-pull-request`. When set, the compiler generates an extra activation step that: (1) creates an empty commit on a run-scoped branch (`gh-aw/pre-created/<runId>-<attempt>`), (2) opens a draft PR titled `[<workflow>] Work in progress`, and (3) attaches an in-progress GitHub Checks run linked to the workflow run. Subsequent agent and `safe_outputs` jobs check out this pre-created branch, and the eventual `create_pull_request` safe-output updates the existing PR (title, body, base) instead of creating a new one. The conclusion job completes the linked check.

### Alternatives Considered

#### Alternative 1: Status comment on the triggering issue

Post a comment on the issue or PR that triggered the workflow with a link to the workflow run. This is zero-cost in terms of permissions and produces no orphan if the agent fails, but leaves no PR artifact; reviewers cannot use the Pull Requests tab or assign reviews until the agent finishes.

#### Alternative 2: Always create the PR immediately, update it later (non-opt-in)

Make early PR creation the default behaviour for all `create-pull-request` workflows. This would remove the opt-in complexity but would impose `contents: write`, `pull-requests: write`, and `checks: write` on all activation jobs unconditionally — even workflows that don't need early visibility — violating the principle of least privilege.

### Consequences

#### Positive
- Workflow runs are immediately visible as a PR, enabling early review assignment, status tracking, and linking from issues.
- The linked GitHub Check provides a progress link on the PR's commit that updates to success or failure when the workflow concludes.
- Downstream jobs receive the pre-allocated branch and PR number as outputs, enabling coherent multi-job coordination without race conditions.

#### Negative
- Requires elevated activation-job permissions (`contents: write`, `pull-requests: write`, `checks: write`), increasing the blast radius of a compromised activation job.
- An empty draft PR is left open if the workflow is cancelled or fails before the agent produces any changes, requiring manual closure or additional cleanup logic.
- The feature is limited to single-repository, single-PR workflows; cross-repository targets and `checkout: false` configurations are not supported.

#### Neutral
- Schema validation (`validatePreCreatePullRequest`) enforces the constraints (max:1, same-repo, default checkout) at compile time, so invalid combinations are caught early.
- Two lock files in the repo (`daily-team-evolution-insights`, `mcp-inspector`) are recompiled as a side effect, removing the `strict` field from their metadata headers.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
