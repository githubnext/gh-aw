# ADR-0002: Allow Secrets in Step-Level env: Bindings Under Strict Mode

**Date**: 2026-04-11
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The workflow compiler enforces a "strict mode" that restricts what GitHub Actions expressions authors may use in user-defined step fields, with the goal of preventing secrets from leaking into the agent job environment. Before this change, strict mode applied an all-or-nothing rule: the presence of *any* `${{ secrets.* }}` expression in `pre-steps`, `steps`, or `post-steps` was an error, regardless of where inside the step the expression appeared. This forced authors who needed tool credentials (API tokens, OAuth keys, SonarQube tokens, etc.) to opt out of strict mode entirely via `strict: false`, surrendering all other protections the mode provides. The GitHub Actions platform already distinguishes between controlled secret surfaces: a secret bound to a step's `env:` map is automatically masked by the runner before it can appear in logs, whereas a secret interpolated directly into a `run:` script string can be echoed, logged, or passed to an external process before masking takes effect.

### Decision

We will introduce a per-field classification of secret references within a step, distinguishing "safe" bindings (`env:` fields, which are automatically masked by GitHub Actions, and `with:` inputs for `uses:` action steps, which are passed to external actions and masked by the runner) from "unsafe" inline interpolations (`run:` and all other step fields). In strict mode, only unsafe secret references will be treated as errors; secrets that appear exclusively in safe bindings will be permitted. We implement this via a new `classifyStepSecrets()` helper that partitions a step's secret references, and update `validateStepsSectionSecrets()` to only block the unsafe partition under strict mode.

### Alternatives Considered

#### Alternative 1: Keep the Existing All-or-Nothing Block

Maintaining the current policy — all `secrets.*` references are errors in strict mode — is the simplest approach. It was rejected because it creates an unacceptable ergonomic cost: workflows that need to supply API tokens to scanning tools (a common real-world pattern) must disable strict mode entirely, removing protection against other classes of secret leak that strict mode prevents. The framework's own generated jobs already use `env:` bindings for secrets, making it inconsistent to block that same pattern in user-defined steps.

#### Alternative 2: Allow All Secrets in Strict Mode

Relaxing strict mode to permit secrets everywhere (matching non-strict mode, but without the warning) would maximally ease author burden. It was rejected because it removes the core protection that strict mode is designed to provide: preventing accidental inline interpolation of secrets into command strings where they can be observed before the runner's masking logic fires.

#### Alternative 3: Introduce a Per-Step Annotation to Opt In

A third option was to keep secrets blocked by default but allow authors to annotate individual steps (e.g., with a `allow-secrets: true` flag) to opt into secret access. This was rejected as unnecessarily complex: the `env:` binding pattern is already a well-established GitHub Actions idiom, so using the structural location of the reference (field name `env` vs. any other field) as the signal provides an equivalent security boundary without requiring new syntax.

### Consequences

#### Positive
- Workflows that supply tool credentials via `env:` bindings or `with:` inputs (for `uses:` action steps) no longer need to disable strict mode entirely, preserving all other strict-mode protections.
- The enforcement policy now mirrors how the framework's own generated jobs handle secrets, making the security model internally consistent.
- The error message for blocked secrets now explicitly suggests `env:` bindings and `with:` inputs as alternatives, improving the developer experience.
- The behavior aligns with GitHub Actions' native masking guarantee: `env:` bindings and `with:` inputs are masked by the runner before command execution.
- Enterprise workflows that use centralized secret managers (e.g., Conjur, HashiCorp Vault) via dedicated GitHub Actions can now use strict mode, passing authentication credentials via `with:` inputs.

#### Negative
- The security policy is now more nuanced: reviewers must understand the `env:`/`with:` vs. inline distinction rather than a simple blanket rule, increasing the surface area to reason about.
- The `classifyStepSecrets()` function must be kept accurate as the step data model evolves; an incorrectly classified field could silently downgrade a secret from "unsafe" to "safe".
- Non-strict mode still emits a warning for all secrets (including safe-bound ones), which may be slightly misleading now that safe bindings are permitted in strict mode.

#### Neutral
- Existing workflows that used `strict: false` specifically to allow safe bindings can now remove that override and adopt strict mode, but this migration is voluntary.
- Unit and integration tests must now cover both the "safe-binding-only" allowed path and the "mixed safe + run" blocked path to maintain adequate coverage.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Secret Classification

1. Implementations **MUST** classify each `${{ secrets.* }}` reference within a step by the name of the YAML field in which it appears.
2. A secret reference found inside a well-formed `env:` mapping (i.e., the `env:` value is a YAML map of key-value pairs) **MUST** be classified as a *safe* reference.
3. A secret reference found inside a malformed `env:` value (e.g., `env:` is a bare string or a YAML sequence) **MUST** be classified as an *unsafe* reference, because the runner cannot apply per-variable masking to such values.
4. A secret reference found inside a well-formed `with:` mapping in a step that also has a `uses:` field **MUST** be classified as a *safe* reference, because `with:` inputs are passed to the external action (not interpolated into shell scripts) and the runner masks values derived from secrets.
5. A secret reference found inside a `with:` mapping in a step that does NOT have a `uses:` field **MUST** be classified as an *unsafe* reference.
6. A secret reference found inside a malformed `with:` value (e.g., `with:` is a bare string or a YAML sequence) **MUST** be classified as an *unsafe* reference.
7. A secret reference found in any other step field (including but not limited to `run:`, `name:`, `if:`) **MUST** be classified as an *unsafe* reference.
8. A step value that is not a YAML map (e.g., a raw string) **MUST** treat all secret references within it as *unsafe* references.
9. When a step contains *safe* secret references AND any non-`env:`/non-`with:` field references `GITHUB_ENV`, implementations **MUST** reclassify all *safe* references in that step as *unsafe*, because writing to `GITHUB_ENV` persists secrets to subsequent steps (including the agent step).

### Strict Mode Enforcement

1. When strict mode is active, implementations **MUST** return an error if one or more *unsafe* secret references are found in a `pre-steps`, `steps`, or `post-steps` section.
2. When strict mode is active, implementations **MUST NOT** return an error solely because *safe* secret references are present in a section.
3. The error message for blocked secrets in strict mode **SHOULD** suggest the use of step-level `env:` bindings (for `run:` steps) or `with:` inputs (for `uses:` action steps) as alternatives to inline interpolation.
4. The built-in `GITHUB_TOKEN` **MUST** be filtered out from both *unsafe* and *safe* reference lists before strict-mode enforcement, as it is present in every runner environment by default.

### Non-Strict Mode Behavior

1. When strict mode is not active, implementations **MUST** emit a warning (to stderr) if any secret references — whether *env-bound* or *unsafe* — are found in a steps section.
2. Implementations **MUST NOT** return an error in non-strict mode for secret references in steps sections; the warning is advisory only.
3. Implementations **SHOULD** deduplicate secret reference identifiers before including them in warning or error messages.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. In particular: permitting *unsafe* secret references in strict mode, or blocking *safe* references in strict mode, are both non-conformant behaviors.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24279167784) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
