# ADR-43409: Post-Parse Diagnostic Improvement for Tool Config Shape Violations

**Date**: 2026-07-05
**Status**: Draft
**Deciders**: Unknown (generated from PR diff)

---

### Context

When a user writes a workflow frontmatter `tools.<name>` key as a scalar string (e.g. `github: "invalid-string"`) and also includes nested child keys beneath it (e.g. `toolsets: [default]`), the upstream YAML parser produces two quality problems:

1. **Wrong source location** — the error caret lands on the child key line rather than on the scalar value that is the actual mistake.
2. **Parser-internal phrasing** — the raw message ("value is not allowed in this context. map key-value is pre-defined") is meaningless to workflow authors and doesn't name the tool key involved.

The compiler also failed to classify this diagnostic as a syntax error, causing it to surface generic event/filter remediation hints instead of syntax remediation guidance. These combined issues made the error experience confusing and hard to act on.

### Decision

We will intercept and rewrite affected YAML diagnostics in a new post-parse adjustment layer (`improveFrontmatterDiagnostic` in `pkg/workflow/frontmatter_error.go`) rather than modifying the upstream YAML parser or catching the pattern pre-parse. This function detects the scalar-with-nested-key diagnostic pattern, re-anchors the reported position to the scalar value line, and replaces the message with a tool-aware phrase: `tools.<name> tool config must be an object, not a string (for example: toolsets: [default])`. The error classifier is also extended to recognise the translated message as a syntax-category error.

### Alternatives Considered

#### Alternative 1: Modify the Upstream YAML Parser

Patch the third-party YAML parser to emit the error at the parent scalar line with a user-friendly message at parse time.

Considered because it would fix the issue at the root cause. Not chosen because it requires forking or upstreaming a change to an external library, is high-risk for regressions across all YAML error paths, and is disproportionate effort for a single error pattern.

#### Alternative 2: Pre-Parse Structural Validation

Before invoking the YAML parser, scan the raw frontmatter text for the `key: "scalar"\n  child:` pattern and produce a custom error without involving the YAML parser at all.

Considered because it avoids dependence on parser error message text (which can change). Not chosen because it duplicates partial YAML understanding in the pre-parse layer, cannot leverage the parser's existing line/column tracking, and adds a fragile regex-based YAML structural check outside the normal parse pipeline.

#### Alternative 3: Generic "Review Your Tools Config" Fallback Message

When the parser emits this error pattern, replace the message with a generic "check your tools configuration" hint without attempting to re-anchor the source position.

Considered as a low-effort improvement. Not chosen because it still leaves the caret on the wrong line, which is the more confusing of the two problems, and because the tool name is available from context and makes the message significantly more actionable.

### Consequences

#### Positive
- Workflow authors see the error caret on the line they actually need to fix (the scalar value line) rather than on an unrelated child key line.
- The message names the specific tool key involved (`tools.github`) and provides a concrete correct example, reducing iteration time to fix the mistake.
- The fix is isolated to the error-formatting layer and does not affect the parse, compile, or validation pipelines.
- The syntax-category classification ensures users receive the correct remediation hint ("Fix the YAML/frontmatter syntax first") instead of an unrelated event/filter hint.

#### Negative
- The detection heuristic relies on matching translated message text and regex-parsing source lines; if the upstream parser changes its raw message or the translation changes, the heuristic silently stops firing (falling back to the original, lower-quality diagnostic).
- The approach is specific to the `tools.<name>` ancestor context; other scalar-with-children patterns elsewhere in frontmatter are not improved.
- `frontmatter_error.go` now carries a third special-case adjustment path, increasing the maintenance surface of the error formatting layer.

#### Neutral
- A custom `renderSourceContextForPosition` helper is added to re-render the source context snippet when the position is re-anchored, keeping the VSCode-compatible output format consistent.
- The `parsePositiveInt` helper duplicates a simple int-parsing pattern rather than importing a shared utility, keeping the dependency surface minimal.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
