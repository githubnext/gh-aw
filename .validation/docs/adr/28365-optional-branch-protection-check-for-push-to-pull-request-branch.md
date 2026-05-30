# ADR-28365: Optional Branch Protection Pre-flight Check for `push-to-pull-request-branch`

**Date**: 2026-04-25
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `push-to-pull-request-branch` safe output handler performs a pre-flight check against the GitHub branch protection API (`GET /repos/{owner}/{repo}/branches/{branch}/protection`) before pushing. This call requires `administration: read` permission, which is a GitHub App-only scope not available on standard `GITHUB_TOKEN`. Without this scope, the API returns HTTP 403, which the handler silently swallowed—logging a warning and continuing—making the check effectively useless noise. At the same time, automatically granting `administration: read` to all workflows that use this handler would increase every deployment's permission surface area, even for teams that don't need or want the protection check.

### Decision

We will make the branch protection pre-flight check opt-out via a new `check-branch-protection` boolean frontmatter key (default `true`). When the check is enabled (the default), the permission compiler automatically adds `administration: read` to the GitHub App token so the API call succeeds. When `check-branch-protection: false` is set, neither the API call nor the `administration: read` permission is added. The default branch guard (blocking pushes to the repository's default branch) runs unconditionally regardless of this setting, because it uses the repos API rather than the branch protection API.

### Alternatives Considered

#### Alternative 1: Always require `administration: read` with no opt-out

All workflows using `push-to-pull-request-branch` would automatically receive `administration: read`. This eliminates the silent-failure problem and simplifies the implementation. It was rejected because it forces every team to grant a broad GitHub App scope even when they have no interest in the branch protection check and prefer minimal permission footprints.

#### Alternative 2: Remove the branch protection pre-flight check entirely

Eliminating the check avoids the permission problem without any new configuration surface. It was rejected because the check provides genuine defense-in-depth: it prevents agentic pushes to branches under review policies, catching misconfigurations before GitHub's push-time enforcement triggers a harder failure. The value of the check outweighs the complexity of making it configurable.

#### Alternative 3: Silently ignore 403 and keep the existing behavior

The current code already falls through on 403 with a warning, meaning the check is already a no-op for most GitHub App tokens. This was rejected because it perpetuates misleading log output and provides no security value while still requiring `administration: read` to appear in the schema. An explicit opt-out is more honest about what the check does and does not guarantee.

### Consequences

#### Positive
- The 403-swallow silent failure is eliminated: when the check is enabled, the token has the right scope and the check is meaningful.
- Users with strict permission policies can set `check-branch-protection: false` to avoid requesting `administration: read` entirely.
- The default branch guard (using the repos API) continues to run unconditionally, preserving the most critical safety check.
- Extracting the combined guard logic into a `checkBranchPushable` helper improves testability and isolation.

#### Negative
- Enabling the default increases the permission footprint of all existing `push-to-pull-request-branch` deployments that use GitHub App tokens (they gain `administration: read` automatically on upgrade).
- The configuration surface for the handler grows by one field, adding a new concept callers must understand.
- The `administration: read` permission is a GitHub App-only scope with no effect on `GITHUB_TOKEN`; this asymmetry may confuse users who rely on `GITHUB_TOKEN`.

#### Neutral
- The JSON schema for `push-to-pull-request-branch` gains a `check-branch-protection` boolean property, which may require documentation updates in tooling that consumes the schema.
- The permission computation in `ComputePermissionsForSafeOutputs` now branches on the effective `CheckBranchProtection` value, so permission calculations are no longer purely additive for this handler.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Branch Protection Check Configuration

1. The `push-to-pull-request-branch` handler **MUST** treat `check-branch-protection` as an optional boolean key in workflow frontmatter, defaulting to `true` when absent.
2. When `check-branch-protection` is `true` (the default), the handler **MUST** call `GET /repos/{owner}/{repo}/branches/{branch}/protection` before executing the push.
3. When `check-branch-protection` is `false`, the handler **MUST NOT** call the branch protection API and **MUST NOT** request `administration: read` permission.
4. The handler **MUST** block the push and return an error when the branch protection API responds with a successful (2xx) status, indicating active protection rules exist.
5. The handler **MUST** permit the push when the branch protection API responds with HTTP 404 (no rules configured).
6. The handler **SHOULD** log a warning and permit the push when the branch protection API responds with HTTP 403 (insufficient permissions), because the GitHub platform still enforces protection at push time.
7. The handler **MUST** block the push when the branch protection API responds with any unexpected error (e.g., 5xx, network failure) to fail closed and prevent accidental writes to potentially protected branches.

### Default Branch Guard

1. The handler **MUST** check whether the target branch is the repository's default branch, regardless of the `check-branch-protection` setting.
2. The handler **MUST** block the push and return an error when the target branch equals the repository's default branch.
3. The handler **SHOULD** log a warning and continue when the repositories API call needed to resolve the default branch fails, rather than blocking the push due to an unrelated API error.

### Permission Computation

1. The permission compiler **MUST** automatically add `administration: read` to the computed GitHub App token permissions when `check-branch-protection` is `true` (or absent/defaulted).
2. The permission compiler **MUST NOT** add `administration: read` when `check-branch-protection` is explicitly `false`.
3. Implementations **SHOULD** document that `administration: read` is a GitHub App-only scope with no effect on standard `GITHUB_TOKEN`.

### Schema

1. The JSON schema for `push-to-pull-request-branch` **MUST** declare `check-branch-protection` as an optional boolean property with `default: true`.
2. The schema **MUST NOT** require `check-branch-protection` to be present.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24917291168) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
