# ADR-68504: Capability-Gated `engine.max-context-rebuild` Frontmatter Setting

**Date**: 2026-10-13
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent (PR #68504)

---

### Context

The Codex harness ships a context-rebuild circuit breaker that stops a run when the cumulative-input-to-largest-request ratio (`rebuild_factor`) and cumulative input tokens both cross thresholds. Until now the only way to tune that factor was the raw `engine.env.GH_AW_CODEX_MAX_REBUILD_FACTOR` environment variable, which is undiscoverable from the frontmatter schema, unvalidated at compile time, and Codex-specific with no path for other engines. Separately, when the guard tripped, the run surfaced only a `report_incomplete` with `reason: infrastructure_error`, which dashboards cannot distinguish from a generic process crash. The compiler already has a capability-gating pattern (`max-turns`, `max-continuations`, `context-window`) for engine-specific frontmatter, and declarative/third-party engines are defined through `behaviors.capabilities` plus `behaviors.execution` env-var mappings.

### Decision

We will add a first-class numeric `engine.max-context-rebuild` frontmatter setting (minimum `1`, fractional values allowed), gated by a new `EngineCapabilities.MaxContextRebuild` flag and mapped to a runtime environment variable through a new `ContextRebuildEnvVarProvider` interface (`GetMaxContextRebuildEnvVarName()`). Codex returns `GH_AW_CODEX_MAX_REBUILD_FACTOR`; declarative engines opt in with `behaviors.capabilities.max-context-rebuild: true` plus `behaviors.execution.max-context-rebuild-env-var`, validated as a contract pair at engine-construction time. Workflows that set the value on an engine without the capability — including via imports, CLI overrides, and explicit threat-detection engines — are rejected at compile time. We will also emit a structured `agent.execution` event with category `context_rebuild_limit_exceeded` when the guard stops a run, preserved in both `agent-session.jsonl` and the unified `usage/aw_session.jsonl`.

### Alternatives Considered

#### Alternative 1: Keep `engine.env` as the only tuning surface

Continue documenting `GH_AW_CODEX_MAX_REBUILD_FACTOR` (and sibling guard variables) as `engine.env` entries. This requires no schema, capability, or interface changes and keeps all guard knobs in one place. Rejected because raw env vars receive no schema validation (a typo or `"0"` silently produces a broken or disabled guard), are absent from the generated frontmatter reference, and provide no mechanism for a non-Codex engine to advertise equivalent support. The PR deliberately keeps `engine.env` working and precedent-setting over the new field, so this alternative remains available for the advanced guard knobs that are not promoted.

#### Alternative 2: Hard-code the setting as Codex-only

Add `max-context-rebuild` to the schema and have the Codex engine read it directly, with a simple `id == "codex"` check for rejection, skipping the capability flag, the `ContextRebuildEnvVarProvider` interface, and the declarative `behaviors` contract. This is a much smaller diff. Rejected because it breaks the existing capability-gating convention used by `max-turns`/`max-continuations`, leaves third-party declarative engines with no opt-in path even when their runtime enforces an equivalent limit, and would need to be reworked the first time a second engine gains the feature.

#### Alternative 3: Enforce the limit in the compiler rather than the runtime

Have the compiler inject a generic circuit breaker around any engine that sets the factor, instead of delegating enforcement to the engine's harness. Rejected because measuring `rebuild_factor` requires per-request token-usage records that only the harness observes; a compiler-side implementation would duplicate harness logic and could not apply to third-party runtimes. The documented contract is explicit that the runtime is responsible for enforcement.

### Consequences

#### Positive
- The limit becomes a validated, documented, discoverable frontmatter setting: invalid values (`0`, negative, `0.5`, strings, expressions, booleans, null) fail at compile time instead of silently falling back to defaults at runtime.
- Declarative/third-party engines can opt in through `behaviors`, so the feature generalizes beyond Codex without further compiler changes.
- Guard cutoffs are now machine-identifiable via the `context_rebuild_limit_exceeded` category in both the canonical and unified session artifacts, letting dashboards separate them from crashes.

#### Negative
- Two overlapping configuration surfaces now exist for the same knob (`engine.max-context-rebuild` and `engine.env.GH_AW_CODEX_MAX_REBUILD_FACTOR`), with a precedence rule authors must learn; the remaining guard variables stay env-only, so the surface is inconsistent.
- The engine interface grows another provider method (`ContextRebuildEnvVarProvider`) that every engine must satisfy, increasing the per-engine implementation surface for a feature only one built-in engine supports.
- Capability/env-var contract validation adds another failure mode for declarative engine definitions (enabling the capability without a mapping, or mapping without the capability).

#### Neutral
- Omitting the setting preserves the existing runtime default of `25`, so no current workflow behaviour changes; the schema addition is additive and the changeset is `minor`.
- The generated frontmatter reference, engine capability matrix, Codex engine docs, and third-party agent docs all gained entries, so documentation drift checks will enforce future updates.
- Validation is applied in three places (post-engine validation, threat-detection engine config, and declarative engine construction), reflecting the existing multi-path engine resolution rather than a new architecture.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
