# ADR-0001: Enforce File Size Limit via Domain-Based Decomposition

**Date**: 2026-04-10
**Status**: Draft
**Deciders**: pelikhan, Copilot (SWE agent)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh-aw` codebase uses an automated architecture linter that flags Go source files exceeding 1,000 lines as **BLOCKER** violations on the grounds that large files mix unrelated concerns, reduce discoverability, and slow code review. Six files in `pkg/cli/` and `pkg/workflow/` had grown well beyond this threshold—some to over 1,300 lines—because feature additions were accumulated in existing files rather than being distributed across new, focused ones. The project needed a clear, repeatable strategy for how to break such files apart and where each extracted function should land.

### Decision

We will enforce a hard 1,000-line cap on Go source files in this repository and resolve violations by splitting the offending file into multiple **domain-specific sibling files**. Each sibling file is named after its parent with a suffix that describes the concern it owns (e.g., `gateway_logs_parser.go`, `gateway_logs_renderer.go`, `gateway_logs_extractor.go`). When a large file contains a monolithic registry variable (such as `handlerRegistry`), we will initialize the registry as an empty map in the primary file and use Go's `init()` functions in domain-specific files to register entries, one domain per file. No logic changes are permitted during a decomposition commit; the split is purely structural.

### Alternatives Considered

#### Alternative 1: Increase or Remove the Line Limit

Accept files longer than 1,000 lines and raise the threshold (or remove it entirely). This was rejected because the existing violations were already causing demonstrable pain: PR diffs against a 1,300-line file are hard to review, `git blame` across such files is noisy, and the underlying mixing of parsing, rendering, and extraction logic in a single file made isolated unit testing difficult. Removing the limit would institutionalize that pain.

#### Alternative 2: Extract to Separate Packages Instead of Sibling Files

Move extracted functions into wholly new sub-packages (e.g., `pkg/cli/gateway/parser`, `pkg/cli/gateway/renderer`). This is a cleaner long-term architecture but was rejected for this iteration because it would require updating all import paths and call sites across the codebase—a larger, riskier change that mixes structural refactoring with the goal of simply clearing BLOCKER violations. A future ADR may revisit this as the package surface stabilises.

#### Alternative 3: Inline the Extracted Code via Interface Dispatch

Introduce interfaces and replace the large functions with smaller implementations spread across types. This was rejected because it would constitute a logic change (not merely a structural split), complicating verification that no behaviour was altered. Keeping the refactor purely mechanical—move functions, change no logic—was a non-negotiable constraint for this PR.

### Consequences

#### Positive
- All six BLOCKER files are now under 1,000 lines; the architecture linter gate passes.
- Each sibling file has a single-concern scope (parsing, rendering, extraction, concurrency, etc.), making it significantly easier to locate and review related code.
- The `init()`-based registry pattern allows future handler domains to be added in their own files without touching the primary config file, reducing merge conflicts.
- Diffs are smaller and more focused per file, improving pull-request reviewability going forward.

#### Negative
- The total file count in `pkg/cli/` and `pkg/workflow/` increases, which adds a small cognitive overhead when navigating the directory listing.
- Sibling files share the same Go package, so there is no compile-time enforcement of the "one concern per file" boundary; discipline is required to prevent concern-creep back into files over time.
- The `init()` registration pattern for registries introduces implicit ordering dependency on the Go runtime's package initialisation order; it is correct but non-obvious to developers unfamiliar with the pattern.
- JavaScript BLOCKER files (`log_parser_shared.cjs` at 1,703 lines, `create_pull_request.cjs` at 1,666 lines, and others) remain above threshold and are out of scope for this PR, creating an inconsistent enforcement state between Go and JavaScript.

#### Neutral
- This refactor requires updating the architecture linter's allowlist (if any) to reflect the new file structure.
- Future contributors adding functionality to these packages should default to creating new domain-specific files rather than extending existing ones past the threshold.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### File Size Constraint

1. Go source files in this repository **MUST NOT** exceed 1,000 lines (including blank lines and comments).
2. A file that exceeds 1,000 lines **MUST** be decomposed before or as part of the pull request that causes it to exceed the threshold; it **MUST NOT** be merged in a violating state.
3. JavaScript source files **SHOULD** also respect the 1,000-line limit; enforcement for JavaScript is deferred to a follow-up initiative [TODO: verify timeline].

### Decomposition Strategy

1. When a Go file must be split, the primary file **MUST** retain its original name and **MUST** keep the package-level declarations (types, constants, package-scope variables, and the primary entry points) that give the file its identity.
2. Extracted functions **MUST** be moved to sibling files within the same directory and package; they **MUST NOT** be relocated to a different package unless a separate architectural decision records that refactoring.
3. Sibling files **MUST** be named using the pattern `{original_basename}_{concern}.go`, where `{concern}` is a short, lowercase, single-word or hyphenated descriptor of the responsibility the file owns (e.g., `_parser`, `_renderer`, `_extractor`, `_helpers`, `_concurrency`, `_network`).
4. A decomposition commit **MUST NOT** alter function signatures, logic, or behaviour; it **MUST** be a purely mechanical relocation of code.

### Registry Splitting via `init()`

1. When a large registry variable (a `map` populated with inline closures) must be split, the registry variable **MUST** be declared and initialised as an empty map in the primary file.
2. Each domain group of registry entries **MUST** be registered in a dedicated `init()` function in its own domain-specific sibling file.
3. Each domain sibling file **MUST** register only entries belonging to its declared domain; it **MUST NOT** register entries from another domain's file.
4. Registry sibling files **SHOULD** be named `{primary_file}_{domain}.go` (e.g., `compiler_safe_outputs_handlers_issues.go`).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically:

- Every Go source file in the repository is ≤ 1,000 lines, OR a decomposition is in progress under an open pull request.
- All decompositions use sibling files in the same package with the prescribed naming pattern.
- No decomposition commit introduces logic changes.
- Registry splits declare an empty map in the primary file and use `init()` functions in domain siblings.

Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent](https://github.com/github/gh-aw/actions/runs/24252159284). Review and finalize before changing status from Draft to Accepted.*
