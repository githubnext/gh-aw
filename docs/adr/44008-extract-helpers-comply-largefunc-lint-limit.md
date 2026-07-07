# ADR-44008: Extract Package-Private Helpers to Comply with the `largefunc` Lint Limit

**Date**: 2026-07-07
**Status**: Draft
**Deciders**: Unknown

---

### Context

The repository enforces a `largefunc` lint rule capping function bodies at 60 lines. Four functions in `pkg/workflow` exceeded this limit after organic growth: `ResolveSHA` (67 lines), `computeAntigravityToolsCore` (65 lines), `buildInputSchema` (69 lines), and `getShellEnvironmentPolicyVars` (61 lines). All four contained self-contained logical sub-problems—semver pin lookup, bash tool expansion, JSON Schema property construction, and MCP env-var mapping—that were implemented as inline blocks. Leaving these violations unaddressed blocks the lint gate and defers the structural clarity cost indefinitely.

### Decision

We will extract each over-limit inline block into a dedicated, package-private helper function co-located in the same source file, with no changes to observable behavior. The helpers (`lookupEmbeddedActionPin`, `appendBashTools`, `buildChoiceInputProperty`, `buildInputProperty`, `addMCPToolEnvVars`) own exactly one sub-problem each and are named after their responsibility rather than their caller. This satisfies the `largefunc` lint limit without raising the threshold or suppressing the check.

### Alternatives Considered

#### Alternative 1: Raise or Suppress the `largefunc` Threshold

The lint threshold could be increased (e.g., from 60 to 80 lines) or suppressed with a per-file `//nolint` comment for the four affected files. This avoids any code restructuring and is trivially fast to apply.

Rejected because it signals that the lint rule is optional and sets a precedent for future exemptions. The four functions each contained a genuinely separable concern; the lint violation was a signal worth acting on rather than silencing.

#### Alternative 2: Promote Helpers to a New Sub-Package

The extracted logic (semver pin matching, bash tool expansion, MCP env-var mapping) could be placed in separate sub-packages (e.g., `pkg/workflow/actionpin`, `pkg/workflow/bashtools`) to give them stronger encapsulation and independent test files.

Rejected because the helpers are not reused outside `pkg/workflow` and the sub-package boundary would add import indirection with no practical benefit. Package-private functions in the same file achieve the same readability improvement with less structural overhead.

### Consequences

#### Positive
- All four functions now comply with the 60-line `largefunc` lint rule, unblocking the lint gate.
- Each extracted helper is independently unit-testable without constructing the full parent receiver.
- Callers of the parent functions are shorter and read at a higher level of abstraction.

#### Negative
- The package now has five additional unexported functions, slightly increasing its exported-symbol footprint as seen by IDEs and documentation tools.
- Readers who want to understand a full call path must navigate one extra indirection for each extracted helper.

#### Neutral
- All helpers are package-private (`lowercase` names) and co-located with their caller; no public API surface changes.
- The refactoring carries zero behavior change risk, but reviewers still need to verify the extraction did not alter edge-case handling (e.g., the early-`continue` logic in `buildInputSchema`'s choice branch).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
