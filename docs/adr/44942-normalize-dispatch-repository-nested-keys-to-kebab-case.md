# ADR-44942: Normalize `dispatch-repository` Nested Keys to Kebab-Case

**Date**: 2026-07-11
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `safe-outputs.dispatch-repository` configuration block was the only area in the gh-aw system that used snake_case for nested config keys (`event_type`, `allowed_repositories`). Every other safe-output feature uses kebab-case keys (e.g., `dispatch-workflow`, `call-workflow`). This inconsistency forced users to context-switch between naming conventions when writing frontmatter, and made documentation harder to reason about. Bringing `dispatch-repository` into line with the rest of the system reduces cognitive load and makes the configuration surface uniform.

### Decision

We will adopt `event-type` and `allowed-repositories` as the canonical kebab-case keys for `dispatch-repository` nested configuration, and retain `event_type` and `allowed_repositories` as deprecated snake_case aliases for backward compatibility. The parser will resolve the canonical key first; if absent, it falls back to the alias. When both are present, the canonical kebab-case value takes precedence. The JSON schema is updated to mark aliases as deprecated and to accept either form via `anyOf` required-field semantics. No existing workflows are broken.

### Alternatives Considered

#### Alternative 1: Keep snake_case as canonical, add kebab-case aliases

The existing snake_case keys (`event_type`, `allowed_repositories`) would remain primary, and kebab-case variants would be added as optional aliases. This approach would preserve full backward compatibility without deprecation warnings, but it would cement the inconsistency permanently and require new users to learn an exception to the system-wide kebab-case convention.

#### Alternative 2: Remove snake_case keys entirely (breaking change)

Drop `event_type` and `allowed_repositories` with no aliases. This gives the cleanest schema and eliminates dual-key maintenance burden, but it immediately breaks all existing workflows that use snake_case keys. Given that `dispatch-repository` was previously marked experimental, some breakage would be acceptable, but the number of affected workflows is unknown and a grace period is safer.

#### Alternative 3: Add a compile-time migration warning and deprecate over multiple releases

Emit a compiler warning when snake_case keys are detected (similar to existing experimental-feature warnings) and plan to remove them in a future major release. This is the most operationally careful approach but adds release-lifecycle complexity and ongoing warning noise for users who have not migrated, without a forcing function for removal.

### Consequences

#### Positive
- Consistent kebab-case key convention across the entire `safe-outputs` configuration surface, eliminating the single exception.
- Existing workflows using snake_case aliases continue to work without modification; migration is opt-in.
- Documentation, schema descriptions, and validation error messages now all use canonical kebab-case, reducing confusion for new users.

#### Negative
- The JSON schema and parser must handle two key forms per field (`event-type`/`event_type`, `allowed-repositories`/`allowed_repositories`), increasing schema complexity and review surface.
- Deprecated aliases create an ongoing maintenance burden: they must be supported indefinitely (or a future breaking-change ADR must supersede this one).
- Test coverage must now exercise canonical-over-alias precedence and alias-only fallback paths, adding test maintenance cost.

#### Neutral
- The experimental-feature compiler warning for `dispatch-repository` is removed as part of this change, reflecting the feature's promotion to stable. This is a related but separate concern bundled into this PR.
- `anyOf` required-field semantics in JSON Schema may interact unexpectedly with some schema-validation tooling that does not handle `anyOf` well; users relying on strict schema validators should test their setups.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
