# ADR-47045: Allow close_issue Agents to Select State-Reason at Runtime

**Date**: 2026-07-21
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `close_issue` safe-output tool previously only accepted a scalar `state-reason` configuration, which fixed the closing reason at workflow-author time. This prevented agents from selecting `duplicate` even when they identified clear duplicate evidence during triage. Workflow authors need three distinct operating modes: a locked reason for strict workflows, a restricted subset for mixed-intent workflows, and full agent freedom for general-purpose agents. The existing single-field config cannot express the omitted (agent-free-choice) and list (agent-restricted) modes, forcing a workaround or silent misconfiguration.

### Decision

We will extend `CloseEntityConfig` with a parallel `StateReasons []string` field and a YAML preprocessing step that converts a list-form `state-reason: [...]` value to the `state-reasons` key before unmarshaling. We will also add a `PropertyInjections` map to `ToolsMeta` so the Go compiler can inject a `state_reason` enum into the `close_issue` tool's JSON Schema at compile time, reflecting the permitted values for the configured mode. The JS handler validates agent-provided `state_reason` values against the permitted set at runtime and throws on invalid choices so the wrapper surfaces `{success: false}`.

### Alternatives Considered

#### Alternative 1: Add a separate `allow-duplicate` boolean flag

Add a single `allow-duplicate: true` flag alongside the existing scalar `state-reason`. This is simpler but does not generalize: it cannot restrict the agent to a specific subset (e.g., `[not_planned, duplicate]`), and does not address the need to allow `completed` as an agent-selectable option. It would require additional flags for each future per-reason use case, creating an unbounded set of boolean knobs.

#### Alternative 2: Remove `state-reason` config and always allow agent to choose

Remove the scalar `state-reason` entirely so agents always choose from all supported values. This simplifies the config surface but breaks backward compatibility for workflows that rely on a fixed closing reason (e.g., always `not_planned` regardless of agent intent), which is the existing documented behavior. It also removes the ability for strict workflows to enforce a single reason.

### Consequences

#### Positive
- Agents can now select `duplicate` (or any configured subset) when appropriate, enabling correct native duplicate linking via `duplicate_of`
- Fully backward-compatible: existing scalar `state-reason: value` configs continue to behave identically
- The `close_issue` tool's JSON Schema reflects the actual permitted enum at runtime, giving the agent accurate self-documentation
- The three-way contract (scalar / list / omitted) is explicit and tested across all code layers (Go config parsing, JS handler, tool schema generation)

#### Negative
- Two parallel YAML keys (`state-reason` for scalar and `state-reasons` for list) coexist in the config struct; the list form is authored as `state-reason: [...]` but stored under `state-reasons`, which is non-obvious
- YAML preprocessing adds a non-obvious indirection in the config parser that future maintainers must understand before modifying config deserialization
- Property injection via `computePropertyInjections` is currently hardcoded to the `close_issue`/`state_reason` case; any future dynamic-enum tool will require extending this function

#### Neutral
- Eight new Go unit tests and several JS unit tests are added to cover the three config modes and edge cases
- Existing lock files for deployed workflows are regenerated to include the new `property_injections` field, reflecting the omitted-config default (all three values)

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
