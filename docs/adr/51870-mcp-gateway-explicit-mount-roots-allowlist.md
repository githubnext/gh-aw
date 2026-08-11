# ADR-51870: Compiler-Computed MCP Gateway Mount-Roots Allowlist

**Date**: 2026-08-11
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`gh-aw-mcpg` v0.4.9 introduced a trusted host-path mount policy that, by default, only permits read-only access to `$GITHUB_WORKSPACE` and read-write access to the system temp dir. The gh-aw compiler mounts `$GITHUB_WORKSPACE` read-write into the built-in `safeoutputs` MCP server container, which caused the gateway to reject that mount at startup with `mount policy violation: read-write access to this host root is not permitted by mount policy`. This broke safeoutputs tool registration for every workflow run. Beyond the workspace, the compiler configures mounts for several other backend MCP surfaces (the gh-aw runtime directory, per-server `mounts` arrays, `-v`/`--volume` flags embedded in server `args`/`entrypointArgs`), all of which could trigger the same policy rejection if not explicitly permitted.

### Decision

We will introduce `buildMCPGatewayAllowedMountRoots` in `pkg/workflow/mcp_setup_gateway.go`, a compiler function that computes a complete `MCP_GATEWAY_ALLOWED_MOUNT_ROOTS` value from all mount surfaces the compiler configures: the built-in workspace, gh-aw runtime, safeoutputs, and temp paths; gateway-level `sandbox.mcp.mounts`; and every per-server `mounts` field or `-v`/`--volume` arg across all configured MCP tools. The compiler exports this value before gateway startup and forwards it to the gateway container via `-e MCP_GATEWAY_ALLOWED_MOUNT_ROOTS`, giving the gateway a precise, per-workflow policy that reflects what the compiler actually mounts. All compiled `.lock.yml` workflows are regenerated to carry this new export.

### Alternatives Considered

#### Alternative 1: Patch the Default Policy in `gh-aw-mcpg`

Make `gh-aw-mcpg` aware of the workspace-mount convention so that its default policy permits read-write access to `$GITHUB_WORKSPACE` without requiring an explicit env var from the compiler.

Not chosen because it embeds gh-aw-specific knowledge into the gateway binary, coupling two independently versioned components. Any future change to the compiler's mount conventions (new built-in servers, different paths) would require a coordinated gateway release rather than a compiler-only change. The explicit env-var contract is the right interface boundary between the two components.

#### Alternative 2: Disable the Mount Policy Check via a Wildcard or Opt-Out Flag

Pass a wildcard value (e.g., `*:rw`) or a dedicated opt-out flag to tell the gateway to skip mount-policy enforcement entirely for gh-aw workflows.

Not chosen because it defeats the security purpose of the policy. The gateway's mount policy prevents backend MCP server containers from being given unintended filesystem access. Bypassing it globally would silently allow any user-supplied mount (including misconfigured or malicious custom MCP servers) to access arbitrary host paths without the gateway's oversight.

### Consequences

#### Positive
- The gateway policy violation is resolved: all built-in MCP servers (safeoutputs, agentic-workflows, compiler-configured custom servers) register successfully with `gh-aw-mcpg` v0.4.9+.
- The compiler is now the single authoritative source for the gateway's mount policy. Future additions of new mount surfaces (new built-in servers, new compiler directives) automatically appear in `MCP_GATEWAY_ALLOWED_MOUNT_ROOTS` without any gateway-side change.
- The allowlist is minimal and precise: only paths the compiler actually mounts are permitted, preserving the gateway's security model for user-configured servers.

#### Negative
- Every compiled `.lock.yml` workflow must be regenerated whenever the allowlist logic changes, producing large diffs across all 290+ lock files even for small compiler changes.
- The compiler now carries mount-policy logic that could grant overly broad access if `buildMCPGatewayAllowedMountRoots` is incorrect (e.g., a bug in `parseMCPGatewayAllowlistMount` or `collectMCPServerConfiguredMounts` that adds unintended paths).

#### Neutral
- The `MCP_GATEWAY_ALLOWED_MOUNT_ROOTS` env var becomes a required interface contract between the compiler and `gh-aw-mcpg` v0.4.9+; older gateway versions that ignore unknown env vars remain unaffected.
- Unit tests for `buildMCPGatewayAllowedMountRoots` (dedup/merge behavior, determinism, mode precedence) are added alongside the implementation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
