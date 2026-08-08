# ADR-51292: Increase Harness Watchdog Default and Add Frontmatter Timeout Control

**Date**: 2026-08-08
**Status**: Draft
**Deciders**: pelikhan

---

### Context

The post-result harness watchdog kills any engine process that is silent (no stdio) for longer than a configured idle timeout after it has emitted a terminal safe output. The previous default was 20 seconds. In practice, large-repo workflows commonly pass through quiet shell phases (package installs, builds, index refreshes) that comfortably exceed 20 seconds without producing output, causing the watchdog to send a premature SIGTERM and leave runs incomplete.

The only existing override surface was `GH_AW_HARNESS_WATCHDOG_TIMEOUT_MS` set via `engine.env`, which requires the caller to know the env var name, know the unit is milliseconds, and accept that the override is indistinguishable from other env-var customizations. This surface is not surfaced in schema validation or reference docs in a discoverable way.

### Decision

We will raise `DEFAULT_POST_RESULT_WATCHDOG_IDLE_TIMEOUT_MS` from 20 seconds (20 000 ms) to 2 minutes (120 000 ms) in `process_runner.cjs` — the shared location read by all harnesses — and we will add `engine.harness.watchdog-timeout` as a first-class frontmatter integer field (unit: seconds) that compiles to `GH_AW_HARNESS_WATCHDOG_TIMEOUT_MS` (unit: milliseconds). The new field sits alongside the existing retry-policy fields under `engine.harness`, follows the same literal-or-expression pattern, and is validated by the main workflow JSON schema.

### Alternatives Considered

#### Alternative 1: Raise the default only, no new frontmatter field

Increasing the default to 120 s without adding a frontmatter override surface would fix the most common regression at minimal API cost. Workflows with genuine requirements for a shorter or longer timeout would still have to fall back to the raw `engine.env.GH_AW_HARNESS_WATCHDOG_TIMEOUT_MS` override, with its undiscoverable name and millisecond unit. This option trades surface simplicity for discoverability and per-workflow configurability.

#### Alternative 2: Keep the 20 s default, improve env-var documentation only

Documenting `GH_AW_HARNESS_WATCHDOG_TIMEOUT_MS` more prominently in reference docs and adding schema validation for it as an `engine.env` value would let advanced users tune the timeout without changing the default. This avoids the risk that a longer default masks genuinely stuck processes, but does not address the root cause: the 20 s window is too short for routine large-repo operations and results in hard-to-diagnose incomplete runs.

### Consequences

#### Positive
- Premature watchdog kills during legitimate quiet phases (installs, builds, index refreshes) are eliminated for typical large-repo workflows under the new 2-minute default.
- Workflow authors can tune per-workflow watchdog behavior via a discoverable, schema-validated frontmatter field without memorising env var names or unit conversions.
- Explicit `engine.env.GH_AW_HARNESS_WATCHDOG_TIMEOUT_MS` overrides continue to take precedence, preserving backward compatibility for callers already using the env-var surface.

#### Negative
- A genuinely stuck post-result process now takes up to 2 minutes (instead of 20 seconds) to be forcibly terminated under the new default, increasing the worst-case cost of a hung run.
- Adding `watchdog-timeout` to `engine.harness` extends the frontmatter API surface; future breaking changes to this field require a deprecation path.

#### Neutral
- The `watchdog-timeout` frontmatter field accepts seconds; the runtime converts to milliseconds before injecting `GH_AW_HARNESS_WATCHDOG_TIMEOUT_MS`. This unit difference is intentional (seconds are more human-readable in config) and is documented in schema and reference docs.
- The existing `MIN_POST_RESULT_WATCHDOG_TIMEOUT_MS` (50 ms) and `MAX_POST_RESULT_WATCHDOG_TIMEOUT_MS` (10 min) guards are unchanged and continue to prevent obviously invalid overrides.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
