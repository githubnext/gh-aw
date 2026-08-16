# ADR-53196: Typed Top-Level Frontmatter Fields for `ambient-folders` and `github-app`

**Date**: 2026-08-16
**Status**: Accepted
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`FrontmatterConfig` is the central typed struct that represents parsed workflow YAML frontmatter in `pkg/workflow`. Two schema-backed top-level keys — `ambient-folders` (a list of paths the agent may read without explicit tool calls) and `github-app` (GitHub App credentials for minting scoped tokens) — were absent from this struct. Typed consumers could not observe these fields without reaching directly into an untyped `map[string]any`, which duplicates parsing logic and bypasses validation. The `GitHubAppConfig` struct and its `parseAppConfig` helper already existed in `safe_outputs_app_config.go`; ambient folder extraction and normalization already existed in `ambient_folders.go`.

### Decision

Add `AmbientFolders []string` and `GitHubApp *GitHubAppConfig` fields to `FrontmatterConfig`. `ParseFrontmatterConfig()` normalizes ambient folders with the same path validation and deduplication used by the compiler. It exposes a top-level GitHub App only when both required credentials are present, matching the compiler fallback path even when `ignore-if-missing` is enabled.

`ToMap()` serializes both fields in raw-frontmatter-compatible forms without retaining caller-owned slices or maps. The deprecated `app-id` input key is accepted and normalized to the canonical `client-id` key. `GitHubAppConfig` uses matching JSON and YAML field names for direct consumers.

### Alternatives Considered

#### Alternative 1: Untyped raw-map access at call sites

Callers could reach into the raw frontmatter `map[string]any` directly whenever they need `ambient-folders` or `github-app`. This avoids modifying `FrontmatterConfig` but scatters parsing and coercion logic across the codebase, making field access inconsistent and untestable in isolation.

#### Alternative 2: Separate accessor functions outside `FrontmatterConfig`

Introduce standalone helper functions (`GetAmbientFolders(fc *FrontmatterConfig) []string`) rather than struct fields, keeping the struct lean. This preserves backward compatibility at the struct level but creates a parallel access pattern that is inconsistent with how all other configuration sections (Tools, Secrets, Engine, etc.) are exposed — every other feature uses a named struct field.

### Consequences

#### Positive
- Typed access to `ambient-folders` and `github-app` is now available to all consumers of `FrontmatterConfig` without duplicating parse logic.
- Round trips produce independent slices and maps, and the `app-id` alias is normalized to `client-id`.
- Typed ambient folders have the same normalized values as compiler consumers.
- Typed and compiler consumers agree on whether a top-level GitHub App is present.
- The `parseAppConfig` helper is now exercised through the typed frontmatter path, giving it broader test coverage via the new `frontmatter_types_test.go` cases.

#### Negative
- `FrontmatterConfig` must be extended for each new top-level frontmatter key that needs typed exposure; the struct will grow over time as the schema evolves.
- Typed parsing depends on the compiler's top-level GitHub App extraction semantics so the two paths cannot disagree.

#### Neutral
- `ToMap()` emits `[]any` and `map[string]any` for nested GitHub App collections, matching raw frontmatter representation.
- No changes to the YAML schema or external API contracts are introduced; this is purely a typed-struct promotion of existing documented fields.
