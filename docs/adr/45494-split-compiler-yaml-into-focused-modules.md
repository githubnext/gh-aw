# ADR-45494: Split compiler_yaml.go into Focused Single-Domain Modules

**Date**: 2026-07-15
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/workflow/compiler_yaml.go` grew to 1262 lines spanning five unrelated functional domains: YAML normalization, workflow header emission, prompt assembly, step generation, and top-level orchestration. A single file of that size is difficult to navigate, produces large and hard-to-review diffs, and makes it harder to locate a specific concern without reading or searching the entire file. The file exceeded the project's informal 800-line threshold that signals a file has taken on too many responsibilities.

### Decision

We will split `compiler_yaml.go` into four focused files — `compiler_yaml_normalize.go`, `compiler_yaml_header.go`, `compiler_yaml_prompt.go`, and `compiler_yaml_steps.go` — leaving `compiler_yaml.go` as a thin orchestration layer (~221 lines). All functions remain in the `workflow` package with identical signatures; this is a purely structural reorganization with no behavior changes.

### Alternatives Considered

#### Alternative 1: Keep the monolithic file with section comments

Add large comment banners (e.g., `// ===== NORMALIZATION =====`) to visually separate the five domains within the single file. This avoids creating new files and keeps navigation in one place via IDE search.

This was not chosen because it does not reduce the cognitive load of holding 1262 lines in context during code review or debugging, and it provides no tooling-level separation. File boundaries in Go are the idiomatic way to group related declarations.

#### Alternative 2: Extract into a separate sub-package

Move the four functional domains into a dedicated `compileryaml/` sub-package, exposing only the symbols needed by the rest of `pkg/workflow/` through an explicit API surface.

This was not chosen because it would require exporting functions that are currently unexported (or restructuring callers), adding non-trivial churn for what is fundamentally a maintenance refactor. The same-package file split achieves the navigability benefit without breaking any existing API contracts or requiring callers to be updated. A future ADR can revisit sub-package extraction if the package boundary becomes valuable.

### Consequences

#### Positive
- Each file is 167–374 lines, well within readable limits, making code review diffs smaller and more focused.
- Developers can navigate directly to the domain they are working on (normalization, header, prompt, steps) without scrolling through an unrelated 1200-line file.
- Separation of concerns is now visible at the file-system level, making it easier for new contributors to understand the compiler's structure.
- Future domain-level changes produce smaller, more readable diffs.

#### Negative
- Go's `package`-level visibility means there is no enforced encapsulation boundary between the four files; any function can still call any other unexported function in the package. The separation is organizational, not contractual.
- Developers navigating the full compilation pipeline must open multiple files rather than scrolling within one; IDE "go to definition" mitigates this but it is a real cost.

#### Neutral
- All function signatures, exported symbols, and package membership are unchanged — no callers inside or outside the package need to be updated.
- The split sets a precedent for how to handle future file-size growth in the `pkg/workflow` package: split by functional domain within the same package before escalating to a sub-package extraction.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
