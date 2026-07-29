# ADR-48788: Merge `actions/checkout` `with:` Block for `--logical-repo` Trial Rewriter

**Date**: 2026-07-29
**Status**: Draft
**Deciders**: Unknown (copilot-swe-agent, pelikhan)

---

### Context

The `gh aw trial --logical-repo` command rewrites GitHub Actions workflow files on-the-fly to point checkout steps at a simulated logical repository. The original implementation used a single-pass regex that matched the `actions/checkout` `uses:` line and unconditionally appended a new `with:` block (with `repository:`) immediately after the matched line. This produced invalid YAML whenever a checkout step already had a `with:` block, because YAML does not allow duplicate mapping keys. The bug affected common real-world workflows that pass additional checkout inputs such as `ref`, `path`, `token`, or `persist-credentials`. Checkout steps where `with:` appears after other step keys (e.g., `id:`, `name:`) were also not handled correctly.

### Decision

We will replace the single-pass, pattern-match-and-append approach with a step-boundary-aware, line-indexed merge algorithm. When a `actions/checkout` `uses:` line is found, the rewriter walks forward within the step's indent scope to locate an existing `with:` block. If one exists, it injects or replaces the `repository:` key inside that block rather than creating a new `with:` block. Only when no `with:` block exists does it create one. The regex is also updated to match both quoted and unquoted `uses:` forms.

### Alternatives Considered

#### Alternative 1: Parse and rewrite with a full YAML library

Use a Go YAML library (e.g., `gopkg.in/yaml.v3`) to parse the workflow file, mutate the AST, and serialize it back. This would correctly handle all YAML edge cases including multi-line values, anchors, aliases, and non-standard indentation.

This approach was not chosen because it requires introducing a new dependency and risks changing cosmetic formatting (comments, blank lines, key order, indentation style) that is unrelated to the intended mutation, making diffs noisy and potentially breaking fragile workflows. The existing codebase uses line-based string manipulation for workflow rewriting throughout, and the scope of the fix is limited to a narrow YAML sub-problem.

#### Alternative 2: Detect duplicate `with:` post-hoc and remove the second one

Keep the existing append logic but scan the output for consecutive `with:` blocks at the same indent level and merge them after the fact.

This approach was not chosen because post-hoc deduplication is harder to reason about and error-prone: it requires another pass, must handle edge cases around intervening blank lines, and still cannot handle the case where `with:` appears after sibling step keys (e.g., `id:`, `name:`) because the original regex does not scan past the `uses:` line.

#### Alternative 3: Conditional append — only emit `with:` when absent

Add a look-ahead check: after matching the `uses:` line, scan the next few lines to see if `with:` already exists, and if so, skip the entire step.

This approach was not chosen because it silently leaves the `repository:` key unset in checkout steps that already have a `with:` block, defeating the purpose of the `--logical-repo` flag in the most common real-world case.

### Consequences

#### Positive
- Checkout steps with existing `with:` blocks (including `ref`, `path`, `token`, `persist-credentials`, `fetch-depth`, etc.) are now correctly rewritten without producing duplicate `with:` keys.
- Quoted `uses:` forms (`uses: "actions/checkout@v5"`) are now matched by the updated regex, broadening coverage.
- Existing `repository:` values are replaced in-place rather than duplicated, producing a clean diff.
- Other checkout inputs are preserved unchanged, reducing risk of breaking workflows that depend on specific checkout options.

#### Negative
- Line-based YAML manipulation remains inherently fragile: the step-boundary heuristic (comparing indent widths) can be confused by unusual YAML constructs such as flow-style mappings, block scalars, or non-standard indentation. A correct long-term solution would require a proper YAML AST.
- The rewriting logic is now stateful and index-based rather than a simple map over lines, which increases cognitive complexity and makes future changes to the rewriter harder to reason about.
- The algorithm assumes two-space indentation for child keys inside `with:`. Workflows using four-space or tab indentation may produce incorrectly aligned output.

#### Neutral
- Three new helper functions (`leadingIndentWidth`, `isCheckoutStepBoundary`, `isIndentedKeyLine`) are added to `trial_repository.go`, expanding the surface area of the module slightly.
- Existing test cases that previously encoded the (now-invalid) duplicate-`with:` behavior are updated to reflect correct output.
- New test cases cover delayed `with:` placement and quoted `uses:` syntax.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
