# ADR-26693: Reaction Target Scoping and Union Activation Permission Derivation

**Date**: 2026-04-16
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

ADR-26535 established that activation job permissions are derived at compile time from the real GitHub event types in the `on:` section, scoped to only the write permissions actually needed. However, that approach treated `on.reaction` as a monolithic toggle — if reactions were enabled, all event groups (issues, pull requests, discussions) were eligible for reactions and their associated permissions were granted based solely on the real event types present. This caused a 403 error during activation on `pull_request` events: the reaction step ran unconditionally on PR events but the permission model from ADR-26535 did not account for configurations where PR reactions should be explicitly disabled. The existing `on.status-comment` configuration already supported per-target boolean toggles (`issues`, `pull-requests`, `discussions`); `on.reaction` had no equivalent, creating an inconsistency and a correctness gap.

### Decision

We will extend `on.reaction` to support an object form with optional `issues`, `pull-requests`, and `discussions` boolean target toggles, mirroring the existing `on.status-comment` object syntax. When the object form is used, only the enabled targets participate in reaction step condition generation and in permission derivation. We will compute activation job permissions as the union of permissions needed for the enabled reaction targets and the enabled status-comment targets, so that disabling PR reactions (`reaction.pull-requests: false`) removes `pull-requests: write` even when `pull_request` events are configured. Scalar reaction forms (`"eyes"`, `"rocket"`, `1`, etc.) remain unchanged and continue to imply all targets enabled.

### Alternatives Considered

#### Alternative 1: Add a Single `pull-requests` Toggle to `on.reaction`

Add only a `pull-requests: false` boolean directly on `on.reaction` (without object refactoring) to unblock the immediate 403 issue. This would fix the specific failure mode but would create a non-orthogonal API — `on.reaction` would support one target toggle but not `issues` or `discussions`. It was rejected because `on.status-comment` already demonstrates the value of full per-target control, and a partial API would require a follow-up breaking change to reach parity.

#### Alternative 2: Suppress Reaction Steps for Events Not in the Permission Set

Automatically derive reaction step conditions from the activation permission scope — if `pull-requests: write` is not granted, exclude PR events from the reaction step condition. This approach avoids a new configuration surface but ties UI behavior (which events get a reaction) to a security concern (which permissions are granted), making the coupling non-obvious. Authors who want reactions on issues but not PRs would have no explicit way to express that intent. It was rejected in favor of explicit configuration.

#### Alternative 3: Unify `reaction` and `status-comment` into a Single `interactions` Object

Replace both `on.reaction` and `on.status-comment` with a single top-level `on.interactions` object that has `reaction` and `status-comment` sub-keys, each with their own target toggles. This would produce a cleaner schema but is a breaking change to all existing workflows that use `on.reaction` or `on.status-comment`. The migration cost and churn outweighed the schema elegance benefit, so it was deferred.

### Consequences

#### Positive
- The 403 activation error on `pull_request` events when `reaction.pull-requests: false` is resolved: the reaction step condition no longer includes PR events, so the step does not run and no PR permission is required.
- `on.reaction` and `on.status-comment` now have symmetric per-target APIs, making the workflow configuration model easier to understand and document.
- Permission derivation is more accurate: disabling a reaction target explicitly removes the corresponding write scope from the activation job, reducing the blast radius of a compromised token.
- All three target dimensions (issues, pull-requests, discussions) are controllable independently for both reactions and status comments.

#### Negative
- The `addActivationInteractionPermissionsMap` and `addBroadActivationInteractionPermissions` functions now accept three additional `bool` parameters each (`reactionIncludesIssues`, `reactionIncludesPullRequests`, `reactionIncludesDiscussions`), increasing function arity and making call sites more verbose. Tests that use these functions directly require updates when the signature changes again.
- A configuration where all reaction targets are disabled (`issues: false, pull-requests: false, discussions: false`) is rejected at parse time as invalid — this may surprise authors who expect a no-op behavior. The validation error message must be clear enough to guide authors to use `reaction: none` instead.
- `parseReactionValue` is now superseded by `parseReactionConfig`, which returns five values including three nullable booleans. Callers must handle the `nil`-means-default convention correctly or risk subtle bugs where the scalar form (returning `nil` pointers) is incorrectly treated as "disabled".

