# ADR-49082: Cache Shared Agentic Logs for Scheduled Workflows

**Date**: 2026-07-30
**Status**: Draft
**Deciders**: Unknown

---

### Context

Scheduled log and audit workflows (e.g., `agentic-token-audit`, `daily-security-observability`, `safe-output-health`) each independently call `gh aw logs` or `gh aw audit`, triggering repeated downloads of the same `activation` and `usage` artifacts from the GitHub Actions API. These redundant transfers cause unnecessary API rate-limit pressure, increased network latency, and occasional timeouts for workflows that share the same 30-day data window. The `gh aw logs` downloader also needs a reliable way to distinguish a partial artifact download from a complete one when reusing cached data across runs; previously, the presence of a non-empty directory was used as a proxy for completeness, which was unreliable.

### Decision

We will introduce a single shared GitHub Actions cache (key: `agentic-logs`, path: `.github/aw/logs`) refreshed daily by a dedicated `agentic-logs-cache.yml` workflow. Consumer scheduled workflows restore this cache read-only before their `gh aw logs` / `gh aw audit` invocations. The Go downloader gains a `.downloaded-artifacts/` marker directory inside each run folder to record which artifact sets were successfully downloaded; cache-hit logic for unfiltered audit requests now requires a complete-download marker (`all`) rather than a non-empty directory. Consumer workflows that contain a custom checkout step restore the cache after that checkout to prevent cleanup from deleting cached data.

### Alternatives Considered

#### Alternative 1: Per-workflow independent download (status quo)

Each scheduled workflow independently downloads the artifacts it needs from the GitHub API on every run. Simple and self-contained, but causes N redundant downloads per day (one per consuming workflow) for the same 30-day log window, contributing to rate-limit pressure and timeouts that motivated this change.

#### Alternative 2: GitHub Actions artifact-based sharing

Publish the pre-downloaded log directory as a named Actions artifact that consumer workflows download via `actions/download-artifact`. Artifacts are more visible in the Actions UI than cache entries. However, artifact retention defaults to 30–90 days with no built-in TTL enforcement at the key level; cache entries can be scoped to a ref and deleted/recreated atomically (as this PR does). Artifact-based sharing also doesn't integrate with the existing `actions/cache/restore` step pattern already used elsewhere in these workflows.

### Consequences

#### Positive
- Eliminates redundant per-workflow artifact downloads: all consumers share a single 30-day log snapshot refreshed once daily.
- Adds a reliable artifact-presence marker system (`markArtifactDownloaded`) in the Go downloader, fixing the existing fragility where a non-empty directory was treated as a complete download.
- The `shouldDownloadWorkflowRunLogs` predicate generalises the previous `isUsageOnlyArtifactFilter` check, correctly skipping heavyweight workflow-run log downloads for any activation/usage-only filter, not just the single-artifact case.

#### Negative
- Adds a new daily maintenance workflow (`agentic-logs-cache.yml`) that must succeed for consumers to benefit; if it fails, consumers fall back to live downloads (via `continue-on-error: true`), but the operational surface area increases.
- Cache entries are scoped to `GITHUB_REF`; feature branches will not have a warm cache and will always fall through to live downloads.
- The marker-based completeness check (`ArtifactSetAll` marker required for unfiltered requests) is a breaking change in cache-hit semantics: existing run directories lacking the marker will be re-downloaded on next access.

#### Neutral
- Consumer workflows require a `actions/cache/restore` step injected by the compiler; the compiler (`compiler_logs_cache.go`) detects eligibility by checking for `schedule:` in `on:` and `gh aw logs` / `gh aw audit` in workflow content.
- The shared cache path (`.github/aw/logs`) is a repository-relative directory, which means checkout cleanup will delete it unless the restore step follows the custom checkout — the compiler handles this ordering automatically.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
