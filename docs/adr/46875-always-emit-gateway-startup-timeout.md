# ADR-46875: Always Emit `gateway.startupTimeout` in MCP Gateway Config

**Date**: 2026-07-20
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The MCP Gateway binary has a built-in startup timeout of 30 seconds. When a `safeoutputs` backend (or any stdio-based MCP backend) takes longer than 30 seconds to become ready — common on busy GitHub Actions runners — the gateway permanently records the backend as having zero tools. The backend may later become healthy and register all its tools, but the gateway's tool cache is never refreshed. The workflow then completes with an empty `agent_output.json` and reports success, with no safe output emitted and no error surfaced to the caller. This was observed in production (dotnet/android run #29618719353).

gh-aw documents a 120-second default for `tools.startup-timeout`, but the MCP gateway JSON configuration emitted by `RenderJSONMCPConfig` did not include the `gateway.startupTimeout` field. As a result, the gateway always applied its own 30-second default instead of the intended 120-second value, regardless of what gh-aw configured.

### Decision

We will always emit `gateway.startupTimeout` in the `gateway` section of the MCP gateway JSON configuration. The value is taken from `tools.startup-timeout` if it is a literal integer; otherwise it falls back to `constants.DefaultMCPStartupTimeout` (120 seconds). Because the default is always 120, the field is now present in every compiled workflow, preventing the gateway from applying its 30-second built-in default.

### Alternatives Considered

#### Alternative 1: Fix the MCP Gateway to retry tool discovery for timed-out backends

The MCP Gateway binary could be patched to detect when a previously-timed-out backend later becomes healthy and re-fetch its tool list. This would make the system self-healing regardless of what timeout value is configured.

This was not chosen because: it requires a change to the gateway binary (owned separately from gh-aw), would take longer to ship, and would not help users running older gateway versions. The immediate fix — propagating the existing 120-second gh-aw default — can be deployed in a single gh-aw release and prevents the timeout from being hit in the first place on typical runners.

#### Alternative 2: Increase the MCP Gateway's built-in default timeout in the gateway binary

The gateway binary's default could be raised from 30 seconds to 120 seconds at the gateway level, so gh-aw would not need to emit the field at all.

This was not chosen because: it requires a change to the gateway binary, it would ignore per-workflow `tools.startup-timeout` overrides configured by users, and it would not propagate user-specified timeouts that differ from 120 seconds. Emitting the field from gh-aw is the correct layer of responsibility since gh-aw owns the compiled workflow configuration.

### Consequences

#### Positive
- Silent workflow failures caused by slow-starting safeoutputs backends are prevented on typical runners.
- The `gateway.startupTimeout` field is now always explicit in compiled lock files, making the effective timeout visible and auditable without reading gateway source code.
- Per-workflow `tools.startup-timeout` values are correctly respected end-to-end.
- No gateway binary change required; ships entirely in a gh-aw compiler release.

#### Negative
- All 260+ existing compiled workflow `.lock.yml` files had to be recompiled to include the new field, adding churn to the lock file history.
- Every compiled workflow now unconditionally includes `startupTimeout: 120` even when the user has not customized the value, slightly increasing lock file verbosity.

#### Neutral
- A new `StartupTimeout int` field is added to `MCPGatewayRuntimeConfig`; any code that serializes or deep-copies this struct will now include the field.
- Unit tests for `buildMCPGatewayConfig` and `RenderJSONMCPConfig` were extended with cases for the new field; golden test files were updated.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