#### Neutral
- `WorkflowData` gains three new nullable `*bool` fields (`ReactionIssues`, `ReactionPullRequests`, `ReactionDiscussions`), consistent with the existing `StatusCommentIssues`, `StatusCommentPullRequests`, `StatusCommentDiscussions` fields. `nil` means "inherit default (true)".
- The JSON schema for `on.reaction` is extended with a new `oneOf` branch for the object form. Existing scalar usages are unaffected.
- `BuildReactionCondition()` now delegates to the new `BuildReactionConditionForTargets(true, true, true)` — it remains the public API for callers that do not need target scoping, so no callers outside the compiler are broken.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Reaction Configuration

1. Implementations **MUST** accept `on.reaction` in both scalar form (string or integer reaction type) and object form (`{type?, issues?, pull-requests?, discussions?}`).
2. In object form, implementations **MUST** default `type` to `"eyes"` when the `type` key is absent.
3. In object form, implementations **MUST** default each of `issues`, `pull-requests`, and `discussions` to `true` when the key is absent.
4. Implementations **MUST** reject a `reaction` object where all three target toggles are explicitly set to `false`, returning a parse-time error; authors **SHOULD** use `reaction: none` to disable reactions entirely.
5. Implementations **MUST NOT** change the behavior of scalar reaction forms — a scalar value **MUST** be treated as if all three target toggles are `true`.

### Reaction Step Condition Generation

1. Implementations **MUST** generate the reaction step `if:` condition using only the event names corresponding to the enabled reaction targets.
2. Implementations **MUST NOT** include `pull_request` or `pull_request_review_comment` event names in the reaction step condition when `reaction.pull-requests` is `false`.
3. Implementations **MUST NOT** include `issues` or `issue_comment` event names in the reaction step condition when `reaction.issues` is `false`.
4. Implementations **MUST NOT** include `discussion` or `discussion_comment` event names in the reaction step condition when `reaction.discussions` is `false`.

### Activation Permission Derivation

1. Implementations **MUST** compute the activation job permission set as the union of permissions required for enabled reaction targets and permissions required for enabled status-comment targets, both evaluated against the real GitHub event types in the `on:` section.
2. Implementations **MUST NOT** grant `issues: write` for reaction purposes unless `reaction.issues` is `true` (or unset) AND at least one of `issues`, `issue_comment`, or `pull_request` event types is configured.
3. Implementations **MUST NOT** grant `pull-requests: write` for reaction purposes unless `reaction.pull-requests` is `true` (or unset) AND `pull_request_review_comment` event type is configured.
4. Implementations **MUST NOT** grant `discussions: write` for reaction purposes unless `reaction.discussions` is `true` (or unset) AND at least one of `discussion` or `discussion_comment` event types is configured.
5. Implementations **MUST** apply the same target-scoped permission derivation to both the activation job `permissions` block and the GitHub App token minting permissions (consistent with ADR-26535).

### Fallback Behavior

1. When the `on:` section is absent or unparseable at permission derivation time, implementations **MUST** fall back to the broad permission grant defined in ADR-26535, respecting the target toggle booleans for the broad-grant path as well.
2. Implementations **MUST NOT** ignore the `reactionIncludesPullRequests` flag in the fallback path — a configuration with `reaction.pull-requests: false` **MUST** suppress `pull-requests: write` even under the broad fallback.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. This ADR extends ADR-26535; conformance with this ADR implies conformance with ADR-26535 except where this ADR explicitly supersedes it.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24526383923) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
