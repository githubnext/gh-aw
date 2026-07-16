# ADR-45858: Consolidate Duplicated Parsing Helpers into Canonical Shared Functions

**Date**: 2026-07-16
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `pkg/workflow` and `pkg/cli` packages contained multiple near-identical inline implementations of the same low-level parsing operations: positive-integer parsing from frontmatter values, integer-or-expression parsing for GitHub Actions template strings (`${{ ... }}`), workflow-ID normalization from file paths, and timeline row construction. Because each call site copied the logic independently, a bug fix or behavioral change (e.g., accepting an additional expression format) required updates in N places. The duplication also introduced subtle inconsistencies: some sites logged field-specific error messages, others logged generic ones, and some omitted logging entirely.

### Decision

We will extract each repeated parsing operation into a single, parameterized, unexported helper within the package that owns the concept:
- `parsePositiveIntValue(raw any, fieldName string) int` and `parseIntOrExpressionValue(raw any, minValue int, fieldName string) string` in `pkg/workflow`
- `isExpression(s string) bool` as a shared predicate replacing all `strings.HasPrefix(…, "${{") && strings.HasSuffix(…, "}}")` checks in `pkg/workflow`
- `normalizeWorkflowID(filename string) string` as the single call site for `.md` basename stripping in `pkg/cli`
- `renderMessageContentTimelineRow(kind TimelineEventKind, evt UnifiedTimelineEvent) []string` replacing duplicated timeline row construction in `pkg/cli`

All existing public-facing functions become thin wrappers that delegate to the shared helper.

### Alternatives Considered

#### Alternative 1: Maintain Inline Copies at Each Call Site (Status Quo)

Keep the parsing logic repeated at each of the five or more call sites. This requires no refactoring and avoids any indirection. We rejected it because a logic change (e.g., updating expression detection) requires touching every site individually, and omissions create behavioral inconsistencies that are hard to catch in code review.

#### Alternative 2: Introduce a Dedicated Utility Package (e.g., `pkg/parseutil`)

Move the helpers to a new package so they are importable across `pkg/workflow`, `pkg/cli`, and any future packages. This would allow cross-package reuse without creating import cycles. We rejected it for this change because all current call sites are within their own package, the helpers are implementation details that should not be part of the public API surface, and the extra package indirection adds maintenance overhead that is not yet justified.

### Consequences

#### Positive
- Parsing behavior is now consistent at all call sites; a fix to `isExpression` or `parseIntOrExpressionValue` propagates automatically to every consumer.
- Error log messages are uniformly parameterized with the field name, making diagnostics more actionable.
- New tests for the shared helpers provide regression coverage that did not previously exist for many of the duplicated call sites.

#### Negative
- Stack traces for parsing failures are now one frame deeper; developers reading a crash log must look through the wrapper to find the shared helper.
- The helpers are unexported, so they cannot be reused across package boundaries without further refactoring (e.g., if `pkg/cli` ever needs `isExpression`, it must either re-implement it or create an import dependency).

#### Neutral
- Existing public-facing parser functions (`parseMaxRunsValue`, `parseMaxTurnsValue`, etc.) retain their signatures — callers are unaffected.
- The `normalizeWorkflowID` and `renderMessageContentTimelineRow` helpers follow the same package-local pattern; their behavior is behavior-preserving and the change is transparent to external packages.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
