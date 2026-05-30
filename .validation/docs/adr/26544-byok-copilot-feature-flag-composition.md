# ADR-26544: BYOK-Copilot Feature Flag as a Composing Abstraction for Offline Mode Wiring

**Date**: 2026-04-16
**Status**: Draft
**Deciders**: lpcox, Copilot (PR author)

---

## Part 1 — Narrative (Human-Friendly)

### Context

Copilot offline BYOK (Bring Your Own Key) mode requires three separate pieces to work correctly together: injecting a well-known dummy API key (`COPILOT_API_KEY`) into the execution environment to trigger AWF's runtime BYOK-detection path, enabling the `cli-proxy` feature so that `gh` CLI calls are routed through the authenticated DIFC proxy sidecar, and forcing the Copilot CLI installer to use `latest` instead of a pinned version. Previously, workflow authors had to compose all three behaviors manually in their workflow frontmatter, which was error-prone and created a high documentation burden. The system needed a first-class abstraction that bundled the required wiring behind a single, discoverable switch.

### Decision

We will introduce a `byok-copilot` feature flag that, when enabled on a `copilot`-engine workflow, automatically composes the three required behaviors: it injects `COPILOT_API_KEY: dummy-byok-key-for-offline-mode` into the execution environment, implicitly enables the `cli-proxy` feature (for Copilot engine only), and overrides the engine version to `latest` during installation. The implicit `cli-proxy` enablement is implemented inside `isFeatureEnabled` as a special case: when the `cli-proxy` flag is queried, the function first checks whether `byok-copilot` is active on a Copilot engine workflow and returns `true` if so. The dummy API key is a well-known sentinel value; the real credential never leaves the AWF API proxy sidecar.

### Alternatives Considered

#### Alternative 1: Require explicit frontmatter composition

Workflow authors could be required to set `cli-proxy`, `COPILOT_API_KEY`, and the version field individually. This avoids any hidden behavior in the feature flag system, and every enabled behavior is visible at the frontmatter layer. It was rejected because the combination is mandatory for BYOK to work—there is no valid use case for enabling one without the others—and requiring authors to assemble all three manually introduces a category of silent misconfiguration bugs that are hard to diagnose at runtime.

#### Alternative 2: A "preset" system separate from feature flags

A named preset (e.g., `preset: byok-copilot`) could expand to a validated set of options at compile time, distinct from the existing feature flag mechanism. This would avoid adding implicit logic to `isFeatureEnabled` and could be extended to other preset bundles in the future. It was not chosen because it would require introducing a new schema concept (`preset`) alongside the existing `features` map, increasing the surface area of the workflow YAML language without clear benefit over the simpler feature flag approach for a single-flag use case.

#### Alternative 3: Inject all BYOK wiring unconditionally for all Copilot workflows

The AWF runtime could detect BYOK mode automatically (e.g., by the absence of a real API key) and apply all necessary wiring without any explicit flag. This removes the flag entirely and avoids any frontmatter requirement. It was rejected because it would silently alter the execution environment for all Copilot workflows, making it harder to understand why certain environment variables or proxy flags appear, and it couples the BYOK decision to runtime inference rather than explicit author intent.

### Consequences

#### Positive
- Workflow authors enable BYOK mode with a single line (`byok-copilot: true`) instead of assembling three separate, order-sensitive options.
- Misconfiguration is structurally eliminated: all three required behaviors are either all present or all absent.
- The dummy key constant (`CopilotBYOKDummyAPIKey`) is centralised in `pkg/constants`, making the sentinel value discoverable and easy to update if the AWF BYOK detection signal changes.

#### Negative
- `isFeatureEnabled` now contains an implicit cross-flag dependency (`cli-proxy` ← `byok-copilot`). Readers of the feature evaluation code must know this special case exists; it is not visible from the frontmatter alone.
- The implicit `cli-proxy` enablement is engine-scoped (Copilot only), adding a conditional that future maintainers must understand when adding new engines or new flag interactions.
- Forcing `latest` on BYOK install overrides any pinned version the author may have set, which may cause unexpected behavior if a regression is introduced in the latest Copilot CLI release.

#### Neutral
- The glossary entry for `features` has been updated to document `byok-copilot` semantics.
- The dummy key value (`dummy-byok-key-for-offline-mode`) is opaque to the AWF container; the real credential path is entirely in the AWF API proxy sidecar and unchanged by this PR.
- Tests for the new flag are co-located with the existing engine tests, following the project's existing test organization pattern.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Feature Flag Behavior

1. Implementations **MUST** treat `byok-copilot` as a valid `FeatureFlag` constant defined in `pkg/constants/feature_constants.go`.
2. When `byok-copilot` is enabled on a workflow with `engine: copilot`, implementations **MUST** inject the value of `CopilotBYOKDummyAPIKey` as the `COPILOT_API_KEY` environment variable into the Copilot execution step.
3. When `byok-copilot` is enabled on a workflow with `engine: copilot`, implementations **MUST** treat the `cli-proxy` feature flag as enabled, regardless of whether it is explicitly set in the workflow frontmatter.
4. The implicit `cli-proxy` enablement described above **MUST NOT** apply to workflows using engines other than `copilot`.
5. When `byok-copilot` is enabled, implementations **MUST** override the Copilot CLI install version to `latest`, ignoring any `engine.version` value set in the workflow frontmatter.

### Dummy Key Constant

1. The placeholder API key used for BYOK detection **MUST** be sourced from the `CopilotBYOKDummyAPIKey` constant in `pkg/constants/engine_constants.go`.
2. Implementations **MUST NOT** embed the dummy key value as a string literal outside of the constants package.
3. The dummy key **MUST NOT** be treated as a real credential; the real AWF API proxy credential **SHALL** remain isolated in the sidecar and **MUST NOT** be injected into the workflow container environment.

### Feature Composition Logic

1. Implicit feature enablement rules (such as `byok-copilot` enabling `cli-proxy`) **MUST** be implemented inside the `isFeatureEnabled` function in `pkg/workflow/features.go`.
2. Each implicit enablement rule **MUST** be scoped to the specific engine or context for which it applies and **MUST NOT** affect unrelated engines or feature flags.
3. Implementations **SHOULD** log a message when an implicit feature enablement rule is triggered, to aid in runtime diagnostics.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24490726928) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
