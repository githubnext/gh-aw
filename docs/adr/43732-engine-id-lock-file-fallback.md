# ADR-43732: Two-Tier Fallback for Resolving engine_id in Workflow Run Logs

**Date**: 2026-07-06
**Status**: Draft
**Deciders**: pelikhan (PR author), copilot-swe-agent (implementer)

---

### Context

The `gh aw logs` command supports an `--engine` filter that matches workflow runs against an `engine_id`. This ID was previously sourced exclusively from an `aw_info.json` artifact uploaded during workflow execution. Older workflow runs (e.g., runs from the `daily-cli-tools-tester` workflow family that pre-date `aw_info.json` adoption) never upload this artifact, causing `engine_id` to resolve to `null`. When null, the `--engine` filter silently excludes these runs, making historical data invisible to users. The lock files (`.lock.yml`) for agentic workflows already embed a `# gh-aw-metadata:` JSON comment that contains an `agent_id` field equivalent to `engine_id`, so a fallback source already exists within the repository itself.

### Decision

We will establish a deterministic two-tier precedence for resolving `engine_id` when processing workflow run logs:

1. **Authoritative**: `aw_info.json` artifact's `engine_id` field — used whenever the artifact is present and non-empty.
2. **Fallback**: the `agent_id` field parsed from the `# gh-aw-metadata:` comment embedded in the run's `.lock.yml` file — consulted only when `aw_info.json` is absent or carries an empty `engine_id`.
3. **Unknown**: if neither source yields a value, `engine_id` remains unresolvable and the run is excluded from `--engine` filter results.

This precedence is implemented in `logs_parsing_core.go` (new `extractEngineIDFromLockFile` and `resolveToLockFilePath` helpers), `logs_orchestrator.go` (fallback parameter added to `matchEngineFilter`), and `logs_report.go` (fallback applied in `buildLogsData`).

### Alternatives Considered

#### Alternative 1: Require all workflows to adopt aw_info.json and do nothing for older runs

Accept that older runs without `aw_info.json` are excluded from `--engine` filter results as a transitional cost. This is simpler — no fallback logic required. This was rejected because the `daily-cli-tools-tester` family represents a significant portion of historical run data, and silently dropping those runs from engine-scoped queries yields misleading results with no user-visible warning.

#### Alternative 2: Back-fill aw_info.json artifacts for historical runs

Retroactively upload `aw_info.json` artifacts for all pre-existing runs missing them. This would make the data consistent and avoid dual-source complexity. This was rejected because GitHub Actions artifacts cannot be added to completed runs after the fact via the public API; the approach is not technically feasible.

#### Alternative 3: Introduce a separate metadata cache (e.g., a database or side-car file)

Maintain a persistent mapping of workflow run ID → engine_id in a side-car cache (e.g., a JSON file in the repo-memory branch). This was rejected because it introduces operational complexity (cache invalidation, race conditions across concurrent runs) and offers no advantage over the lock file source, which is already co-located in the repository and read-only after merge.

### Consequences

#### Positive
- Older workflow runs that pre-date `aw_info.json` adoption now appear correctly in `--engine` filtered queries, restoring historical data visibility.
- The lock file's `gh-aw-metadata` comment becomes a formally recognized secondary source for engine identity, leveraging data that was already present.
- `aw_info.json` retains full authority; the fallback does not alter behavior for any run that already populates it.
- The precedence hierarchy is explicit and documented in code, reducing the risk of future contributors inadvertently changing resolution order.

#### Negative
- The `engine_id` resolution path now has two code paths that must stay in sync — if the lock file format for `gh-aw-metadata` changes, the fallback parser (`extractEngineIDFromLockFile`) must be updated separately.
- Resolving engine_id now requires reading a local file (`.lock.yml`) for older runs, adding a file I/O dependency that was not present before; runs whose lock files are missing or malformed will silently fall back to `engine_id: unknown`.
- The `matchEngineFilter` function signature changed (added `lockFileEngineID` parameter), requiring all call sites to be updated — a small but non-zero refactoring cost for future callers.

#### Neutral
- The `resolveToLockFilePath` helper normalises `.yml`, `.md`, and `.lock.yml` inputs to a canonical `.lock.yml` path; this normalization is reusable but currently only used in one context.
- The fallback only applies to `engine_id` — other metadata fields in `aw_info.json` do not gain a lock-file fallback, keeping the scope of this change narrow.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
