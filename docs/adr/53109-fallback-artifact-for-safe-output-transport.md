# ADR-53109: Fallback Artifact for Safe-Output Transport

**Date**: 2026-08-16
**Status**: Draft
**Deciders**: Unknown

---

### Context

Safe outputs produced by agent jobs are transported to downstream processing jobs via a GitHub Actions artifact named `agent`. The upload step for this artifact uses `continue-on-error: true` so that upload timeouts against blob storage do not fail the entire agent job. When blob storage is slow or unavailable, the upload times out, the artifact is never created, and downstream jobs silently skip safe-output processing — causing "no safe outputs" errors even when the agent successfully called safe-output tools. The fix must preserve the `continue-on-error` behavior (to avoid failing agent jobs on storage issues) while ensuring safe outputs are reliably delivered.

### Decision

We will introduce a dedicated, lightweight fallback artifact (`agent-output-fallback`) that carries only `agent_output.json` and `safeoutputs.jsonl` and is uploaded immediately before the full agent artifact. Downstream download steps switch from `name:` to a brace pattern (`{agent,agent-output-fallback}`) with `merge-multiple: true`, so they extract whichever artifact is available without changing the downstream file paths. This decouples safe-output transport from the larger, more failure-prone full agent artifact.

### Alternatives Considered

#### Alternative 1: Remove `continue-on-error: true` from the agent artifact upload

The agent artifact upload step could be made mandatory so that any blob storage failure fails the job visibly rather than silently. This eliminates the silent-failure mode but causes agent job failures whenever blob storage is slow, shifting the failure from safe-output processing to the job itself — which is a worse outcome for end users and harder to recover from without re-running the full agent job.

#### Alternative 2: Retry the agent artifact upload on failure

The upload step could be wrapped in a retry loop (e.g., using a retry action) to reduce the chance of a one-off timeout causing a permanent failure. This adds implementation complexity, significantly increases runtime when blob storage is degraded, and still fails silently after all retries are exhausted — it does not fundamentally change the failure mode.

### Consequences

#### Positive
- Safe outputs are reliably delivered to downstream processing jobs even when the large agent artifact upload fails or times out.
- Backward compatible: if both artifacts are missing (e.g., the agent job crashed entirely), the download step still fails gracefully and `GH_AW_AGENT_OUTPUT` stays unset, preserving existing behavior.

#### Negative
- Each workflow run now triggers two artifact upload steps, adding minor overhead in upload time and blob storage consumption per run.
- All 285 `.lock.yml` files must be regenerated whenever the upload or download template changes, making even small modifications to the artifact transport logic expensive to propagate.

#### Neutral
- The fallback artifact is intentionally named `agent-output-fallback` rather than reusing `agent` or `agent-output` to avoid conflicting with legacy CLI flattening logic that is keyed on the `agent-output` directory name.
- The `ArtifactDownloadConfig.FallbackArtifact` field and `buildArtifactDownloadSteps` update ensure the pattern is applied consistently across all four download sites (safe_outputs, detection, evals, conclusion) and the custom safe-jobs download.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
