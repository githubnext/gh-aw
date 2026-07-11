# ADR-44922: Block toJSON(secrets) Serialization in Compiler Expression Validation

**Date**: 2026-07-11
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

GitHub Actions expressions like `${{ toJSON(secrets) }}` convert the entire `secrets` context to a JSON string. When used inside gh-aw workflow markdown or frontmatter YAML, this passes **all** repository secrets to the agent rather than only the specific values it requires. The existing expression safety validator (`validateExpressionSafety`) blocks unlisted expressions but does not emit a specific, actionable error message that explains *why* `toJSON(secrets)` is dangerous. Without a dedicated rule, contributors can unknowingly include this pattern and receive only a generic "expression not allowed" error that does not guide them toward a safe alternative.

### Decision

We will add a dedicated compiler validation rule — `validateSecretsSerializationExpressions` — that detects `toJSON(secrets)` (and case/whitespace variants) in both markdown content and frontmatter YAML before the general allowlist check runs. In strict mode (the default), the rule returns a compilation error with a message that names the offending expression and directs the author to use specific secret references (`secrets.MY_SECRET`) instead. In non-strict mode (`strict: false` in frontmatter), the rule emits a warning and allows compilation to proceed; matched expressions are then neutralized (`${{ false }}`) before the allowlist check to prevent a redundant secondary error.

### Alternatives Considered

#### Alternative 1: Rely solely on the existing allowlist validator

The existing `validateExpressionSafety` already blocks `${{ toJSON(secrets) }}` as an unlisted expression. No new rule would be needed. This was rejected because the error message produced is generic ("expression not in allowed list") and does not tell the author *why* the pattern is dangerous or what safe alternative to use. A dedicated rule provides a specific, educational error that improves the contributor experience and reduces support burden.

#### Alternative 2: Documentation guidance without compiler enforcement

Add a "security best practices" note in project documentation advising against `toJSON(secrets)`. This was rejected because documentation is easy to miss and provides no enforcement. A compiler rule catches the pattern at authoring time regardless of whether the contributor has read the docs.

### Consequences

#### Positive
- Authors receive a clear, specific error message naming the offending expression and explaining the security risk, reducing the likelihood of accidental credential exposure.
- Strict/non-strict behavior follows the same pattern used by other secret-related validators in the codebase (`strict_mode_validation.go`, `strict_mode_env_validation.go`), keeping the validation layer consistent and predictable.
- The rule runs before the general allowlist check, so the actionable message is surfaced first and the secondary "expression not allowed" error is suppressed.

#### Negative
- The neutralization step (`neutralizeSecretsSerializationExpressions`) adds a code path that rewrites markdown content in memory before the allowlist check; this must be kept in sync with the detection pattern and can be a source of subtle bugs if the regex diverges.
- If GitHub Actions adds new functions that serialize secrets in the future (beyond `toJSON`), this rule will not catch them and will require a separate update.

#### Neutral
- The new file follows existing file-naming conventions (`expression_*_validation.go`) and is cross-referenced from the package-level comments in related files, easing discoverability for future contributors.
- 23 unit tests cover strict/non-strict modes, markdown body vs. frontmatter YAML locations, case variants, and safe patterns; the test file is self-contained and does not require integration infrastructure.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
