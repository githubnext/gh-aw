# ADR-54656: Unify close-older Config Fields via Shared Struct Embed

**Date**: 2026-08-22
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`create-issue`, `create-discussion`, and `create-pull-request` each independently declared close-older enable (`CloseOlderIssues`, `CloseOlderDiscussions`, `CloseOlderPullRequests`) and key (`CloseOlderKey`) fields on their respective config structs. Downstream handler consumers in `safe_outputs_handler_registry.go` accessed these fields through parallel but structurally identical code paths. Any behavioural change to close-older logic required coordinated edits across three config struct definitions and three consumer call sites, with no compiler enforcement that they stayed in sync.

### Decision

We will introduce a shared `CloseOlderConfig` struct in `pkg/workflow/create_entity_helpers.go` containing canonical `Enabled *string` and `Key string` fields, and embed it inline into `CreateIssuesConfig`, `CreateDiscussionsConfig`, and `CreatePullRequestsConfig`. Existing public YAML keys (`close-older-issues`, `close-older-discussions`, `close-older-pull-requests`) are preserved: `Enabled` is tagged `yaml:"-"` so it is never directly settable from workflow frontmatter, and is instead populated after unmarshaling via `closeOlderEnabledFromConfigData`, which reads the already-preprocessed entity-specific value out of the raw config map. All downstream consumers are updated to read from `CloseOlderConfig.Enabled` and `CloseOlderConfig.Key`.

### Alternatives Considered

#### Alternative 1: Shared accessor functions without struct consolidation

Keep per-entity fields on each config struct but introduce a shared interface or helper functions to access them uniformly. This would allow downstream code to call a common accessor rather than field-by-field paths, without changing the struct layout.

**Why not chosen**: The struct duplication itself is the problem — the accessors would hide it but not eliminate it. New close-older options would still require changes in three places. The embed approach removes that duplication at the type level, giving the compiler the ability to catch missed updates.

#### Alternative 2: Unified YAML key without backward-compatible aliasing

Replace the three entity-specific YAML keys with a single `close-older-enabled` key across all handler configs, removing the aliasing step.

**Why not chosen**: This would be a breaking change for all existing workflow YAML files that use `close-older-issues`, `close-older-discussions`, or `close-older-pull-requests` keys. The aliasing approach achieves consolidation internally while keeping the public API stable, which is required for backward compatibility.

### Consequences

#### Positive
- Single source of truth for close-older configuration: `CloseOlderConfig` is defined once and embedded by reference everywhere.
- Downstream consumers (`safe_outputs_handler_registry.go`) read from a uniform path (`c.CloseOlderConfig.Enabled`, `c.CloseOlderConfig.Key`) regardless of the originating handler type.
- New close-older fields only need to be added to `CloseOlderConfig` to propagate across all three create handlers automatically.
- Parser coverage for all three aliasing paths is validated by dedicated tests in `create_close_older_config_test.go`.

#### Negative
- Post-unmarshal population introduces a small amount of indirection: `Enabled` is not set by YAML unmarshaling directly but by an explicit call to `closeOlderEnabledFromConfigData` in each handler's `postUnmarshal` callback, reading the already-preprocessed entity-specific key. A reader unfamiliar with this pattern may be confused that `CloseOlderConfig.Enabled` has no YAML tag despite being populated from YAML input.
- The embed uses `yaml:",inline"`, which means YAML tags on `CloseOlderConfig` fields coexist in the same namespace as the parent struct's tags; tag conflicts in future fields require care.

#### Neutral
- Existing per-entity YAML keys continue to work unchanged; no migration of consumer configs is required.
- The `close-older-key` YAML tag is shared across all three handlers through the embed, making its semantics consistent by construction.
- `isCloseOlderPullRequestsEnabled` is updated to dereference `config.CloseOlderConfig.Enabled` instead of the old `config.CloseOlderPullRequests` field.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
