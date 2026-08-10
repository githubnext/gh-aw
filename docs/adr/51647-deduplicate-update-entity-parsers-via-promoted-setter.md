# ADR-51647: Deduplicate Update-Entity Parsers via Promoted Setter and Go Generic Constraint

**Date**: 2026-08-10
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/workflow` contains four update-entity parsers (`parseUpdateIssuesConfig`, `parseUpdateDiscussionsConfig`, `parseUpdatePullRequestsConfig`, `parseUpdateReleaseConfig`), each a thin wrapper around the shared `parseUpdateEntityConfigTyped` helper. Two problems existed:

1. `parseUpdateEntityConfigTyped` used a `switch v := any(cfg).(type)` to copy the parsed base config (`max`, `target`, `target-repo`) into the concrete struct. A new entity type that forgot to add a `case` would parse successfully but silently drop all three fields — a correctness bug waiting to happen.
2. The footer field spec (`{Name: "footer", Mode: FieldParsingTemplatableBool, StringDest: &cfg.Footer}`) was copy-pasted verbatim into all four parsers. Any change to how footer is parsed (mode, type, default) had to be applied in four places.

### Decision

We will eliminate the type-switch dispatch by adding a second generic parameter `PT` constrained to `interface { *T; updateEntityConfigSetter }`. A promoted `setUpdateEntityConfig` method on `UpdateEntityConfig` satisfies `updateEntityConfigSetter` for every concrete struct that embeds it. The helper calls `PT(cfg).setUpdateEntityConfig(*baseConfig)` instead of the switch, making a missing embedding a compile error. We will also extract the footer field spec into `updateEntityFooterField(&cfg.Footer)` so it is defined once.

### Alternatives Considered

#### Alternative 1: Add compile-time exhaustiveness assertions to the type switch

A `var _ updateEntityConfigSetter = (*UpdateIssuesConfig)(nil)` assertion per concrete type could turn the missing-case problem into a compile error without changing the generic function signature. This is simpler — the switch remains readable — but it still requires a human to remember to add a new assertion for every new entity type and does not remove the four-way switch boilerplate itself.

#### Alternative 2: Replace per-entity parsers with a declarative registry table

Define a map or slice keyed by `UpdateEntityType` that carries each parser's entity-type constant, output key, logger, field-spec callback, and optional post-processor. A single generic dispatcher iterates the table, eliminating all four wrapper functions entirely. This is the most thorough deduplication but is a substantially larger refactor, touches more files, and blurs the boundary between entity-specific logic and shared infrastructure.

### Consequences

#### Positive
- A new entity config struct that forgets to embed `UpdateEntityConfig` now fails at compile time instead of silently dropping `max`, `target`, and `target-repo` at runtime.
- The footer field spec is defined once; a change to its parsing mode or destination type is a one-line edit rather than four synchronized changes.
- No call sites of `parseUpdateEntityConfigTyped` changed — `PT` is inferred by the compiler from the concrete `*T` argument.

#### Negative
- The two-parameter generic signature (`[T any, PT interface { *T; updateEntityConfigSetter }]`) is significantly harder to read than `[T any]`, especially for contributors unfamiliar with the Go "pointer receiver on embedded type" pattern.
- Understanding why the method is promoted (rather than defined directly on each concrete type) requires knowing Go struct embedding semantics; this is non-obvious to maintainers encountering the pattern for the first time.

#### Neutral
- The `updateEntityConfigSetter` interface and `updateEntityFooterField` helper are package-private (`update_entity_helpers.go`); they do not widen the public API surface.
- Test coverage was added (`TestParseUpdateEntityConfigTypedBaseConfigAssignment`) verifying base-config propagation and footer parsing for all four entity types.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
