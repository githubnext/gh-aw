# ADR-28452: Pre-Download Workflow Exclusion Filter for the Logs Command

**Date**: 2026-04-25
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh aw logs` command downloads workflow run artifacts and logs for analysis. As the number of workflows in a repository grows, users frequently need to download logs for only a subset of workflows — for example, skipping noisy CI workflows that are not relevant to a particular investigation. Without an exclusion mechanism, every `logs` invocation downloads artifacts for all matching runs, consuming unnecessary GitHub API quota and network bandwidth. There is already a `--workflow` flag for positive selection, but no complement for negative selection.

### Decision

We will add a `--exclude` (StringSlice) flag to the `logs` CLI command and a corresponding `exclude_workflows` field to the MCP `logs` tool struct. Excluded workflows are resolved to their canonical display names via lock files and then filtered out of each batch of runs **before** `downloadRunArtifactsConcurrent` is called. This pre-download filtering ensures no API requests or download traffic are incurred for excluded workflows. Matching is case-insensitive to tolerate capitalization differences between user input and stored workflow names.

### Alternatives Considered

#### Alternative 1: Post-Download Filtering

Filter the downloaded run list after artifacts have already been fetched. This approach is simpler to implement because it does not require threading the exclude list through the batch loop. It was rejected because it defeats the stated goal of saving API requests and download bandwidth — the excluded artifacts would still be fetched before being discarded.

#### Alternative 2: Deny-List Configuration File

Store excluded workflows in a persistent configuration file (e.g., `.github/aw/logs-config.yml`) rather than as a command-line flag. This would allow users to set standing exclusions without repeating the flag on every invocation. It was not chosen for this PR because it introduces config file discovery, parsing, and merge-with-CLI-flag semantics that add scope beyond the immediate need. A config file could be added in a future iteration.

#### Alternative 3: Negate the Existing `--workflow` Positive Filter

Extend the existing `--workflow` flag with a negation prefix (e.g., `--workflow !ci-tests`). This was considered because it avoids adding a new flag. It was not chosen because mixing positive and negative selectors in a single flag is less ergonomic and more error-prone than two separate, clearly named flags (`--workflow` / `--exclude`).

### Consequences

#### Positive
- Excluded runs are skipped before any artifact download, preserving GitHub API quota and reducing network traffic proportionally to the number of excluded runs.
- Case-insensitive matching and lock-file resolution make the flag resilient to minor naming discrepancies between user input and stored display names.
- The MCP `logs` tool gains the same capability through a new `exclude_workflows` array field, keeping CLI and MCP interfaces in parity.

#### Negative
- `DownloadWorkflowLogs` gains yet another parameter, worsening an already long function signature. This may be a motivation for future refactoring of the signature into a struct.
- If lock files are absent or out of date, `resolveExcludeWorkflows` silently falls back to raw-value matching, which may produce unexpected results without a clear error to the user.
- The exclusion filter is applied per batch, not globally across the full pagination history, which means the effective run count returned may be less than `--count` when many runs are excluded in a batch.

#### Neutral
- All existing callers of `DownloadWorkflowLogs` must pass a new `nil` argument for `excludeWorkflows`, requiring updates across test files.
- Verbose mode (`--verbose`) now prints a summary of active exclude filters and individual skipped run IDs to stderr.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Exclusion Flag and Parameter

1. The `logs` CLI command **MUST** expose an `--exclude` flag of type `StringSlice` that accepts one or more workflow names or IDs, either comma-separated or via repeated flags.
2. The MCP `logs` tool **MUST** expose an `exclude_workflows` array field in its argument struct with a JSON schema description.
3. Implementations **MUST** pass the exclude list to `DownloadWorkflowLogs` as the `excludeWorkflows []string` parameter.

### Resolution of Workflow Names

1. Implementations **MUST** attempt to resolve each entry in the exclude list to a canonical display name via lock files before applying the filter.
2. Implementations **MUST** fall back to the raw value for case-insensitive matching when lock-file resolution fails.
3. Implementations **MUST NOT** abort or return an error when a workflow name cannot be resolved; the raw value **MUST** be used as the fallback.

### Pre-Download Filtering

1. Implementations **MUST** apply the exclusion filter to each batch of `WorkflowRun` objects **before** calling `downloadRunArtifactsConcurrent`.
2. Implementations **MUST NOT** download any artifacts or logs for runs whose `WorkflowName` matches an entry in the resolved exclude list.
3. Matching **MUST** be performed case-insensitively (i.e., `strings.ToLower` on both sides of the comparison).
4. Implementations **SHOULD** log each skipped run at the debug level, and **SHOULD** print each skipped run to stderr when verbose mode is active.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24930818842) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
