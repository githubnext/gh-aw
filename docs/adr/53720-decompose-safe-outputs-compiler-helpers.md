# ADR-53720: Decompose Safe-Outputs Compiler Helpers into Focused Files

**Date**: 2026-08-18
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`compiler_safe_outputs_job.go` had grown past the project's file-size guideline. The function `buildJobLevelSafeOutputEnvVars` alone was 144 lines, and several artifact-upload and custom-script helper functions were co-located with job-level orchestration logic, making the file difficult to review, navigate, and evolve. The `safe_outputs` job compiler is a high-churn area where multiple contributors work concurrently, so large files create merge conflicts and slow code review.

### Decision

We will extract helpers from `compiler_safe_outputs_job.go` into two new files grouped by responsibility:

- `compiler_safe_outputs_envvars.go` — holds `buildJobLevelSafeOutputEnvVars` (environment variable construction for the job level).
- `compiler_safe_outputs_steps.go` — holds artifact-upload step builders (`buildSafeOutputItemsManifestUploadStep`, `buildSarifArtifactUploadStep`) and custom safe-output script helpers (`scriptNameToHandlerName`, `generateSafeOutputScriptContent`, `buildCustomScriptFilesStep`).

The compiler's public interface, generated YAML output, and all existing tests remain unchanged.

### Alternatives Considered

#### Alternative 1: Retain all code in `compiler_safe_outputs_job.go` with section comments

Add clear `// === Environment Variables ===` delimiters inside the single file to improve navigation without adding new files. Why not chosen: comments do not enforce boundaries at the Go package level and provide no compile-time feedback if a future change moves a function to the wrong section. The file would remain large and difficult to diff.

#### Alternative 2: Extract to a single new helper file (`compiler_safe_outputs_helpers.go`)

Move all non-orchestration code into one new file instead of two thematically-grouped files. Why not chosen: env-var construction and step-building are distinct concerns with different import sets (`encoding/json` vs `strings`/`sliceutil`); splitting them further reduces import surface per file and makes future extraction easier if either group grows again.

### Consequences

#### Positive
- Each file has a single, well-defined responsibility, reducing cognitive load when locating or reviewing a specific helper.
- `compiler_safe_outputs_job.go` is reduced from ~900+ lines to 765 lines, within the file-size guideline.
- Smaller files produce smaller, more focused diffs, reducing merge-conflict surface for concurrent contributors.

#### Negative
- Readers must learn that the `workflow` package is organized by concern type across multiple `compiler_safe_outputs_*.go` files rather than by call-site proximity.
- The total file count in `pkg/workflow/` increases by two, adding minor navigation overhead.

#### Neutral
- The compiler interface, generated YAML output, and all existing tests are unaffected — this is a pure source reorganization.
- Import sets in `compiler_safe_outputs_job.go` shrink (`encoding/json`, `sort`, `sliceutil` removed), which may reduce future accidental coupling.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
