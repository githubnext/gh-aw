# ADR-48099: Forward GH_AW_INPUT_* Vars to MCP Gateway Container

**Date**: 2026-07-26
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Since v0.80.0, the safe-outputs MCP server runs inside a Docker container with a filtered `-e` allowlist. When a workflow author uses a dynamic safe-outputs field — for example `base-branch: ${{ inputs.base_branch }}` — the compiler writes a `${GH_AW_INPUT_BASE_BRANCH}` shell-style placeholder into the container's `config.json`. However, because `GH_AW_INPUT_*` variables were never included in the docker run `-e` allowlist, and were not emitted in the "Start MCP Gateway" step `env:` block, those placeholders remained unresolved at runtime inside the container. This caused `create_pull_request` (and other safe-outputs handlers) to fail with the cryptic error "No remote refs available for merge-base calculation" — with no indication that an unresolved placeholder was the root cause.

### Decision

We will extend the workflow compiler to extract all `GH_AW_INPUT_*` environment variable names referenced in the safe-outputs config, emit them in the "Start MCP Gateway" step `env:` block (so the runner process holds the resolved values), and append a `-e GH_AW_INPUT_*` flag for each one in the `docker run` command (so the container inherits those values). Additionally, we will add a `collectUnresolvedInputPlaceholders()` diagnostic in the MCP server's config loader to surface any remaining unresolved placeholders with a clear error message instead of a silent runtime failure.

### Alternatives Considered

#### Alternative 1: Pass All Runner Environment Variables to the Container

Instead of selectively forwarding only `GH_AW_INPUT_*` vars, forward the entire runner environment into the container via `--env-file` or `--env-host`. This would have resolved the issue without any compiler changes. It was rejected because the filtered `-e` allowlist exists specifically to limit what the container can observe — passing the full runner env would expose secrets, tokens, and other runner-level vars to the containerized MCP server, defeating the security boundary that the allowlist enforces.

#### Alternative 2: Resolve Placeholders at Compile Time

Substitute `${{ inputs.* }}` values directly into `config.json` at compile time so no placeholder survives to runtime. This was rejected because workflow input values are not known at compile time — they are resolved by GitHub Actions at workflow-dispatch time. Embedding them statically would require a separate "late-compilation" step at workflow startup, and would force config.json to be regenerated per-run, complicating the current architecture where `config.json` is written once at compile time as a static artifact.

#### Alternative 3: Redesign Config Using a Runtime-Resolved Mechanism

Replace the `${GH_AW_INPUT_*}` placeholder pattern in `config.json` with a runtime mechanism (e.g., the container reads vars directly from a mounted secrets file or a well-known env var namespace). This would eliminate the coupling between safe-outputs config parsing and MCP gateway code generation. It was rejected because it would require a significant redesign of the safe-outputs config system and a coordinated breaking change to the MCP server's config loader — too high a cost for a targeted bug fix.

### Consequences

#### Positive
- Dynamic safe-outputs fields like `base-branch: ${{ inputs.base_branch }}` now work correctly in the containerized MCP server for all safe-outputs handlers, not just `create-pull-request`.
- Unresolved placeholder errors now surface with an explicit, actionable error message (`ERR_CONFIG: Unresolved workflow input placeholder(s): ...`) rather than a cryptic merge-base failure.
- The fix is compiler-driven and backward-compatible: existing workflows without dynamic safe-outputs fields are unaffected (the new code paths only activate when `GH_AW_INPUT_*` vars are detected in the safe-outputs config).

#### Negative
- Each `${{ inputs.* }}` reference in a safe-outputs config now results in one additional env var forwarded to the MCP gateway container, incrementally widening the container's visible environment beyond what the original allowlist intended.
- The compiler now couples safe-outputs config parsing (`extractSafeOutputsInputEnvVars`) to MCP gateway step generation (`generateMCPGatewaySetup`), increasing the surface area that must be updated if either subsystem changes independently.

#### Neutral
- Two new test cases validate the fix: an update to the existing dynamic-allowed-repos test (asserts `GH_AW_INPUT_*` appears in both step env blocks and the docker run command) and a new regression test `TestSafeOutputsDynamicBaseBranchPassedToMCPContainer` covering the exact failure scenario.
- The `appendMCPGatewaySafeOutputsInputEnvFlags` function is ordered after `appendMCPGatewayConditionalEnvFlags` in the docker run command assembly, which means `GH_AW_INPUT_*` flags appear after standard conditional flags — this ordering has no functional impact but may affect readability of generated workflow YAML.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
