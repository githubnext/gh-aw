# ADR-41231: Split threat_detection.go into Focused Single-Responsibility Modules

**Date**: 2026-06-24
**Status**: Draft
**Deciders**: Unknown (PR #41231 author)

---

### Context

`pkg/workflow/threat_detection.go` had grown to 1542 lines, mixing configuration parsing, helper predicates, inline step builders, inline engine execution, external-detector orchestration, and top-level job assembly in a single file. A file of this size is hard to navigate, hard to review, and obscures the natural responsibility boundaries within the threat-detection subsystem. The repository maintains a convention of keeping source files under a ~500-line target so that each unit of code stays reviewable in isolation. This is a pure structural reorganization: no behavior, control flow, or public API changes are intended.

### Decision

We will split `threat_detection.go` into six focused files, each scoped to a single responsibility and each under the 500-line target, while keeping all of them in `package workflow` so no import paths change for callers. The split is: `threat_detection_config.go` (config struct, parsers, `extractRawExpression`), `threat_detection_helpers.go` (shared predicates and constants), `threat_detection_steps.go` (inline step builders), `threat_detection_inline_engine.go` (`buildDetectionEngineExecutionStep`), `threat_detection_external.go` (external-detector install/run/conclude), and `threat_detection_job.go` (`buildDetectionJob` assembler). The primary driver is reviewability: smaller, responsibility-aligned files are easier to read, navigate, and reason about.

### Alternatives Considered

#### Alternative 1: Keep `threat_detection.go` as a single file

Leave the code as one 1542-line file. This avoids churn and keeps all related code physically adjacent. Rejected because the file had already crossed the point where its size impeded navigation and review, and it violated the repository's file-size convention. The cost of a one-time split is small relative to the recurring friction of maintaining an oversized file.

#### Alternative 2: Split into five files per the original issue proposal

The originating issue proposed a five-file split with a ~460-line `steps.go`. Rejected because that estimate excluded `buildDetectionEngineExecutionStep` (~168 lines); folding it into `steps.go` would push that file over the 500-line target. Extracting it into a dedicated `threat_detection_inline_engine.go` keeps every resulting file comfortably under the target, at the cost of one additional file.

### Consequences

#### Positive
- Every resulting file is under the ~500-line target, restoring per-file reviewability.
- Responsibility boundaries (config, helpers, inline steps, inline engine, external detectors, job assembly) are now explicit at the file level, easing navigation.
- Public API (`IsDetectionJobEnabled`, `IsConditionalDetection`, `ThreatDetectionConfig`) is unchanged, so no caller is affected.

#### Negative
- Logic that previously lived in one file is now spread across six, requiring cross-file navigation to follow some flows end to end.
- A larger file count increases the surface for future merge conflicts and makes the directory listing longer.

#### Neutral
- All files remain in `package workflow`; the split is purely physical (file organization), not logical (package boundaries).
- Because there is no behavioral change, existing tests should pass unmodified and serve as the regression guard for the move.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
