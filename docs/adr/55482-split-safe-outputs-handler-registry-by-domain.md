# ADR-55482: Split Safe Outputs Handler Registry by Domain

**Date**: 2026-08-24
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/workflow/safe_outputs_handler_registry.go` accumulated all safe-output handler builders in a single monolithic `map[string]handlerBuilder` literal that grew to 1 000+ lines. Reviewing, editing, or adding handlers required navigating a single large file with no logical grouping, making domain-specific changes error-prone and PR reviews difficult. The handlers naturally cluster by GitHub entity: issues, discussions, pull requests, workflow-level actions, projects, assignments, comments, releases, and diagnostics.

### Decision

We will decompose the single `handlerRegistry` map into focused sub-registries (one per domain), each in its own file, and compose them at package init time via a new `mergeHandlerMaps(...)` helper in the existing file. The resulting `handlerRegistry` variable and all call sites remain unchanged; only the file layout and initialization sequence change.

### Alternatives Considered

#### Alternative 1: Keep the monolith, improve navigation with comments/regions

Add prominent section dividers and a table-of-contents comment. No structural change; IDE jump-to-definition still works. Rejected because the file continues to grow with each new handler and the readability problem recurs with every future domain.

#### Alternative 2: Plugin/interface-based extensible registry

Define a `HandlerProvider` interface; each domain registers via `init()`. Fully decoupled, but requires a registry-registration protocol, initialization-order care, and significantly more boilerplate for a problem that does not require runtime extensibility. Over-engineered for a compile-time-only registry.

### Consequences

#### Positive
- Domain handlers can be reviewed, tested, and changed in isolation without touching an unrelated 1 000-line file.
- `mergeHandlerMaps` explicitly detects duplicate handler keys and logs a warning rather than silently overwriting, reducing accidental collision risk.
- Smaller per-file diffs improve PR review clarity.
- New registry test coverage verifies domain membership, full composition, builder enable/disable behavior, duplicate-key handling, and token-helper behavior.

#### Negative
- `mergeHandlerMaps` applies a first-wins policy on key collisions (logged, not panicked). A future developer who inadvertently registers the same key in two domain files will only see a log line rather than a compile-time or startup error.
- Registry initialization order (the order of arguments to `mergeHandlerMaps`) is now implicit; the order matters for collision resolution but is not enforced by type system or test.

#### Neutral
- All existing call sites (`handlerRegistry[key]`, `handlerSupportsPerHandlerGitHubAppToken`, etc.) are unchanged; this is a pure internal reorganization.
- The `add_comment` handler lives in `commentHandlerRegistry` because it applies to both issues and discussions; this placement is a documentation convention, not enforced by code.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
