# ADR-29351: Dedicate a Separate Artifact for Firewall Audit Logs

**Date**: 2026-04-30
**Status**: Draft
**Deciders**: pelikhan (copilot-swe-agent)

---

## Part 1 — Narrative (Human-Friendly)

### Context

Agentic workflows run inside a sandboxed environment with a network firewall. When a blocked request occurs, the firewall writes audit records (`policy-manifest.json`, `audit.jsonl`) and proxy logs to `/tmp/gh-aw/sandbox/firewall/audit/` and `/tmp/gh-aw/sandbox/firewall/logs/`. Previously these paths were included in the paths list of the generic `agent` artifact uploaded at the end of every workflow run. The `actions/upload-artifact` action strips common path prefixes during upload, which flattened the directory structure and made the files indistinguishable from other agent outputs. As a result, `detectFirewallAuditArtifacts` could not reliably locate the audit files, causing 362 out of 372 blocked firewall requests to be reported as `(unknown)` with no policy rule attribution.

### Decision

We will upload firewall audit logs under a dedicated `firewall-audit-logs` artifact name in a separate upload step (`generateSquidLogsUploadStep`), and remove the corresponding paths from the generic `agent` artifact. We chose this approach because an isolated artifact preserves the subdirectory layout produced by `actions/upload-artifact` path-stripping in a predictable and recoverable way: the CLI can look for `firewall-audit-logs/audit/` first, fall back to `firewall-audit-logs/` for flat placements, and finally fall back to the legacy `sandbox/firewall/audit/` path from the old merged layout. Only `awf-config.json` remains in the `agent` artifact for post-run config inspection.

### Alternatives Considered

#### Alternative 1: Fix path detection inside the merged `agent` artifact

Enhance `detectFirewallAuditArtifacts` to enumerate all possible flattened paths within the `agent` artifact and reconstruct the correct audit file location heuristically. This was rejected because the flattening behaviour of `actions/upload-artifact` strips an unpredictable common prefix, making reliable path reconstruction fragile and difficult to test. Keeping the logic inside the `agent` artifact also tightly couples audit-log discovery to the artifact packaging format, which changes as new files are added.

#### Alternative 2: Preserve subdirectory structure inside the `agent` artifact using a synthetic wrapper path

Upload all firewall files under a predictable sub-path within the `agent` artifact (e.g., `firewall/`) to prevent flattening from destroying the layout. This was rejected because it requires coordinated changes to both the upload paths and every downstream consumer, and it still leaves firewall audit data mixed with unrelated agent outputs, making selective download and size-bounded downloads harder for debugging tools.

### Consequences

#### Positive
- Policy attribution now resolves correctly for all firewall-blocked requests: the dedicated artifact path is stable, predictable, and unambiguous.
- Firewall audit data can be downloaded independently of the full `agent` artifact, reducing bandwidth for tooling that only needs policy analysis.
- Explicit artifact naming (`firewall-audit-logs`) improves discoverability when browsing workflow run artifacts in the GitHub UI.

#### Negative
- Every firewall-enabled workflow run now performs an additional artifact-upload step, adding marginal latency and storage overhead per run.
- Consumers that previously downloaded the `agent` artifact for firewall audit access must now also download the `firewall-audit-logs` artifact; any tooling or scripts that assume all post-run data lives in `agent` will silently miss firewall data until updated.

#### Neutral
- 205 workflow lock files were regenerated to propagate the new upload step; this is a mechanical recompilation with no semantic change beyond the added step.
- `hasFirewallArtifact` was updated to also match `FirewallAuditArtifactName` so that policy analysis triggers even when only the dedicated artifact is downloaded — a necessary consistency fix that comes naturally with the separation.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Artifact Separation

1. Firewall audit logs (`/tmp/gh-aw/sandbox/firewall/audit/`) and firewall proxy logs (`/tmp/gh-aw/sandbox/firewall/logs/`) **MUST** be uploaded under the artifact name `firewall-audit-logs` and **MUST NOT** be included in the `agent` artifact paths.
2. The `firewall-audit-logs` upload step **MUST** run with `if: always()` and **MUST** set `continue-on-error: true` so that a missing firewall directory does not fail the workflow.
3. The `agent` artifact **MUST NOT** include `AWFProxyLogsDir` or `AWFAuditDir` in its path list.

### Audit Log Discovery (`detectFirewallAuditArtifacts`)

1. Implementations **MUST** check for audit files under the `firewall-audit-logs/audit/` subdirectory as the primary search path, since this is the layout produced by `actions/upload-artifact` stripping the `/tmp/gh-aw/sandbox/firewall/` common prefix.
2. Implementations **SHOULD** fall back to `firewall-audit-logs/` (flat placement) if the subdirectory layout is absent, to handle edge cases in artifact download behaviour.
3. Implementations **MAY** additionally fall back to the legacy `sandbox/firewall/audit/` path to support existing downloaded artifacts from before this ADR was applied.
4. Implementations **MUST NOT** attempt to discover firewall audit files inside the `agent` artifact after this ADR is accepted.

### Firewall Artifact Detection (`hasFirewallArtifact`)

1. Implementations **MUST** consider `FirewallAuditArtifactName` (`firewall-audit-logs`) as a valid match when determining whether firewall policy analysis should run.
2. Implementations **SHOULD** also continue to match the legacy artifact names to support analysis of older workflow runs.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25179263531) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
