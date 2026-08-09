# ADR-51465: Persist Agent Runtime in aw_info.json and Add --runtime Filter to logs/audit

**Date**: 2026-08-08
**Status**: Accepted
**Deciders**: pelikhan, app/copilot-swe-agent

---

### Context

The `sandbox.agent.runtime` field (e.g., `gvisor`, `docker-sbx`) identifies the container isolation technology used for each agent run. This value is known at compile time but was never written to `aw_info.json` or any other post-compile artifact. As a result, operators using `gh aw logs` or `gh aw audit` have no way to filter or audit runs by sandbox runtime after the fact. This gap matters because different runtimes have different security properties, and incident investigation or compliance queries frequently need to scope results to a specific runtime.

### Decision

We will emit `sandbox.agent.runtime` as `GH_AW_INFO_AGENT_RUNTIME` during the `generate_aw_info` compilation step, write it as `agent_runtime` in `aw_info.json`, and expose it as a `--runtime` filter flag on both the `logs` and `audit` CLI commands and their MCP tool counterparts. This follows the existing pattern for `--engine` filtering and keeps `aw_info.json` as the single compiled source of truth for per-run metadata, making the MCP and CLI surfaces symmetrical.

### Alternatives Considered

#### Alternative 1: Derive Runtime at Query Time from Workflow Metadata

Parse the sandbox runtime from GitHub Actions workflow YAML or GitHub API metadata at query time rather than baking it into `aw_info.json` at compile time.

This was rejected because it would require parsing workflow files on every `logs`/`audit` invocation, adding latency and coupling the query path to workflow file structure. It also breaks the established pattern where `aw_info.json` is the canonical, pre-parsed metadata store—adding a second path for runtime info would create inconsistency.

#### Alternative 2: Store Runtime in a Separate Sidecar File

Write runtime info to a dedicated `runtime.json` sidecar artifact rather than extending the existing `aw_info.json` schema.

This was rejected because it adds file proliferation and forces the CLI to read multiple files to build a complete metadata picture. The existing convention is that `aw_info.json` is the single per-run metadata file; extending it is the lower-friction path.

### Consequences

#### Positive
- Operators can now filter `gh aw logs` and `gh aw audit` by sandbox runtime using `--runtime <value>`, enabling runtime-scoped auditing and incident investigation.
- MCP and CLI surfaces stay in sync: the `runtime` parameter on MCP tools maps directly to `--runtime` on the CLI, following the same forwarding pattern as other filter flags.
- The implementation mirrors the existing `--engine` filter, minimising new abstractions.

#### Negative
- All compiled `.lock.yml` workflow files must be regenerated whenever `aw_info.json` schema changes; this PR regenerates 284 such files, which is a large blast radius for a one-line schema addition.
- The `aw_info.json` schema now has one more field to maintain. Each future addition will similarly require bulk workflow regeneration, growing the maintenance surface over time.

#### Neutral
- The `AwInfo` Go struct gains an `AgentRuntime` field, which must be kept in sync with the compiler-side `GH_AW_INFO_AGENT_RUNTIME` environment variable and the `generate_aw_info.cjs` reader.
- Workflows that do not specify a sandbox runtime will emit an empty string for `GH_AW_INFO_AGENT_RUNTIME`, which is handled gracefully by the filter (empty value matches no `--runtime` query).

---

*ADR created by [adr-writer agent]. Finalized after addressing PR #51465 review feedback (audit runtime validation and shared filter reuse).*
