# ADR-29433: Security Validation for pull_request_target Trigger

**Date**: 2026-05-01
**Status**: Draft
**Deciders**: Unknown [TODO: verify]

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `pull_request_target` GitHub Actions trigger runs workflows in the context of the base (target) branch with full write permissions and access to all repository secrets. Unlike the `pull_request` trigger, it can access secrets even when the PR originates from an untrusted fork. When combined with a checkout of PR code, this creates a well-known critical vulnerability known as a "pwn request" — a malicious fork PR can inject code that executes with elevated privileges and exfiltrates repository secrets. The workflow compiler (`pkg/workflow`) lacked any enforcement mechanism to detect or prevent this configuration, leaving authors unaware of the risk at compile time.

### Decision

We will add a dedicated `validatePullRequestTargetTrigger` validation step to the workflow compiler's `validatePermissions` pipeline. In non-strict mode the validator emits a warning when `pull_request_target` is used without `checkout: false`; in strict mode it promotes that warning to a hard compile error. In strict mode, a warning is always emitted regardless of checkout state because the trigger inherently runs with elevated privileges. This makes the security risk visible at the earliest possible point — compile time — and provides actionable remediation guidance in the error message.

### Alternatives Considered

#### Alternative 1: Documentation-Only Guidance

Document the danger of `pull_request_target` in the workflow authoring guide and rely on authors to follow the guidance voluntarily. This was not chosen because it is purely passive: existing and new workflows can violate the security rule without any tooling signal. The GitHub Actions security community has repeatedly identified "pwn requests" as a widespread real-world incident class, suggesting passive documentation is insufficient.

#### Alternative 2: Always Hard-Error on pull_request_target (Regardless of Checkout State)

Block `pull_request_target` entirely unless it appears on an explicit allowlist. This would be maximally safe but would break legitimate uses of `pull_request_target` with `checkout: false`, which is a valid pattern for workflows that need write-back access to comment on PRs without executing fork code. The chosen tiered approach (warning vs. error based on checkout state and strict mode) preserves backward compatibility while still enforcing the security boundary.

### Consequences

#### Positive
- Pwn-request vulnerabilities are surfaced at compile time with a specific, actionable error message and a link to the GitHub Security Lab advisory.
- Strict-mode enforcement creates a hard gate for teams that require security compliance, preventing the misconfiguration from ever reaching production.

#### Negative
- Existing workflows that use `pull_request_target` without `checkout: false` will begin receiving warnings (non-strict) or compile errors (strict), requiring authors to audit and update their workflows.
- The validation adds a YAML parse of the `On` field for any workflow containing the string `pull_request_target`, introducing a small per-compile cost (mitigated by an upfront string fast-path check).

#### Neutral
- The validator is inserted as step 6 of the existing `validatePermissions` pipeline, consistent with the established pattern for other trigger-scoped validators (e.g., `validateWorkflowRunBranches`).
- Both unit tests and integration tests with shared-workflow import fixtures are included, following the project's testing conventions.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Trigger Detection

1. Implementations **MUST** use the literal string `"pull_request_target"` as a fast-path pre-check before parsing the `on:` YAML field, to avoid unnecessary YAML parsing for workflows that do not use this trigger.
2. Implementations **MUST** confirm the presence of `pull_request_target` as a key in the parsed `on:` map before applying any diagnostic; a false-positive match from a string substring (e.g., `pull_request_target_staging`) **MUST NOT** trigger validation.

### Diagnostic Rules

1. In strict mode, implementations **MUST** always emit a compiler warning indicating that `pull_request_target` is a very dangerous trigger, regardless of whether `checkout: false` is set.
2. When `checkout` is not explicitly disabled (`checkout: false` absent) and the compiler is in strict mode, implementations **MUST** return a hard compile error with a message containing the phrase "extremely insecure" and a reference to the pwn-request attack vector.
3. When `checkout` is not explicitly disabled and the compiler is in non-strict mode, implementations **MUST** emit a compiler warning with the same message content and increment the warning counter.
4. When `checkout: false` is set, implementations **MUST NOT** emit the insecure-checkout error or warning; only the strict-mode dangerous-trigger warning (rule 1) **MAY** apply.

### Error Message Content

1. All diagnostic messages for this validator **MUST** include a reference URL to the GitHub Security Lab "Preventing pwn requests" advisory.
2. All diagnostic messages **SHOULD** include a suggested remediation step (e.g., "Add `checkout: false` to your workflow frontmatter").

### Integration

1. The `pull_request_target` validation step **MUST** be invoked within the `validatePermissions` pipeline after `validateWorkflowRunBranches` and before GitHub MCP toolset permission alignment.
2. Implementations **MUST NOT** return a non-nil error from this validator for any trigger other than `pull_request_target`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25201985048) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
