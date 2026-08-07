# ADR-51153: Retain Firewall and MCP Observability Artifacts via Dedicated Upload Steps

**Date**: 2026-08-07
**Status**: Draft
**Deciders**: copilot-swe-agent, gh-aw maintainers

---

### Context

Each workflow run in this repository activates a firewall proxy and an MCP gateway, both of which generate runtime telemetry: the firewall produces `access.log` audit files and AWF reflection data; the MCP gateway emits JSONL tool-call records. Prior to this change, that evidence was either discarded at run end or folded into the general `agent` artifact without explicit retention guarantees. When post-run egress auditing or tool-call reconstruction was needed (e.g., to diagnose a security incident or debug a failing skill), the required files were often absent, making forensic analysis unreliable.

### Decision

We will add two dedicated `actions/upload-artifact` steps — `Upload MCP observability logs` and `Upload firewall observability logs` — to every generated workflow lock file. These steps run after the main agent step (`if: always()`), tolerate missing files (`if-no-files-found: ignore`), and produce distinct artifacts (`mcp-logs` and `firewall-audit-logs`) separate from the existing `agent` artifact. The compiler templates and wasm golden files are updated to generate these steps automatically for all future lock files.

### Alternatives Considered

#### Alternative 1: Ingest logs into an external observability platform (e.g., Datadog, Splunk, OpenTelemetry collector)

Streaming logs to an external service would provide richer query and alerting capabilities and would not depend on GitHub artifact retention periods. However, it requires standing up and maintaining an external service, adding secret management (API keys), network egress from every runner, and per-byte ingestion costs. The complexity and operational burden are disproportionate to the current need, which is simple post-run access to raw log files for manual review.

#### Alternative 2: Append firewall and MCP logs to the existing unified `agent` artifact

Folding these logs into the `agent` artifact is low-friction and requires no new artifact quota. However, it makes programmatic retrieval harder (callers must filter inside a mixed-purpose archive), and it conflates security-sensitive egress audit data with general agent outputs. Dedicated artifacts enable targeted retention policy overrides and access-control granularity in the future.

### Consequences

#### Positive
- Post-run egress auditing is reliable: firewall `access.log`, `audit/`, AWF reflect data, and AWF config are all retained as a named artifact.
- MCP tool-call reconstruction is possible after the run: JSONL telemetry is preserved in a separate `mcp-logs` artifact.
- Compiler-level codegen ensures new workflow lock files automatically include both upload steps, so the fix does not require per-workflow manual intervention.
- `continue-on-error: true` prevents a missing log directory from failing an otherwise successful run.

#### Negative
- Every workflow run now produces two additional artifact uploads, increasing GitHub Actions artifact storage consumption proportionally to the number of active lock files.
- Artifact retention is bounded by the repository's artifact-retention policy; very old runs still lose their observability data when artifacts expire.

#### Neutral
- The existing `agent` artifact is unchanged; consumers of that artifact are not affected.
- Compiler tests and wasm golden files must be kept in sync with the new upload step templates, adding a small maintenance surface for future template changes.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
