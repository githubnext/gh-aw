# ADR-27639: Unify Allowed-Domains Configuration for Activation Input Sanitization

**Date**: 2026-04-21
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The workflow compiler generates a GitHub Actions job that processes incoming text (issues, PR bodies, comments) before passing it to the agent. This job includes a `sanitized` step (`id: sanitized`) that redacts URLs not in an allowed-domains list, guarding against prompt-injection via untrusted URLs in user-submitted content. However, this step was hard-wired to use only the default domain allow-list, ignoring any domains the workflow author had configured via `network.allowed` and `safe-outputs.allowed-domains`. Output sanitization already consumed those user-configured domains correctly. The asymmetry meant that URLs explicitly permitted by the workflow author were being silently redacted from the agent's input, causing unexpected agent behavior.

### Decision

We will pass the same computed allowed-domains value to the `sanitized` (input) step that is already used for output sanitization. When `safe-outputs.allowed-domains` is configured, we use `computeExpandedAllowedDomainsForSanitization`; otherwise we use `computeAllowedDomainsForSanitization`. This brings input-side sanitization into parity with output-side sanitization, honoring the workflow author's intent consistently across both directions.

### Alternatives Considered

#### Alternative 1: Remove Input Sanitization from the Activation Step

Drop URL sanitization from the `sanitized` step entirely and rely solely on output sanitization. This eliminates the inconsistency by having only one sanitization pass. It was rejected because input sanitization provides a meaningful defense-in-depth layer: by stripping untrusted URLs before the agent processes the text, it reduces the attack surface for prompt-injection even if the agent generates output that later passes through output sanitization.

#### Alternative 2: Introduce a Separate `input-sanitization.allowed-domains` Config Key

Add a new, explicitly scoped configuration field for input-side sanitization rather than reusing the existing output-side allow-list. This was rejected because it would require workflow authors to duplicate their domain allowances in two places to achieve consistent behavior, increasing cognitive overhead and the risk of accidental divergence with no clear benefit.

### Consequences

#### Positive
- Workflow authors' `network.allowed` and `safe-outputs.allowed-domains` settings now apply symmetrically to both input and output, eliminating silent URL redaction from the agent's incoming context.
- No new configuration surface is required; the fix reuses existing compiler helpers already tested for output sanitization.

#### Negative
- Domains listed in `safe-outputs.allowed-domains` now implicitly affect input sanitization, even though the field's name suggests an output-only concern. This may surprise workflow authors who intended `safe-outputs.allowed-domains` to scope only what the agent is allowed to reference in its outputs.
- The env-var assembly block in `addActivationRepositoryAndOutputSteps` is slightly more complex, accumulating lines conditionally rather than emitting them inline.

#### Neutral
- The `GH_AW_ALLOWED_BOTS` env var emission is unchanged in behavior; it was refactored into the same slice-accumulation pattern as part of this change for consistency.
- New regression test `TestComputeTextStepIncludesAllowedDomainsEnv` verifies that both `GH_AW_ALLOWED_BOTS` and `GH_AW_ALLOWED_DOMAINS` appear in the compiled `sanitized` step with the expected domain values.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Activation Input Sanitization — Allowed-Domains Wiring

1. The compiled `sanitized` step **MUST** receive the same allowed-domains value that is used for output sanitization in the same compiled workflow.
2. When `safe-outputs.allowed-domains` is non-empty, the compiler **MUST** compute the allowed-domains value using `computeExpandedAllowedDomainsForSanitization`.
3. When `safe-outputs.allowed-domains` is absent or empty, the compiler **MUST** compute the allowed-domains value using `computeAllowedDomainsForSanitization`.
4. If the computed allowed-domains string is non-empty, the compiler **MUST** emit a `GH_AW_ALLOWED_DOMAINS` environment variable in the `sanitized` step's `env` block.
5. The compiler **MUST NOT** emit `GH_AW_ALLOWED_DOMAINS` when the computed allowed-domains string is empty.
6. The compiler **MUST NOT** use a hardcoded or default-only domain list for the `sanitized` step when user-configured domains are present.

### Activation Input Sanitization — Bot Allowlist

1. If the workflow's `bots` list is non-empty, the compiler **MUST** emit `GH_AW_ALLOWED_BOTS` in the `sanitized` step's `env` block as a comma-separated string.
2. The `GH_AW_ALLOWED_BOTS` env var **MUST NOT** be emitted when the `bots` list is empty.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24739465880) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
