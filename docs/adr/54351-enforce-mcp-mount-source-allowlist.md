# ADR-54351: Enforce Allowlist Policy for MCP Container Mount Sources

**Date**: 2026-08-20
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Custom MCP server configurations support a `mounts` field that maps host paths into the container at launch time. Before this change, the system only verified that the `mounts` value followed the `source:destination:mode` syntax — no check was performed on whether the source path was safe. This allowed an imported workflow component to declare dangerous mounts such as `/:/host_root:rw` or `/var/run/docker.sock:/var/run/docker.sock:rw`, which would be passed verbatim to `docker run -v`, enabling container escape or privilege escalation against the runner host.

### Decision

We will enforce a strict allowlist policy for MCP container mount sources at three layers — JSON schema validation, parser-level parsing, and MCP gateway allowlist construction — rejecting any host source that is not:

- A path under `${GITHUB_WORKSPACE}` or `$GITHUB_WORKSPACE`
- A path under `${RUNNER_TEMP}/gh-aw` or `$RUNNER_TEMP/gh-aw`
- A path under `/tmp/gh-aw`
- A Docker named volume (alphanumeric, `_`, `.`, `-`; no path separators or shell variables)

Absolute paths, the filesystem root, the Docker socket path, and traversal sequences (`..`) are all rejected. The same `IsAllowedMCPMountSource` function (in `pkg/parser/mcp.go`) is used across all three enforcement layers to ensure a single source of truth for the policy.

### Alternatives Considered

#### Alternative 1: Runtime-only validation (reject unsafe mounts only at Docker invocation)

Validate mount sources only when the `docker run` command is about to be executed, rather than also at schema parse time and at the compiler layer. This would still block dangerous mounts from being launched, but would allow them to pass schema validation and parser-layer checks. It provides a single enforcement point rather than defense in depth, and gives authors later and less informative error feedback (at runtime rather than at config-parse time).

#### Alternative 2: Prohibit all user-configured host mounts for custom MCP servers

Remove the `mounts` field entirely from the custom MCP server configuration schema, disallowing any host-to-container path sharing. This eliminates the attack surface completely, but breaks legitimate use cases: containerized MCP tools commonly need read-only access to workspace files (source code, credentials, build artifacts) to function. The allowlist approach preserves this capability under a tightly scoped policy.

### Consequences

#### Positive
- Eliminates the container escape / host privilege escalation risk introduced by unvalidated custom MCP mount sources.
- Defense in depth: the allowlist is enforced at schema validation, parser, and MCP gateway allowlist construction, so a bypass at one layer cannot silently propagate.
- Early, actionable error messages: mount rejections surface at config-parse time with a clear explanation of which sources are permitted.

#### Negative
- Breaking change for any existing workflow that uses arbitrary absolute host paths (e.g., `/etc`, `/usr`, `/opt`) in custom MCP server mounts. Those mounts will fail validation after this change.
- The allowlist prefixes are hardcoded. Accommodating new legitimate runner paths (e.g., a new temp directory) requires a code change and re-deployment rather than a configuration update.

#### Neutral
- The `IsAllowedMCPMountSource` function is exported from `pkg/parser` and imported by `pkg/workflow`, creating a new cross-package dependency that couples the parser and workflow layers on the mount policy definition.
- Test fixtures for `mcp_config_validation_test.go` and `mcp_gateway_mount_policy_test.go` were updated to use allowlisted sources, reflecting the stricter policy in test coverage.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
