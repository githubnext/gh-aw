# ADR-41580: Marker-Based Deduplication and Labeling for Org Runner PRs and Issues

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `gh aw update --org` and `gh aw upgrade --org` commands create PRs and issues in target repositories to notify maintainers that workflow updates or upgrades are available. Before this change, these org runner operations had no reliable deduplication: each run would stack a new open PR or issue on top of existing ones. The upgrade path had a fragile title-based idempotency guard (skipping creation only if an open issue with the exact title existed), while the update path had no deduplication at all. Items also lacked any label, version stamp, or link back to the triggering gh-aw release, making them difficult to search or filter in GitHub.

### Decision

We will embed version-stamped HTML comment markers (e.g., `<!-- gh-aw-upgrade: v1.2.3 -->`) in all org runner PR and issue bodies. Before creating a new item, the org runner will fetch all open issues or PRs in the target repository and close any whose body contains the marker prefix. We will also apply the `agentic-workflows` label and include a link to the triggering gh-aw release in every created item. Shared helpers for marker construction, deduplication, labeling, and issue creation are extracted into `pkg/cli/org_issue_pr_helpers.go`.

### Alternatives Considered

#### Alternative 1: Title-Based Deduplication

The previous upgrade path used title-based idempotency: before creating an issue, it fetched all open issues and skipped creation if one with the exact title already existed. This approach is fragile — issue titles can be edited by maintainers, the check was absent from the PR creation flow entirely, and it carries no version information. The approach also requires a separate list call followed by in-memory string comparison, without any ability to distinguish which version of gh-aw created the item.

#### Alternative 2: External State Tracking in Target Repositories

Deduplication state could be stored as a committed file in each target repository (e.g., `.github/gh-aw-state.json`), recording the last seen gh-aw version and the number of the open issue or PR. This enables richer state tracking and avoids per-run API scans but requires write access to every target repository's default branch, introduces synchronization hazards when multiple org runner instances run concurrently, and couples the deduplication mechanism to the repository's file tree rather than to the items themselves.

### Consequences

#### Positive
- Deduplication is reliable: marker strings are embedded in item body content, which is stable even when titles are edited
- Version-stamped markers let operators and automation determine which gh-aw release created a given item without inspecting git history
- The `agentic-workflows` label enables GitHub search and filtering across the org (`is:open label:agentic-workflows`)
- Release links in issue/PR bodies give maintainers direct access to the gh-aw changelog for the triggering version

#### Negative
- Closing stale items requires fetching up to 100 open issues or PRs per target repository on every org runner invocation; this is an O(n) operation that grows with the repository's open item count and adds API calls before the actual create
- If the gh-aw releases API is unavailable, release links are silently omitted and the marker falls back to `latest`; maintainers may see inconsistent body formats across runs

#### Neutral
- Shared helper functions (`buildOrgXMLMarker`, `closeExistingOrgIssuesByMarker`, `closeExistingOrgPRsByMarker`, `addLabelToOrgPR`, `createOrgIssue`) are extracted into `pkg/cli/org_issue_pr_helpers.go`, making the deduplication policy a single code path that both update and upgrade flows depend on
- HTML comment markers are invisible when issues/PRs are rendered in the GitHub UI but are present in raw body content; this is a common convention but may surprise contributors reading raw API responses

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
