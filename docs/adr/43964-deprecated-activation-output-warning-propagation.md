# ADR-43964: Track and Propagate Deprecated Activation-Output Warning Counts to Compile Summary

**Date**: 2026-07-07
**Status**: Draft
**Deciders**: Unknown

---

### Context

The workflow compiler silently rewrites deprecated `needs.activation.outputs.{text,title,body}` expressions to their `steps.sanitized.outputs.*` equivalents at compile time. A per-expression warning was already emitted to stderr, but two problems persisted: (1) the warning message gave no migration path, leaving authors unsure how to fix it; (2) the count of deprecation warnings was never fed into the compiler's central warning counter, so the compile-summary statistics omitted them entirely. As a result, authors had no incentive to migrate—the build succeeded silently and the summary showed no warnings.

### Decision

We will add a `deprecationWarningCount` field to `ExpressionExtractor` (exposed via `GetDeprecationWarningCount()`) and propagate that count to `Compiler.IncrementWarningCount()` at every call site where expression extraction occurs, including `extractPromptChunksFromMarkdown` and the inline/main extractor paths. The deprecation warning message is also strengthened to include the exact `gh aw fix` migration hint so authors know how to act on it.

### Alternatives Considered

#### Alternative 1: Fail the Build on Deprecated Expressions

Treat use of `needs.activation.outputs.{text,title,body}` as a compile error rather than a warning. This would force authors to migrate immediately and remove the deprecated path sooner.

Not chosen because it is a breaking change: existing workflows using the deprecated form would suddenly fail to compile without any migration window. The preferred path is a visible warning plus an automated fix tool (`gh aw fix`), giving authors time to migrate.

#### Alternative 2: Centralize Deprecation Detection in the Compiler (Not in ExpressionExtractor)

Move the deprecation-detection and counting logic into `Compiler.generatePrompt` or a dedicated post-processing pass, rather than adding state to `ExpressionExtractor`.

Not chosen because `ExpressionExtractor` is already the single place where expression substitution and rewriting happens. Duplicating or splitting detection elsewhere would require either re-scanning the markdown or coupling the compiler more tightly to substitution internals, increasing maintenance burden.

#### Alternative 3: Use a Global/Package-Level Counter Instead of Instance State

Track deprecation warnings in a global counter shared across all extractor instances.

Not chosen because global mutable state complicates concurrent use and testing. Per-instance counting is composable: callers aggregate counts as needed, which is exactly what `Compiler.IncrementWarningCount()` does.

### Consequences

#### Positive
- Authors see a clear, actionable warning message including `gh aw fix` every time a deprecated expression is encountered.
- Compile-summary statistics now reflect deprecation warnings, making the issue visible at a glance.
- The implementation is localised to `ExpressionExtractor` and its existing call sites—no new abstractions or packages required.
- Test coverage added for single/multiple/duplicate deprecated expressions and for the absence of warnings on non-deprecated expressions.

#### Negative
- `ExpressionExtractor` carries additional state (`deprecationWarningCount`), slightly increasing its complexity; all callers that care about the count must explicitly propagate it.
- Deduplication of the count is per-extractor-instance, not global across a full compilation: if the same deprecated expression appears in two separate markdown bodies processed by different extractor instances, two warnings are emitted and two counts are propagated. This matches the desired "warn once per unique expression per extraction context" semantics but differs from "warn once per unique expression per full compile run."

#### Neutral
- The `extractPromptChunksFromMarkdown` helper's return signature changes from `([]string, []*ExpressionMapping)` to `([]string, []*ExpressionMapping, int)`; all call sites are updated in this PR.
- No changes to public API surfaces, output schema, or workflow file format.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
