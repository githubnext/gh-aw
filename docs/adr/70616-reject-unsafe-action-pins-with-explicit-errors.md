# ADR-70616: Reject Unsafe Action Pins with Explicit Errors Instead of Escaping

**Date**: 2026-10-20
**Status**: Proposed (pending maintainer approval)
**Deciders**: No human decision or additional deciders are recorded in the evidence reviewed for this ADR.
**PR author**: pelikhan

---

### Context

The compiler emits `uses:` lines into generated YAML from action pins. Those pins can come from the built-in pin cache, a SHA resolver, or user-configured action mappings and prefixes. Single-quoted references were already escaped for apostrophes (`escapeYAMLSingleQuoted`). [PR #70616](https://github.com/github/gh-aw/pull/70616) reports that code scanning alerts 1030/1031 were dismissed for that escaping and that no attacker-controlled exploit was demonstrated; [issue #69882](https://github.com/github/gh-aw/issues/69882) provides the hardening context.

Previously, `quoteYAMLActionPin` returned only a `string` and could not signal unsafe input. Newlines and other YAML line separators in a reference or trailing version comment can alter generated YAML structure. `${{` introduces a separate GitHub Actions expression concern that YAML quoting alone does not address. The original change hardened maintenance label jobs and quoted mapped pins, but unquoted mappings and maintenance jobs emitted with labels disabled need the same boundary checks.

### Decision

Validate every emitted maintenance or mapped action-pin reference and its trailing comment before YAML emission, regardless of whether the reference is quoted or whether label-triggered maintenance jobs are enabled. Reject unsafe values with an explicit error instead of sanitizing them. This invariant covers default maintenance jobs as well as label jobs and both quoted and unquoted mapped pins; validation must not depend on visiting a label builder or a quoting branch.

The shared `validateYAMLActionPin` policy rejects Unicode control characters except tab (including newline, carriage return, NUL, and U+0085), U+2028/U+2029, and any `${{` sequence. `quoteYAMLActionPin` returns `(string, error)` and retains apostrophe escaping for safe single-quoted references. Errors must propagate through mapping and maintenance generation to compilation, without returning successful YAML containing an unsafe pin. Rejecting rather than rewriting preserves action identity and exposes malformed resolver or configuration values.

### Alternatives Considered

#### Alternative 1: Escape or strip unsafe characters and keep the `string` return type

Stripping or escaping forbidden characters could avoid error-return plumbing, but silently rewriting a pin may change action identity or conceal a bad resolver result or mapping. Apostrophe escaping is different: it preserves the scalar's value after YAML parsing. Rejection is preferable for values outside the emission policy.

#### Alternative 2: Rely on the existing dismissals and emit double-quoted YAML scalars

Existing dismissals do not establish that every emission path handles unsafe comments or expressions. Double-quoted scalars with full YAML escaping could preserve reference characters without changing YAML structure, but would not protect the trailing comment or block GitHub Actions expression processing. Changing quoting everywhere would also cause avoidable lock-file output churn.

#### Alternative 3: Validate pins once at their source (resolver/mapping configuration)

Source validation can improve diagnostics, but does not replace validation after references and comments have been composed from prefixes, mappings, cache entries, and resolver results. Emission-boundary validation protects the final value independently of its origin.

These alternatives document the technical tradeoffs evaluated in this ADR, not an independently verified history of human deliberation or approval.

### Consequences

#### Positive
- The forbidden characters and expression markers are rejected in emitted maintenance and mapped references and comments, independently of quoting and label configuration. This is an emission-safety policy, not a complete validator for all GitHub Actions reference syntax.
- Failures are explicit and include context, such as `label_apply_safe_outputs: invalid action pin: ...` and `invalid action pin comment for repo@version`, so misconfiguration is easy to diagnose.
- Existing safe quoting and comment formatting are preserved. `TestQuoteYAMLActionPin` verifies YAML round trips for plain pins, version comments, apostrophes, and YAML punctuation; `TestMapPinnedUsesQuotedScalars` checks exact mappings, prefixes, source-version comments, and tab-separated comments.
- The PR's recorded validation reports 345 workflows compiled with lock files in sync. This is compatibility evidence for the checked-in workflows at that validation point, not proof that every possible external pin is accepted or that subsequent revisions have been validated.

#### Negative
- Checked maintenance and mapping paths require error propagation and corresponding caller/test updates.
- Previously accepted values containing a forbidden character or expression marker now fail compilation, even if a caller intended them literally. There is no claim that all external configurations have been inventoried. Ordinary SHA pins and version-tag references remain supported; compatibility is demonstrated by the focused fixtures and recorded workflow compilation, rather than an assertion that unusual legitimate pins cannot exist.

#### Neutral
- The validation rule (control characters except tab, Unicode line separators, `${{`) is a shared policy that future maintenance or mapped action-pin emitters must reuse.
- The PR also adds test-only hardening (pip option-like package names, Pi subagent stdin-only task passing) that does not change runtime behavior.

---

### Evidence

- [Implementation commit 0fa58157fb](https://github.com/github/gh-aw/commit/0fa58157fbafc0fea4671632023c044941448ebf) records fail-closed pin hardening; draft commit `92783475ae` records generation of this ADR, not human acceptance.
- [Maintenance pin tests](../../pkg/workflow/maintenance_workflow_labels_test.go) cover safe scalar round trips, forbidden characters and expressions, and label-builder error propagation.
- [Mapping tests](../../pkg/workflow/action_pins_test.go) cover preserved mappings/comments and rejection of unsafe quoted prefixes and resolver comments.
- [PR #70616](https://github.com/github/gh-aw/pull/70616) records the compatibility validation summarized above. The complete batch must also validate the unquoted-mapping and labels-disabled maintenance paths against the invariant in this decision.
