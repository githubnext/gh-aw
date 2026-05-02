# ADR-29663: Default Sandbox Agent Type to AWF for Ambiguous Configurations and Require Explicit ID in Strict Mode

**Date**: 2026-05-02
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `sandbox.agent` configuration in gh-aw workflow files allows users to customize the AWF (Agent Workflow Firewall) sandbox that isolates agent execution. A user can specify just a version pin — for example `{ version: "v0.25.29" }` — without providing an explicit `id` or `type` field, intending to use the AWF sandbox at a specific version. However, `getAgentType()` returns an empty string when neither `ID` nor `Type` is set on `AgentSandboxConfig`, and `isSupportedSandboxType("")` returns `false`, which caused `isSandboxEnabled()` to return `false`. This silently disabled the AWF firewall and ran the agent on the host runner, breaking MCP connectivity (the "smoke-gemini" incident). `applySandboxDefaults` only defaulted the agent type to AWF for a `nil` agent config, not for a non-nil agent config with an empty type.

### Decision

We will default `sandbox.agent.type` to `awf` whenever `sandbox.agent` is a non-nil, non-disabled object that carries no explicit `id` or `type`. This is a fail-safe default: a user who writes any `sandbox.agent` block clearly intends to use a sandbox, so we treat an absent `id`/`type` as an implicit AWF selection rather than as disabled. Additionally, we will reject `sandbox.agent` configurations without an explicit `id` in strict mode, where ambiguous configurations are not acceptable and must fail loudly rather than silently defaulting.

### Alternatives Considered

#### Alternative 1: Fail-Closed — Reject Ambiguous Configurations in All Modes

Require `id: awf` explicitly in every `sandbox.agent` block, in both normal and strict mode. Return a validation error for any agent object with no `id`/`type`.

This was not chosen for non-strict mode because it would break existing workflow files that pin a version without an explicit id — a common and reasonable pattern. The intent is unambiguous enough (user provided an agent block) to justify defaulting. Failing closed in non-strict mode would be a breaking change with no security benefit, since the only ambiguity is *which* sandbox type, not whether a sandbox should run.

#### Alternative 2: Document the Behavior and Do Nothing

Accept the existing behavior where a version-only agent config disables the sandbox, and document this as a known footgun. Users would need to always write `id: awf` explicitly to opt into the sandbox.

This was not chosen because the prior behavior (silently disabling the firewall) is a security regression. Any `sandbox.agent` object signals clear intent to run in a sandbox; treating it as disabled is surprising and unsafe. The cost of changing the behavior is minimal while the security benefit (always-on firewall when configured) is high.

### Consequences

#### Positive
- Workflows that pin an AWF version without an explicit `id` now correctly run inside the sandbox, closing the silent firewall-bypass footgun.
- Strict mode now provides a clear, actionable error when `sandbox.agent` lacks an explicit `id`, preventing ambiguous configurations from reaching production.
- The default behavior is "more sandboxed" rather than "less sandboxed", which is the safer default for a security boundary.

#### Negative
- Any workflow that intentionally relied on the empty-type behavior to bypass the sandbox (i.e., accidentally "using" the bug as a feature) will now have the sandbox re-enabled. Such workflows must add `disabled: true` explicitly if they truly intend to skip the sandbox.
- Strict mode is now more restrictive for `sandbox.agent` blocks: workflows that previously passed strict validation with a version-only agent must be updated to add `id: awf`.

#### Neutral
- The `hourly-ci-cleaner.md` workflow required a one-line fix (`id: awf`) to conform to strict mode after this change was introduced.
- `applySandboxDefaults` now has an additional code path; the behavior is guarded by `isSupportedSandboxType(getAgentType(...))`, reusing existing helper functions without adding new abstractions.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Sandbox Agent Default Behavior

1. When `sandbox.agent` is a non-nil object and `Disabled` is `false`, `applySandboxDefaults` **MUST** set `Agent.Type` to `awf` if `isSupportedSandboxType(getAgentType(agent))` returns `false`.
2. `applySandboxDefaults` **MUST NOT** override `Agent.Type` when the agent is explicitly disabled (`Disabled == true`).
3. `applySandboxDefaults` **MUST NOT** override an already-valid, explicitly-set `Agent.Type` (i.e., when `isSupportedSandboxType(getAgentType(agent))` returns `true`).
4. The AWF default **SHOULD** be applied before returning from `applySandboxDefaults` so that downstream consumers always receive a fully-resolved sandbox configuration.

### Strict Mode Validation

1. In strict mode, `validateStrictSandboxCustomization` **MUST** return an error if `sandbox.agent` is non-nil, `Disabled` is `false`, and `isSupportedSandboxType(getAgentType(agent))` returns `false`.
2. The error message **MUST** instruct the user to add `id: awf` explicitly to their `sandbox.agent` configuration.
3. A `sandbox.agent` block with `Disabled: true` **MUST NOT** trigger this strict-mode validation error; the disabled-agent case is handled by `validateStrictFirewall`.
4. Workflow files that use the AWF sandbox in strict mode **MUST** include an explicit `id: awf` field in their `sandbox.agent` block.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25240361053) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
