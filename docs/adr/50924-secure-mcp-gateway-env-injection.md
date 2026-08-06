# ADR-50924: Secure MCP Gateway Custom Environment Injection via Compiler-Controlled Transport Variables

**Date**: 2026-08-06
**Status**: Draft
**Deciders**: pelikhan

---

### Context

The MCP gateway allows workflow authors to supply custom environment variables under `sandbox.mcp.env`. Before this change, these values were interpolated directly into two unsafe positions: (1) the GitHub Actions `run:` shell script as `export NAME=VALUE` statements, and (2) the serialized Docker command string as `-e NAME` flags (with values forwarded from the shell environment). This approach is exploitable: values containing shell metacharacters (`;`, backticks, newlines) can inject arbitrary commands into the shell script before the gateway process starts, and a value assigned to `BASH_ENV` would be sourced by Bash on the runner host before the run script executes. These issues are tracked as GHSA-j77w-g4jj-hp99.

### Decision

We will route `sandbox.mcp.env` values through compiler-controlled transport environment variables (`GH_AW_MCP_GATEWAY_ENV_0`, `GH_AW_MCP_GATEWAY_ENV_1`, …) and a companion manifest variable (`GH_AW_MCP_GATEWAY_CUSTOM_ENV_NAMES`) instead of embedding them in the shell script or Docker command string. The Go compiler writes the values under indexed names that have no special meaning to Bash; a marker token (`__GH_AW_MCP_GATEWAY_CUSTOM_ENV__`) is placed in the Docker command string and replaced at runtime by the JavaScript launcher (`start_mcp_gateway.cjs`) with atomic `-e NAME=VALUE` arguments constructed from the transport variables. Env var names are validated at compile time (Go regex) and at runtime (JS regex) against `^[A-Z_][A-Z0-9_]*$`.

### Alternatives Considered

#### Alternative 1: Shell-escape values before embedding in the run script

Escape all shell-special characters in `sandbox.mcp.env` values before interpolating them into `export NAME=VALUE` lines in the generated `run:` block. This is the simplest approach and requires no protocol changes.

Not chosen because: shell escaping is notoriously difficult to get right across all Bash versions and special variables (e.g., `BASH_ENV` is sourced before the script body runs, so even a correctly-escaped assignment can still be exploited if the name itself is dangerous). A missed edge case would reintroduce the vulnerability silently.

#### Alternative 2: Quote env values in the Docker command string

Embed values as `"NAME=VALUE"` tokens in the serialized `MCP_GATEWAY_DOCKER_COMMAND` string, relying on the existing shell-split regex to pass them as quoted arguments to `spawn()`.

Not chosen because: the serialized command is a string that is split by regex in JS; adding values there keeps runtime secrets inside a compiled YAML string and makes the split logic responsible for correctly handling arbitrary byte sequences including quotes, spaces, and newlines. A value containing `"` or a newline would break the split or the quoting, and the serialized form would capture secret values in CI logs.

### Consequences

#### Positive
- Shell metacharacters (`;`, backticks, newlines, `BASH_ENV`) in custom env values can no longer inject commands into the runner shell or be sourced by Bash before the run script executes, because values never appear in the `run:` block.
- Custom values are never embedded in the compiled Docker command string, so they do not appear in workflow lock files, CI logs, or debug output of the gateway command.
- Defense-in-depth: env var name validation occurs at both compile time (Go) and at gateway launch time (JS), so a malformed name is rejected before it can reach the container.

#### Negative
- Increased complexity: the transport is a two-layer protocol (Go compiler writes indexed transport vars; JS launcher reads the manifest, validates names, and reconstructs atomic `-e` arguments) that must be kept in sync across two languages.
- The marker token `__GH_AW_MCP_GATEWAY_CUSTOM_ENV__` is a non-obvious in-band signal that appears in the Docker command string and must remain undocumented to avoid forgery; any refactor that removes or renames it silently breaks custom env injection.

#### Neutral
- The number of GitHub Actions step environment variables increases by `N+1` for a workflow with `N` custom MCP env vars.
- Workflows that previously relied on the `export NAME=VALUE` path in the run script to make custom env vars available to shell commands other than the gateway itself will no longer have those exports — those workflows would have needed explicit `export` steps anyway.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
