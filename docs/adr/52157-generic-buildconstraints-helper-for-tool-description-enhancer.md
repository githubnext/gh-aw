# ADR-52157: Generic `buildConstraints` Helper for Tool Description Enhancer

**Date**: 2026-08-11
**Status**: Draft
**Deciders**: pelikhan (copilot-swe-agent)

---

### Context

`pkg/workflow/tool_description_enhancer.go` contained approximately 33 per-tool constraint builder functions (e.g., `createIssueConstraints`, `closeDiscussionConstraints`, `updatePullRequestConstraints`). Every builder repeated the same 4-line skeleton: a nil guard (`if config == nil { return nil }`), a local slice declaration (`var constraints []string`), a series of field-conditional appends, and a return statement. The nil guard alone appeared in every function; `appendTargetConstraint`, `appendTargetRepoSlugConstraint`, and `appendRequiredTitlePrefixConstraint` patterns appeared 12, 13, and 5 times respectively. The duplication inflated the file to 740 lines and created a risk of inconsistency if the shared pattern needed to change across all builders.

### Decision

We will introduce a generic `buildConstraints[T any](config *T, build func(*T, *[]string)) []string` helper that centralizes the nil-guard and slice-setup boilerplate. We will also extract `appendTargetConstraint`, `appendTargetRepoSlugConstraint`, and `appendRequiredTitlePrefixConstraint` as standalone helpers for the three most common field-gate patterns. All existing `*Constraints` functions will be rewritten to delegate to `buildConstraints`, keeping only their tool-specific field logic. No observable behavior change is made; all message strings, field selectors, and special cases (e.g., `addCommentConstraints` always appending a trailing constraint) are preserved exactly.

### Alternatives Considered

#### Alternative 1: Keep the duplication (status quo)

Each builder function independently handles its nil check and slice initialization, as it did before. The pattern is simple to read in isolation and requires no shared abstraction. This was rejected because the repeated boilerplate created a real maintenance risk: a future change to the nil-check behavior or the slice initialization strategy would require touching 33 functions, and an inconsistent edit would not be caught by the compiler.

#### Alternative 2: Code generation

A code generator (e.g., `go generate` with a template) could produce all builder functions from a declarative spec. This would fully eliminate duplication and could also auto-generate tests. It was rejected because it introduces toolchain complexity (template authoring, build step, generated-file tracking) that is disproportionate to the problem: the field-specific logic inside each builder is already small and easily readable without generation. A simple helper function achieves the same nil-guard deduplication without the build tooling overhead.

### Consequences

#### Positive
- The file shrinks from 740 to 585 lines (a ~21% reduction), making it easier to scan and navigate.
- The nil-check and slice-initialization behavior is now consistent by construction across all builders — a future change to the shared pattern requires editing one function instead of 33.
- New tool builders only need to implement field-specific logic inside the callback, reducing the error surface for contributors adding new tools.

#### Negative
- `buildConstraints` uses Go generics (type parameter `[T any]`), which requires Go 1.18+. Readers unfamiliar with generic callback patterns in Go may need a moment to parse the abstraction.
- `addCommentConstraints` is a special case: it must return a non-nil slice even when `config` is nil (to append the trailing `"Supports reply_to_id for discussion threading."` constraint). It uses `buildConstraints` in an asymmetric way (`constraints := buildConstraints(...)` followed by `append(constraints, ...)`), making it slightly less uniform than the other builders.

#### Neutral
- All declarative differences between builders (message strings, field selectors, ordering) remain visible inside each function's callback, so the per-tool logic stays fully readable and easy to extend.
- The three new `append*Constraint` helpers are package-private and co-located with the existing `appendMaxConstraint` helper, following the established pattern in the file.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
