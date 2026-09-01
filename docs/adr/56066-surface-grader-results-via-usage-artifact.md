# ADR-56066: Surface Grader Results via the Usage Artifact and CLI Commands

**Date**: 2026-08-26
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The gh-aw agent job writes grader outcomes (`grader_manifest.json` and `grader_results.json`) into the `agent` artifact under `/tmp/gh-aw/agent/graders/`. However, the compact `usage` artifact — which drives `gh aw audit` and `gh aw logs` — did not collect those files, so grader results were invisible to CLI consumers. Evals were already collected and surfaced through this path; graders were not. Operators running `gh aw audit` or `gh aw logs --json` had no way to retrieve or display grader outcomes without manually downloading the full `agent` artifact.

### Decision

We will mirror grader files from the `agent` artifact into the `usage` artifact at collection time, and add grader-rendering logic to `gh aw audit` and `gh aw logs --json`. The `collect_usage_artifact_files.sh` script copies `grader_manifest.json` and `grader_results.json` into `/tmp/gh-aw/usage/graders/`, which are then included in the `usage` upload paths. The CLI resolves grader files across multiple search paths (`usage/graders`, `graders`, `agent/graders`, run root) to accommodate `workflow_call` hash-prefixed directory layouts and logs the joined manifest+results data.

### Alternatives Considered

#### Alternative 1: Create a Dedicated `graders` Artifact

Introduce a separate GitHub Actions artifact (e.g., `graders`) specifically for grader output, analogous to how some systems separate test results from build outputs.

This would have given graders a clean, independent download path. It was rejected because it requires new artifact upload steps in every workflow lock file, increases artifact storage and download costs, and adds infrastructure complexity without meaningful benefit — the `usage` artifact already provides a compact, CLI-optimized surface that exactly matches what `gh aw audit` and `gh aw logs` need.

#### Alternative 2: Fetch Graders Directly from the `agent` Artifact in the CLI

Modify the CLI to download and parse grader files from the `agent` artifact rather than the `usage` artifact.

This avoids any changes to the artifact collection pipeline. It was rejected because the `agent` artifact is large and downloading it solely for grader data would be slow and wasteful. The `usage` artifact already exists as the canonical compact data surface for CLI tools, and evals follow exactly this pattern — consistency favors mirroring graders into `usage` as well.

### Consequences

#### Positive
- `gh aw audit` now renders a graders section in console output and includes a `graders` object in the JSON report, giving operators visibility into grader pass/fail outcomes without manual artifact inspection.
- `gh aw logs --json` carries per-run `graders` data, enabling programmatic grader-outcome queries over run history.
- No new artifact type is needed; the existing `usage` artifact and CLI infrastructure are reused, keeping the overall system simpler.
- A new `graders` artifact-set filter is available via `--artifacts`, consistent with the existing `evals` and other artifact-set conventions.

#### Negative
- `pkg/workflow/notify_comment.go` had to be refactored — split into `buildConclusionJobSteps`, `buildUsageArtifactInputDownloadSteps`, and `computeConclusionJobPermissions` — to stay under the repository's function-size lint limit. This refactor is purely structural and introduces no behavior change, but it is additional churn.
- The grader file lookup logic must handle multiple search paths (`usage/graders`, `graders`, `agent/graders`, hash-prefixed run-root dirs) to support `workflow_call` layouts, adding path-resolution complexity to the CLI.

#### Neutral
- All ~250 workflow lock files must be regenerated to include the two new `usage` upload paths, producing a large but mechanically generated diff.
- The `findGraderFile` helper is now shared between `audit_report_graders.go` and `experiments_grader_observations.go`, eliminating duplicate lookup logic.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
