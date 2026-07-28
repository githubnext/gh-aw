# ADR-48700: Dedicated MCP Logs Artifact Upload for Telemetry Observability

**Date**: 2026-07-28
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

MCP telemetry files (`gateway.jsonl`, `rpc-messages.jsonl`) were missing from approximately 5 out of 8 sampled agentic workflow runs. Two root causes were identified: (1) the `/tmp/gh-aw/mcp-logs/` directory lacked a `chmod` step before upload — unlike the firewall logs path which already had one — causing permission-denied failures in upload when gateway containers wrote files under a different UID (e.g. AWF isolation mode or tool sub-containers); (2) MCP logs were only captured via the unified `agent` artifact, which is a single upload aggregating many file types and is therefore a single point of failure for telemetry delivery. The gap was reported in gh-aw#48674 and means MCP tool-call observability is unreliable for production debugging and analysis.

### Decision

We will add two compiler-generated steps to every agentic workflow post-agent collection sequence: (1) a best-effort `chmod -R a+rX /tmp/gh-aw/mcp-logs/` step to normalize file permissions before upload, mirroring the existing pattern used for firewall logs; and (2) a dedicated `actions/upload-artifact` step that produces a `mcp-logs-{sanitized-workflow-name}` artifact independently of the unified agent artifact. The dedicated artifact uses `if: always()`, `continue-on-error: true`, and `if-no-files-found: ignore` to be resilient to gateway absence, and the `ArtifactSetMCP` download set is updated to include this new artifact alongside the existing `agent` artifact.

### Alternatives Considered

#### Alternative 1: Fix the unified agent artifact upload to reliably include MCP logs

Enhance the existing unified artifact upload step to also `chmod` the MCP logs directory and ensure the glob patterns always capture MCP log files. This avoids adding a second upload step per workflow and keeps artifact count constant.

This was not chosen because the unified artifact is already a complex aggregation of many file types and failure in any path can silently drop the MCP logs; adding chmod to that step would not decouple the upload. The single point of failure problem would remain — if the unified upload step encounters an error or is skipped due to a previous step failure, MCP logs are still lost.

#### Alternative 2: Add a chmod-only fix to the existing unified upload path

Add the `chmod -R a+rX /tmp/gh-aw/mcp-logs/` step before the unified upload, without creating a dedicated artifact. This addresses the permissions root cause while avoiding storage overhead.

This was not chosen because it only fixes the permission root cause, not the single-point-of-failure root cause. Logs would still be lost whenever the unified artifact upload itself fails, is incomplete, or is slow to process, which accounts for the other failure modes observed in the sampled runs.

#### Alternative 3: Use a separate CI job for MCP log collection

Collect MCP logs in a separate downstream job that fetches and re-uploads them after the agentic job completes.

This was not chosen because it introduces job-level dependency complexity, adds latency before logs are available, and requires the main job to expose the logs directory through a job output or interim artifact anyway — net more infrastructure for equivalent reliability.

### Consequences

#### Positive
- MCP telemetry (`gateway.jsonl`, `rpc-messages.jsonl`) is now captured per-run as a dedicated artifact, independent of the unified agent artifact, eliminating the observed ~62% loss rate.
- The `ArtifactSetMCP` download command now fetches both sources, giving consumers a complete view of MCP traffic even when the unified artifact is partially complete.
- The chmod step uses the same `a+rX` pattern already validated for firewall logs, reducing the number of distinct permission-fix patterns in the codebase.
- The new `generateMCPLogsArtifactUpload` function is unit-tested with step-order assertions covering both Playwright and non-Playwright workflow shapes.

#### Negative
- Every agentic workflow now produces an additional artifact (`mcp-logs-{name}`), increasing artifact storage consumption and the number of upload API calls per run. For the 264 currently compiled workflows, this adds 264 uploads per full batch run.
- Workflows where the MCP gateway never started will still trigger the upload step (it will be a no-op due to `if-no-files-found: ignore`), adding a small fixed overhead per job.

#### Neutral
- The `TmpMcpLogsDirExpr` constant is introduced to handle ARC/DinD topologies where `/tmp` is not daemon-visible; the path resolves to `${{ runner.temp }}/gh-aw/mcp-logs/` in those environments, matching the rewrite already applied for other log types.
- The step ordering `Stop MCP Gateway → Fix MCP logs permissions → Upload MCP logs → Upload agent artifacts` is now enforced by the step-order tracker and tested explicitly.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
