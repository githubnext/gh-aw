# ADR-43056: Download safe-outputs-items manifest in the conclusion job to restore SafeItemsCount

**Date**: 2026-07-03
**Status**: Draft
**Deciders**: Unknown

---

### Context

Since 2026-05-31, `SafeItemsCount=0` and `actuation_style=read_only` were observed on every workflow run, causing telemetry and reporting to under-count safe-output actions. The root cause was that the `safe_outputs` job uploads a `safe-output-items.jsonl` manifest as the `safe-outputs-items` artifact, but the conclusion job never downloaded that artifact before generating `activity/summary.json`. Because the manifest was absent from the conclusion job's working directory, the Go logs reader (`logs_usage_activity.go`) always saw zero items, masking genuine non-zero runs. The fix must work across all 264 lock YAML files that share the same conclusion-job structure.

### Decision

We will prepend an `actions/download-artifact` step (with `continue-on-error: true`) to every conclusion job in every lock file so the `safe-output-items.jsonl` manifest is present when `activity/summary.json` is generated. We will also add `parseSafeOutputsManifest()` to `generate_usage_activity_summary.cjs` so the manifest is parsed and its counts are included in the `safe_outputs` key of `activity/summary.json`. Finally, we will add a Go-layer backfill in `logs_usage_activity.go` that sets `SafeItemsCount` from `summary.SafeOutputs.TotalItems` when the field is zero and the summary indicates non-zero items, preserving genuine zero-item runs.

### Alternatives Considered

#### Alternative 1: Backfill SafeItemsCount at log-read time via the GitHub Artifacts API

The Go layer could call the GitHub Artifacts API at log-read time to fetch the `safe-outputs-items` artifact and count its entries, without modifying any workflow YAML. This avoids the 264-file lock-file diff and keeps the workflow definitions unchanged. It was not chosen because it would require an authenticated outbound API call from the CLI at read time, adding latency and a new network dependency that could fail silently in environments without credentials.

#### Alternative 2: Embed safe-output counts in the existing agent_output.json artifact

The `safe_outputs` job could write item counts directly into `agent_output.json` (which is already downloaded in the conclusion job) instead of using a separate `safe-output-items.jsonl` manifest. This would require no new download step and no lock-file changes. It was not chosen because changing the format of `agent_output.json` would require coordinated changes across the safeoutputs tool, the agent harness, and all consumers, creating a larger scope of change with higher risk of schema breakage.

### Consequences

#### Positive
- Restores accurate `SafeItemsCount` in telemetry from the date of deployment forward, fixing dashboards and reports that were silently under-counting safe-output actions.
- The `continue-on-error: true` flag ensures workflows with no safe outputs (where the artifact does not exist) continue to succeed without modification to their behavior.
- The Go-layer backfill differentiates between genuine zero-item runs and missing-manifest runs, avoiding false negatives in historical data.

#### Negative
- 264 lock YAML files are modified with an identical 7-line block, creating a large PR diff that is difficult to review and increases the risk of merge conflicts with concurrent lock-file changes.
- A new download step is added to every conclusion job, adding minor CI overhead (artifact download time) for all future workflow runs.

#### Neutral
- The `safe_outputs` key becomes a new required field in `activity/summary.json`; consumers that do not yet read this field are unaffected but must be updated to benefit from the data.
- The backfill only applies when `SafeItemsCount == 0` and `summary.SafeOutputs.TotalItems > 0`, so historical runs before the manifest upload was introduced remain correctly reported as zero.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
