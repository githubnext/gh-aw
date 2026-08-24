# ADR-55532: Expose Threat-Detect Per-Attempt Controls via Frontmatter

**Date**: 2026-08-24
**Status**: Accepted
**Deciders**: gh-aw maintainers

---

### Context

`gh-aw-threat-detection` added per-attempt controls (`--engine-timeout`, `--max-turns`, `--retries`) to the `threat-detect` binary to let callers cap AI resource usage and retry behavior on each invocation. However, `gh-aw` had no mechanism to pass these controls per workflow — all compiled workflows invoked `threat-detect` with its binary compiled-in defaults and no override path. Workflow authors running cost- or time-sensitive security scans could not constrain the detector without patching the upstream binary. This PR closes that gap by surfacing the three flags in the `safe-outputs.threat-detection` frontmatter block and forwarding them to `threat-detect` only when explicitly configured.

### Decision

We will add three optional fields — `engine-timeout` (Go duration string or `0`), `max-turns` (non-negative integer), and `retries` (non-negative integer) — to the `safe-outputs.threat-detection` frontmatter section. When a field is set, the corresponding CLI flag is appended to the `threat-detect` invocation. When a field is absent, no flag is emitted and `threat-detect` applies its own compiled-in default. Schema-level validation rejects invalid values at compile time so errors surface before CI starts.

### Alternatives Considered

#### Alternative 1: Environment Variables

Pass per-workflow overrides via environment variables (e.g., `GH_AW_ENGINE_TIMEOUT`, extending the existing `GH_AW_MAX_TURNS` pattern). Environment variables are already used for some `threat-detect` settings. This approach was not chosen because env vars are process-wide and bleed across concurrent jobs or child processes running in the same environment; they cannot be scoped to a single `threat-detect` invocation without additional shell gymnastics. CLI flags are explicit, per-invocation, and composable, making the intent and scope unambiguous.

#### Alternative 2: Global Workflow-Level Frontmatter Controls

Expose `engine-timeout`, `max-turns`, and `retries` at the top-level workflow frontmatter rather than scoped under `safe-outputs.threat-detection`. This was considered to keep the API surface flat. It was not chosen because these controls are threat-detector-specific operational parameters, not workflow-wide settings. Placing them under `threat-detection` config preserves clear semantic ownership, avoids any risk of the values being misapplied to other engines or compile steps, and keeps the footprint of the change minimal.

### Consequences

#### Positive
- Workflow authors can cap threat-detection engine runtime, AI turn count, and retry attempts on a per-workflow basis, enabling cost and time governance without upstream changes.
- Threat-detect compiled-in defaults remain unchanged for all existing workflows: fields omitted from frontmatter produce no flag emission, so backward compatibility is guaranteed.
- Schema validation at compile time means invalid values (negative integers, malformed duration strings) are caught before any CI runner is allocated.

#### Negative
- Three new schema fields increase the surface area of the workflow specification that must be kept in sync with `threat-detect`'s CLI interface over time.
- YAML's flexible type system (integers, strings, floats can all represent the value `0`) required a defensive multi-case parser (`parseThreatDetectionEngineTimeout`), adding implementation complexity and maintenance burden.

#### Neutral
- The `buildThreatDetectCommand` helper function extracted from the inline `fmt.Sprintf` call improves testability of command construction but is an internal package change with no external API impact.
- The PR references `GH_AW_MAX_TURNS` as a fallback in `threat-detect`; this ADR does not address whether the env-var fallback should eventually be deprecated in favour of the new frontmatter field.

---
