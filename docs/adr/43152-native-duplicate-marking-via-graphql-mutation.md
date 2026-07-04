# ADR-43152: Use GitHub GraphQL `markAsDuplicate` Mutation for Native Duplicate Relationships

**Date**: 2026-07-03
**Status**: Draft
**Deciders**: pelikhan (PR author)

---

### Context

The `close_issue` safe output handler supports closing issues with `state_reason: DUPLICATE`, but previously this only set the issue's state reason via the REST API and optionally posted a text comment. No native GitHub "marked this as a duplicate of #X" timeline event was created. As a result, the duplicate relationship was invisible in the issue timeline unless the closing comment manually mentioned the canonical issue — and even then, the link was only a text reference, not a tracked relationship. Teams triaging duplicate issues had no reliable programmatic way to discover which canonical issue an issue was closed as a duplicate of.

### Decision

We will add an optional `duplicate_of` field to the `close_issue` tool schema. When `duplicate_of` is provided together with `state_reason: DUPLICATE`, the handler will resolve the reference (supporting bare numbers, `#N`, `owner/repo#N`, and full GitHub issue URLs) and call the GitHub GraphQL `markAsDuplicate` mutation after closing the issue. This creates a native "marked this as a duplicate of #X" timeline event on the issue. Mutation failures are treated as non-fatal: a warning is logged and the close operation still succeeds so that missing GraphQL permissions cannot break issue automation.

### Alternatives Considered

#### Alternative 1: Parse the Closing Comment Body to Infer the Duplicate Target

Parse `body` for issue references (e.g. `#123`, `github/gh-aw#123`) and automatically call `markAsDuplicate` when `state_reason: DUPLICATE` is set and a reference is found. This avoids adding a new field, keeping the interface simpler.

Rejected because comment format varies widely (agents may phrase the body differently each time), making reliable extraction fragile. It also creates implicit, surprising behavior — the mutation would be called based on incidental comment phrasing rather than an explicit user intent, and false positives could mark wrong duplicates.

#### Alternative 2: Require Only Full GitHub Issue URLs for `duplicate_of`

Accept `duplicate_of` as a string but restrict valid inputs to full `https://github.com/owner/repo/issues/N` URLs only, rejecting bare numbers or shorthand references.

Rejected because the most common case — closing a duplicate within the same repository — benefits greatly from bare number syntax (e.g. `duplicate_of: 123`). Forcing full URLs adds friction for the common case without a meaningful safety benefit, since all formats ultimately resolve to the same node ID pair.

#### Alternative 3: Use a REST API Endpoint for Marking Duplicates

Use the GitHub REST API instead of GraphQL to create the duplicate relationship, avoiding the complexity of a GraphQL call in the handler.

Rejected because GitHub does not expose `markAsDuplicate` via REST. The operation is only available through the GraphQL API, so GraphQL is the only viable path.

### Consequences

#### Positive
- Issues closed as duplicates now display the native "marked this as a duplicate of #X" timeline event, making the canonical issue discoverable directly from the issue timeline.
- Eliminates the need for a separate `add-comment` step or label to record the duplicate linkage — the native relationship is created atomically with the close.
- The accepted reference formats (bare number, `#N`, `owner/repo#N`, full URL) cover all common usage patterns including cross-repository duplicates.

#### Negative
- The mutation is non-fatal by design; if the caller lacks `graphql` permission or the canonical issue is inaccessible, the timeline event is silently skipped with only a warning log. The issue is still closed, but the timeline relationship may be absent without obvious error to the end user.
- Adds REST round-trips (two `issues.get` calls to fetch node IDs) before the GraphQL mutation, increasing latency and API quota usage when `duplicate_of` is supplied.

#### Neutral
- The `duplicate_of` field is entirely optional and backward-compatible; existing `close_issue` usage without this field is unaffected.
- Both the runtime tool schema (`safe_outputs_tools.json`) and the compiler copy (`pkg/workflow/js/safe_outputs_tools.json`) must be kept in sync — this dual-copy pattern is pre-existing and not introduced by this change.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
