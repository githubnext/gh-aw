# ADR-47264: Extend detectTextOutputUsage to Recognise Deprecated needs.activation.outputs Forms

**Date**: 2026-07-22
**Status**: Draft
**Deciders**: Unknown (Copilot SWE Agent, pelikhan)

---

### Context

The workflow compiler automatically rewrites deprecated `${{ needs.activation.outputs.{text,title,body} }}` expressions to their modern `${{ steps.sanitized.outputs.{text,title,body} }}` equivalents. However, the `detectTextOutputUsage` function — which decides whether the sanitized step must be injected into the compiled workflow — only scanned for the modern form.

Workflows that had not yet been migrated to the modern syntax would silently compile without the sanitized step. This caused runtime failures particularly for `workflow_dispatch`-only triggers, where no content context is provided and the missing step caused undefined outputs.

The fix must not break any already-migrated workflows, and must preserve the existing deprecation-warning pathway so users are still notified to migrate.

### Decision

We will extend `detectTextOutputUsage` to also check for `${{ needs.activation.outputs.text }}`, `${{ needs.activation.outputs.title }}`, and `${{ needs.activation.outputs.body }}` alongside the already-supported modern forms, using short-circuit `if !hasXUsage` guards so that a single `strings.Contains` hit is enough to set the flag. The auto-rewrite that transforms deprecated to modern syntax at the expression level is left untouched.

### Alternatives Considered

#### Alternative 1: Reject deprecated syntax with a compile-time error

Return a compile error (or emit a fatal diagnostic) whenever a deprecated `needs.activation.outputs.*` expression is found, requiring authors to migrate before the workflow compiles.

Rejected because it is a breaking change for all existing unmigrated workflows. The project's stated intent is a gradual migration; forcing an immediate hard break contradicts that policy and would block legitimate workflows from compiling at all.

#### Alternative 2: Emit a lint warning and rely on authors to migrate before runtime

Log a prominent warning, leave `detectTextOutputUsage` as-is (only scanning modern forms), and document that deprecated syntax may produce incorrect compiled output until migrated.

Rejected because it preserves the silent-failure bug that caused the issue in the first place. Users would see a warning but would still receive a broken compiled workflow, making this behaviour invisible until a runtime failure occurred.

### Consequences

#### Positive
- Unmigrated workflows now compile correctly and include the sanitized step, eliminating the silent runtime failure.
- The fix is non-breaking: already-migrated workflows are unaffected because the `if !hasXUsage` guards skip the deprecated-form check once the modern form is found.
- Deprecation warnings (emitted to stderr by `ExpressionExtractor`) are preserved, continuing to guide authors toward the modern syntax.

#### Negative
- The deprecated `needs.activation.outputs.*` strings are now referenced in two places in the compiler: the auto-rewrite in `ExpressionExtractor` and the new detection guards in `detectTextOutputUsage`. This increases the surface area that must be updated when deprecated syntax support is eventually removed.
- Retaining dual-path detection extends the effective deprecation window, since unmigrated workflows no longer fail visibly and authors have less urgency to migrate.

#### Neutral
- Three new test cases were added to `TestDetectTextOutputUsage` and a new `TestExpressionExtractor_DeprecatedActivationOutputWarning` suite was introduced, increasing test coverage of the backwards-compatibility layer.
- Existing tests that used the deprecated form in `generateEnvVarName` and `NoCollisions` scenarios were updated to the modern form, reflecting that deprecated syntax is transformed before it reaches those code paths in production.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
