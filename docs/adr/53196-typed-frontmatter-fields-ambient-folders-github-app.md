# ADR-53196: Typed Top-Level Frontmatter Fields for `ambient-folders` and `github-app`

**Date**: 2026-08-16
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`FrontmatterConfig` is the central typed struct that represents parsed workflow YAML frontmatter in `pkg/workflow`. Two schema-backed top-level keys — `ambient-folders` (a list of paths the agent may read without explicit tool calls) and `github-app` (GitHub App credentials for minting scoped tokens) — were absent from this struct. Typed consumers such as the safe-outputs pipeline and the deep-report workflow could not observe these fields without reaching directly into an untyped `map[string]any`, which duplicates parsing logic and bypasses validation. The `GitHubAppConfig` struct and its `parseAppConfig` helper already existed in `safe_outputs_app_config.go` for a related purpose; `AmbientFolders` parsing also needed a dedicated extractor to handle `[]any` → `[]string` coercion.

### Decision

We will add `AmbientFolders []string` and `GitHubApp *GitHubAppConfig` fields to `FrontmatterConfig`, wire them into `ParseFrontmatterConfig()` using the existing `extractAmbientFolders` and `parseAppConfig` helpers, and serialize them back through `ToMap()` via a new `githubAppConfigToMap` function. The deprecated `app-id` input key is accepted on parse and normalized to the canonical `client-id` on serialization.

### Alternatives Considered

#### Alternative 1: Untyped raw-map access at call sites

Callers could reach into the raw frontmatter `map[string]any` directly whenever they need `ambient-folders` or `github-app`. This avoids modifying `FrontmatterConfig` but scatters parsing and coercion logic across the codebase, making field access inconsistent and untestable in isolation.

#### Alternative 2: Separate accessor functions outside `FrontmatterConfig`

Introduce standalone helper functions (`GetAmbientFolders(fc *FrontmatterConfig) []string`) rather than struct fields, keeping the struct lean. This preserves backward compatibility at the struct level but creates a parallel access pattern that is inconsistent with how all other configuration sections (Tools, Secrets, Engine, etc.) are exposed — every other feature uses a named struct field.

### Consequences

#### Positive
- Typed access to `ambient-folders` and `github-app` is now available to all consumers of `FrontmatterConfig` without duplicating parse logic.
- Round-trip fidelity is guaranteed: `ParseFrontmatterConfig(fc.ToMap())` returns equivalent values, and the `app-id` alias is normalized to `client-id` on the way out.
- The `parseAppConfig` helper is now exercised through the typed frontmatter path, giving it broader test coverage via the new `frontmatter_types_test.go` cases.

#### Negative
- `FrontmatterConfig` must be extended for each new top-level frontmatter key that needs typed exposure; the struct will grow over time as the schema evolves.
- `parseAppConfig` (originally private to `safe_outputs_app_config.go`) is now shared across two parsing contexts (safe-outputs and general frontmatter), coupling those subsystems through a package-level function.

#### Neutral
- The type-switch idiom (`case []string:`, `case map[string]string:`) is adopted in `safe_outputs_app_config.go` to handle values that may already be typed (e.g., after a round-trip through `ToMap()`), which is an internal implementation detail not visible to callers.
- No changes to the YAML schema or external API contracts are introduced; this is purely a typed-struct promotion of existing documented fields.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
