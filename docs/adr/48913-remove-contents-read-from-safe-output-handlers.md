# ADR-48913: Remove `contents: read` from Output-Only Safe-Output Handlers

**Date**: 2026-07-29
**Status**: Draft
**Deciders**: GitHub Agentic Workflows Team

---

### Context

The `safe_outputs` job compiles and dispatches agent-produced outputs (create issue, add comment, add labels, update pull request, etc.) by downloading an NDJSON artifact and calling GitHub APIs. It does **not** check out repository file contents.

Nearly every handler in the job requested `contents: read` as an unconditional baseline, documented as "always included for repository context access." This applied to output-only handlers — those that call only the issues, pull-requests, discussions, checks, security-events, or organization-projects APIs — where no repository file read occurs. The permission had no functional basis for these handlers.

Operators using tightly scoped GitHub Apps (one App per role, each granted the minimum permissions for its job) were forced to grant `contents: read` to an output-only App and justify that grant in security review. The spec's stated rationale ("Repository metadata and context") was insufficient because GitHub App installations already carry implicit `metadata: read`; the broader `contents` scope goes further and is unnecessary for pure output workloads. This was raised in [issue #48882](https://github.com/github/gh-aw/issues/48882).

### Decision

We will remove `contents: read` from the permission sets of all output-only `safe_outputs` handlers and add dedicated factory functions (`NewPermissionsIssuesWrite`, `NewPermissionsPRWrite`, `NewPermissionsDiscussionsWrite`, etc.) that express each handler's actual minimal scope. Handlers that legitimately need `contents` access (`create-pull-request`, `merge-pull-request`, `push-to-pull-request-branch`, `update-pull-request` with `update-branch: true`, `update-release`) retain `contents: write`. The `upload_asset` PermissionBuilder is removed entirely because the `safe_outputs` job never processes `upload_asset` items — those are handled by the separate `publish_assets` job.

### Alternatives Considered

#### Alternative 1: Document the rationale and keep `contents: read`

Document why `contents: read` is universally included (e.g., some implementation detail requires it) without removing it. This would be the right approach if there were a real functional dependency. Analysis of the output path showed no such dependency: the job downloads an artifact and calls GitHub APIs; it does not read repository files. Keeping an unnecessary permission violates least-privilege and does not help operators justify the grant in security review.

#### Alternative 2: Make `contents: read` opt-in per handler configuration

Add a configuration flag allowing operators to request `contents: read` when they believe their workflow needs it. Rejected because the permission is not needed by any current output-only handler. An opt-in mechanism adds complexity without benefit and defers a clear decision; the correct baseline is no permission, not a configurable one.

### Consequences

#### Positive
- Output-only `safe_outputs` jobs now operate with the minimal permission set required by the APIs they actually call, consistent with the least-privilege principle stated in the spec (section 2.1).
- Operators using tightly scoped GitHub Apps can configure an issue-writer or comment-writer App without granting repository-content read access, removing a friction point that previously required justification in security review.
- The `upload_asset` handler registration is simplified: the `safe_outputs` job no longer contributes a spurious `contents: read` for a handler type it never processes.

#### Negative
- If any undiscovered runtime dependency on `contents: read` exists in an output handler (e.g., a third-party action invoked within the job that silently relied on the token scope), it would fail at runtime rather than at permission-grant time. Such a failure would be visible in job logs but could surface unexpectedly in production before being caught.
- Existing operator workflows that hardcode `contents: read` in their `safe-outputs` job permissions block will no longer have that permission contributed by the handler factories; they must supply it explicitly if needed for another reason. This is unlikely to break anything but requires awareness.

#### Neutral
- The spec is bumped to v1.27.0 and all permission tables for output-only handlers are updated to reflect the minimal sets, giving operators accurate documentation for the first time.
- Twelve new factory functions are added to `permissions_factory.go`; the previous `NewPermissionsContentsRead*` variants remain for handlers that still need `contents` access, keeping backwards compatibility within the codebase.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
