# ADR-29037: Time-Bounded Cache Eviction via `--after` Flag in the Logs Command

**Date**: 2026-04-29
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh aw logs` command downloads GitHub Actions workflow run logs and caches them locally in `run-{ID}` subdirectories under a configurable output directory. In shared storage environments — where multiple engineers or CI jobs share a single output directory — these cached folders accumulate without bound, consuming disk space indefinitely. Before this change, no built-in mechanism existed to prune stale cache entries; users were forced to write external scripts or perform manual cleanup. The date-delta format already used by `--start-date` and `--end-date` (e.g., `-1w`, `-30d`, `YYYY-MM-DD`) provides a natural, consistent way to express a cutoff.

### Decision

We will add a `--after` flag to `gh aw logs` that, when provided, deletes all cached `run-{ID}` directories whose creation date (taken from `run_summary.json` inside each folder, falling back to the directory's modification time) predates the specified cutoff before proceeding with the normal download. We will reuse the existing `workflow.ResolveRelativeDate` helper to parse the flag value so that relative deltas and absolute dates are handled consistently with the rest of the CLI. Cleanup failures are non-fatal: a warning is printed and the download proceeds regardless.

### Alternatives Considered

#### Alternative 1: Dedicated `gh aw logs clean` Subcommand

A separate subcommand (e.g., `gh aw logs clean --before -1w`) would make the destructive cache-eviction operation a first-class, explicit user action rather than a side-effect of the download command. This approach provides clearer UX separation — users who want to download logs would never accidentally trigger cleanup. It was not chosen because it adds a new top-level entry point for a narrow utility concern, and the intended primary use case is "clean up, then download in one step" (e.g., in a cron job maintaining a rolling window), which a combined flag handles more ergonomically.

#### Alternative 2: Time-Range Filter on Downloads (not cache eviction)

An `--after` flag could alternatively mean "only download runs created after this date", making it a symmetrical companion to `--start-date`. This was not chosen because `--start-date` already serves that role. Re-using `--after` as a download filter would create ambiguity with the existing flag semantics and would not solve the disk-space accumulation problem at all.

### Consequences

#### Positive
- Disk space management is available natively without requiring external scripts or cron jobs that call `rm -rf`.
- The flag reuses the same date-delta format (`-1w`, `-30d`, `YYYY-MM-DD`) already familiar to users of `--start-date` / `--end-date`, reducing the learning surface.
- Cleanup is non-fatal, so a transient file-system error does not block the download that follows.
- Only directories matching the `run-{ID}` naming pattern are touched, making the operation safe against accidentally deleting user-created files in the output directory.

#### Negative
- The `--after` name is ambiguous: a user reading the flag description for the first time may expect it to be a download date filter rather than a cache eviction trigger. The help text must be precise to avoid confusion.
- Cache eviction is a side-effect of the download command rather than a standalone operation; users who want to evict without downloading must still invoke `gh aw logs` even when they have no interest in downloading new runs.
- Adding yet another positional parameter to the already-wide `DownloadWorkflowLogs` function signature increases the cognitive cost of calling that function from tests.

#### Neutral
- All existing call sites of `DownloadWorkflowLogs` must be updated to pass an `after` string; passing an empty string disables cleanup, preserving backward compatibility.
- The fallback from `run_summary.json` creation timestamp to directory modification time means that eviction behaviour may differ slightly for incomplete downloads, but this is bounded and documented.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Cache Eviction Scope

1. Implementations **MUST** restrict cache eviction to directories whose names match the `run-{ID}` prefix pattern; directories with any other name **MUST NOT** be deleted.
2. Implementations **MUST** determine a directory's effective run date by reading the `CreatedAt` field from `run_summary.json` inside that directory when the file is present and parseable.
3. Implementations **MUST** fall back to the directory's file-system modification time as the effective run date when `run_summary.json` is absent or unparseable.
4. Implementations **MUST** delete a `run-{ID}` directory if and only if its effective run date is strictly before the resolved cutoff timestamp.

### Cutoff Parsing

1. Implementations **MUST** accept the `--after` flag value in either absolute (`YYYY-MM-DD`) or relative delta (`-Nd`, `-Nw`, `-Nmo`) formats.
2. Implementations **MUST** resolve relative delta values using the same `workflow.ResolveRelativeDate` helper used by `--start-date` and `--end-date` to ensure consistent date arithmetic across the CLI.
3. Implementations **MUST** return a user-facing error and abort if the provided `--after` value cannot be parsed into a valid cutoff timestamp.

### Error Handling and Ordering

1. Implementations **MUST** execute cache eviction before initiating any new log downloads when `--after` is specified.
2. Implementations **MUST NOT** abort the download step if cache eviction encounters a file-system error; the error **MUST** be surfaced as a non-fatal warning on stderr.
3. Implementations **SHOULD** print a human-readable summary of removed folder count on stderr when one or more directories are deleted.
4. Implementations **MAY** suppress the "nothing to clean" message unless `--verbose` is also set.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25090240769) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
