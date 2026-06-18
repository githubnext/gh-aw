# ADR-30071: Decouple Safe-Outputs Base Branch from GitHub Actions Event Context

**Date**: 2026-05-04
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

Agent workflows that push to pull request branches or create pull requests need to check out the correct base branch before applying the agent's output. The checkout `ref` was resolved entirely from GitHub Actions event-context expressions (`github.base_ref`, `github.event.pull_request.base.ref`, etc.). These expressions are only populated for `pull_request` and `pull_request_target` events; for `issue_comment` events (which can trigger agent runs on PRs targeting non-default branches), the expressions silently resolve to the repository default branch instead of the actual PR base branch. The agent harness JavaScript handlers already call the GitHub API at runtime to resolve the correct `base_branch`; that resolved value was not being persisted anywhere the workflow runner could use it.

### Decision

We will store the API-resolved `base_branch` value in the safe-output entry payload (persisted to `agent_output.json`) and add a dedicated shell step — `extract-base-branch` — that reads this value from the downloaded artifact and writes it to `GITHUB_OUTPUT` before the checkout step. The checkout `ref` expression will lead with the extracted value and fall back to event-context expressions for backward compatibility with older agent outputs that do not carry `base_branch`.

### Alternatives Considered

#### Alternative 1: Extend the Event-Context Fallback Chain

Add more GitHub Actions expressions (e.g., `github.event.issue.pull_request.base.ref`) to the existing fallback chain without persisting the value from the agent. This was rejected because the `issue_comment` event payload does not expose the PR base branch through any available expression; the only reliable source is a GitHub API call, which the agent already makes at scheduling time.

#### Alternative 2: Fetch the PR Base Branch in a Separate Workflow Step

Insert a `gh api` call into the workflow itself (before checkout) to fetch the PR base branch at execution time. This was considered but rejected because it introduces a redundant API call — the agent already resolves the branch — and it tightly couples the compiled workflow to GitHub API availability at checkout time, increasing blast radius. Embedding the resolved value in the agent output keeps the fetch logic co-located with the other branch resolution code in the agent harness.

### Consequences

#### Positive
- Checkout now uses the correct base branch for all event types, including `issue_comment` events on PRs targeting non-default branches.
- Resolves a longstanding `TODO` and `LIMITATION` comment in the codebase; the architectural debt is fully addressed.
- The extracted value is validated for safe branch-name characters before being written to `GITHUB_OUTPUT`, preventing injection via a malicious agent output.

#### Negative
- Both the JavaScript frontend (`safe_outputs_handlers.cjs`) and the Go backend (`compiler_safe_outputs_steps.go`, `compiler_safe_outputs_job.go`) must be updated in lockstep; a version mismatch between agent output format and workflow step would silently fall back to the (potentially wrong) event-context value.
- All 48+ compiled `.lock.yml` workflow files must be regenerated whenever the extraction step template changes, adding bulk to refactoring PRs.

#### Neutral
- Agent outputs without a `base_branch` field (older format) continue to work via the unchanged event-context fallback expressions; no migration is required for existing in-flight runs.
- The new `extract-base-branch` step appears in every compiled workflow's job, slightly increasing workflow step count but with negligible runtime cost.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Agent Output Payload

1. Implementations **MUST** write a `base_branch` field into each safe-output entry of type `create_pull_request` or `push_to_pull_request_branch` when a resolved base branch is available.
2. The `base_branch` value **MUST** be the API-resolved branch name, not an unevaluated GitHub Actions expression string.
3. Implementations **MUST NOT** omit `base_branch` from the payload when the agent harness has already resolved the correct base branch via an API call.

### Workflow Extraction Step

1. Every compiled workflow job that performs a safe-outputs checkout **MUST** include an `extract-base-branch` step that runs after the agent artifact download and before the checkout step.
2. The extraction step **MUST** validate the extracted branch name using `git check-ref-format --branch` semantics and enforce a maximum length of 255 characters before writing to `GITHUB_OUTPUT`.
3. The extraction step **MUST NOT** fail the workflow if `agent_output.json` is absent or if no matching safe-output entry contains `base_branch`; it **MUST** exit successfully (silently) in those cases.
4. Checkout steps **MUST** lead the `ref` expression with `steps.extract-base-branch.outputs.base-branch` and **SHOULD** retain event-context fallbacks (`github.base_ref`, `github.event.pull_request.base.ref`, etc.) for backward compatibility.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: (1) safe-output entries include `base_branch` when resolved, (2) every checkout-capable compiled workflow includes the validated extraction step ordered before checkout, and (3) the checkout `ref` leads with the extracted value. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25323006144) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
