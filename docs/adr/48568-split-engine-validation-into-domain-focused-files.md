# ADR-48568: Split engine_validation.go Into Domain-Focused Files

**Date**: 2026-07-28
**Status**: Draft
**Deciders**: Unknown (automated refactor by copilot-swe-agent)

---

### Context

`pkg/workflow/engine_validation.go` had grown to 521 lines by mixing three unrelated validation domains in a single file: engine driver and harness script filename safety checks, inline engine definition and auth strategy registration, and top-level engine settings validation (version pinning, MCP timeouts, multi-file specification consistency). The existing 300-line refactor threshold existed specifically to prevent files from mixing unrelated concerns, which makes individual domains harder to navigate, extend, and test in isolation.

### Decision

We will split `engine_validation.go` into three domain-focused files within the same `workflow` package: `engine_driver_validation.go` (driver/harness script path safety, ~171 lines), `engine_inline_definition_validation.go` (inline runtime/provider auth registration, ~177 lines), and a trimmed `engine_validation.go` (~255 lines, retaining version, MCP timeout, and multi-file specification validation). No logic changes, no API surface changes — this is a pure structural reorganization. Each file includes a header comment explaining its responsibility and directing contributors to the right file for future additions.

### Alternatives Considered

#### Alternative 1: Keep the monolithic file

Leave `engine_validation.go` as-is and continue adding to it. This avoids merge risk and churn but the file will continue to grow — the three validation domains have independent growth trajectories (new driver runtimes, new auth strategies, new MCP settings). Rejected because the 521-line size already exceeds the repository's stated 300-line threshold, and mixing domains increases the risk that unrelated changes conflict in the same file.

#### Alternative 2: Extract to a dedicated sub-package (`pkg/workflow/enginevalidation/`)

Move the engine validation logic into its own sub-package. This would provide stronger encapsulation via Go's package visibility rules and prevent accidental access to internal types. Rejected for this PR because it would change import paths, may require exporting currently unexported types (breaking encapsulation in the other direction), and is a larger structural change with broader impact that warrants its own explicit decision. The within-package file split is the lower-risk, immediately actionable step.

#### Alternative 3: Split only the largest domain (driver validation)

Extract only `validateEngineDriver` and related helpers into a separate file, leaving inline definition and auth validation in `engine_validation.go`. This would partially address the size issue. Rejected because the inline definition and auth domain is equally unrelated to the top-level engine settings remaining in `engine_validation.go`, so a partial split would still leave the file mixing concerns and require a follow-up split anyway.

### Consequences

#### Positive
- Each file is under 260 lines and owns a single validation domain, reducing cognitive load when navigating engine validation logic.
- The new file headers explicitly state what to add where, creating a self-enforcing contribution convention for future engine validation work.
- Driver and inline definition validation can now be tested and reviewed independently without reading unrelated validation code.

#### Negative
- The number of files in `pkg/workflow/` increases by two, adding some noise to directory listings.
- Future growth of any of the three new files may require further splitting; this reorganization does not eliminate the risk of future monolith accumulation.

#### Neutral
- All function signatures are preserved and all existing tests pass unchanged — the split is transparent to callers.
- The `workflow` package boundary is preserved; no import path changes are required.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
