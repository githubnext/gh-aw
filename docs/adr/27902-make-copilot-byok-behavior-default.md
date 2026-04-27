# ADR-27902: Make Copilot BYOK Behavior Default for `engine: copilot`

**Date**: 2026-04-22
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw compiler supports a Copilot engine (`engine: copilot`) that routes workflows through GitHub Copilot's BYOK (Bring Your Own Key) runtime. BYOK mode requires four coordinated behaviors: injecting a dummy `COPILOT_API_KEY` sentinel to activate AWF's BYOK runtime path, enabling the `cli-proxy` sidecar for authenticated network routing, forcing the Copilot CLI to install at `latest` (bypassing any pinned version), and providing a non-empty `COPILOT_MODEL` fallback since BYOK providers require an explicit model name. Previously, these four behaviors were gated behind an explicit `features.byok-copilot: true` feature flag, requiring every Copilot workflow author to opt in. In practice, every `engine: copilot` workflow requires BYOK behavior to function correctly in the AWF runtime, making the flag mandatory in all non-trivial cases.

### Decision

We will make all four BYOK behaviors unconditionally active for every `engine: copilot` workflow, removing the `features.byok-copilot` opt-in requirement. The `ByokCopilotFeatureFlag` constant is deprecated and has no effect; the compiler now treats Copilot BYOK as the engine's baseline. A `features-byok-copilot-removal` codemod is added to automatically strip the deprecated flag from existing workflows.

### Alternatives Considered

#### Alternative 1: Keep `features.byok-copilot` as an explicit opt-in flag

Maintain the status quo: authors must include `features.byok-copilot: true` to enable BYOK behaviors. This was rejected because every production `engine: copilot` workflow required the flag, making it de facto mandatory rather than optional. The opt-in created friction for new authors and caused subtle breakage (empty `COPILOT_MODEL`, wrong CLI version, missing cli-proxy) when the flag was omitted.

#### Alternative 2: Introduce a separate `byok` sub-field on `engine: copilot`

Expose BYOK as an engine-level option (e.g., `engine: copilot\nbyok: true`) rather than a feature flag. This was not chosen because it would be another breaking rename with no behavioral advantage over the chosen approach, and it would still leave a gate that has no valid "off" state for standard AWF Copilot deployments.

### Consequences

#### Positive
- Copilot workflows work correctly out of the box without manual feature-flag wiring.
- Reduces the surface area of the feature flags system; one fewer flag to document, test, and support.
- The provided codemod eliminates `features.byok-copilot` from existing workflows automatically.

#### Negative
- Workflows that previously omitted `features.byok-copilot` and relied on the non-BYOK Copilot path (empty `COPILOT_MODEL`, unpinned CLI behavior) will silently change behavior after the upgrade.
- Pinned `engine.version` values for Copilot are now ignored; teams that pinned a specific CLI version lose that control.

#### Neutral
- The `ByokCopilotFeatureFlag` constant is retained in the codebase for backwards-compatibility reference but carries a deprecation notice.
- The `features-byok-copilot-removal` codemod is registered in the standard codemod order and runs automatically on `gh-aw fix`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Copilot Engine BYOK Defaults

1. The compiler **MUST** inject a dummy `COPILOT_API_KEY` sentinel value for all `engine: copilot` workflows, regardless of whether `features.byok-copilot` is present.
2. The compiler **MUST** treat `cli-proxy` as enabled for all `engine: copilot` workflows unless `tools.github.mode` explicitly overrides it.
3. The compiler **MUST** install the Copilot CLI at `latest` for all `engine: copilot` workflows and **MUST NOT** honour a pinned `engine.version` for the Copilot CLI install step.
4. The compiler **MUST** set `COPILOT_MODEL` to `${{ vars.GH_AW_MODEL_AGENT_COPILOT || 'claude-sonnet-4.6' }}` when no explicit model is configured, providing a non-empty fallback required by BYOK providers.

### Feature Flag Deprecation

1. The `features.byok-copilot` flag **MUST NOT** alter compiler behavior; it **SHALL** be treated as a no-op.
2. Implementations **SHOULD** use the `features-byok-copilot-removal` codemod to strip `features.byok-copilot` from existing workflow frontmatter.
3. The `ByokCopilotFeatureFlag` constant **MAY** be retained in source code with a deprecation notice but **MUST NOT** be used to gate any compiler behavior.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24804504144) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
