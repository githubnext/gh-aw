# ADR-43051: Make Harness Retry Policy Configurable via `GH_AW_HARNESS_*` Env Vars

**Date**: 2026-07-04
**Status**: Draft
**Deciders**: Unknown

---

### Context

The Copilot, Claude, and Codex harnesses each contained a hardcoded retry policy (3 retries, 5 s initial delay, ×2 backoff, 60 s cap). During sustained transient API outages the fixed window exhausted all retries before the agent could write safe outputs, leaving the run with no recoverable state. There was no way for a workflow author to widen the retry window without forking or replacing the entire harness script.

Additionally, the constants were duplicated across `copilot_harness.cjs`, `claude_harness.cjs`, and `codex_harness.cjs`, making future adjustments error-prone.

### Decision

We will extract a shared `harness_retry_config.cjs` module that:

1. Defines the four retry policy constants with their current values as defaults (`DEFAULT_MAX_RETRIES = 3`, `DEFAULT_INITIAL_DELAY_MS = 5000`, `DEFAULT_BACKOFF_MULTIPLIER = 2`, `DEFAULT_MAX_DELAY_MS = 60000`).
2. Reads optional overrides from `GH_AW_HARNESS_MAX_RETRIES`, `GH_AW_HARNESS_INITIAL_DELAY_MS`, `GH_AW_HARNESS_BACKOFF_MULTIPLIER`, and `GH_AW_HARNESS_MAX_DELAY_MS` environment variables.
3. Validates each value strictly (decimal-digit strings only; `Number.isSafeInteger` for integers; rejects formats like `1e3` or `0x10` to avoid ambiguous timer values that differ from what the operator typed; caps `max-retries` at 100; clamps `max-delay` to at least `initial-delay`).
4. Falls back silently to the default and emits a warning log on invalid input.
5. Exports a single `resolveRetryConfig(env?, logger?)` function consumed by all three harnesses.

We will also expose the four parameters as sub-keys under `engine.harness` in workflow frontmatter (`max-retries`, `initial-delay-ms`, `backoff-multiplier`, `max-delay-ms`) compiled to the corresponding `GH_AW_HARNESS_*` env vars. The legacy string form (`engine.harness: "file.cjs"`) maps to `engine.harness.use` and remains accepted for backward compatibility. All four fields are templatable integers, accepting literal values or `${{ ... }}` expressions.

### Alternatives Considered

#### Alternative 1: Per-harness env vars with no shared module

Each harness could read its own set of env vars independently without a shared module. This avoids a new file dependency but perpetuates the duplication problem and would require identical validation logic to be maintained in three places.

#### Alternative 2: Top-level frontmatter fields (`engine.harness-max-retries`)

The four parameters could be added as flat fields directly under `engine` (e.g. `engine.harness-max-retries`). This is simpler to parse but creates a flat namespace collision risk and separates the retry fields visually from `engine.harness.use`. Grouping them under `engine.harness` makes the relationship explicit.

#### Alternative 3: Only allow env var overrides via `engine.env`

Workflows can already set arbitrary env vars via `engine.env`. Documenting the `GH_AW_HARNESS_*` names in the reference docs would be sufficient without adding dedicated frontmatter fields. This was rejected because typed frontmatter fields provide schema validation, autocomplete, and expression-level templating that raw `engine.env` strings do not.

### Consequences

#### Positive
- Workflow authors can widen the retry window during sustained outages without replacing built-in harnesses.
- Single source of truth for retry policy constants eliminates drift across the three harness files.
- Strict decimal-digit parsing prevents surprising timer values from scientific-notation or hex-encoded overrides.

#### Negative
- Adding a new `require()` dependency in all three harnesses couples them to `harness_retry_config.cjs`; renaming or moving that file requires updating all three.
- The `MAX_RETRIES_CAP = 100` hard limit is an arbitrary constant; operators who set a value above the cap will have it clamped to 100 and will receive a warning log message, but there is no validation error at workflow-compile time.

#### Neutral
- Existing behavior is fully preserved: all defaults match the prior hardcoded constants, so workflows that do not set any `GH_AW_HARNESS_*` vars or `engine.harness` sub-keys are unaffected.
- The `GH_AW_HARNESS_*` vars remain the canonical mechanism; `engine.harness` sub-keys compile down to them, keeping the two surfaces consistent.

---

*ADR created by [copilot-swe-agent]. Review and finalize before changing status from Draft to Accepted.*
