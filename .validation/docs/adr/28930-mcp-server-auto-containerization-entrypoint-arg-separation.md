# ADR-28930: MCP Server Auto-Containerization — Entrypoint/Arg Separation Contract

**Date**: 2026-04-28
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The agentic workflow compiler auto-containerizes MCP servers whose `command` field matches a known package-manager binary (`npx`, `uvx`). When no explicit `container` is specified in the workflow frontmatter, `getMCPConfig()` selects an appropriate container image and rewrites the server config so that Docker can run it. A container launch requires two distinct pieces of configuration: the `entrypoint` (the executable the container starts with) and `entrypointArgs` (the arguments passed to that executable). An earlier implementation conflated these by prepending the command to `entrypointArgs` after also assigning it to `entrypoint`, causing Docker to receive `npx npx @sentry/mcp-server` instead of `npx @sentry/mcp-server` — which exposed zero tools and silently broke any workflow that relied on Sentry MCP.

### Decision

We will maintain a strict separation between the container `entrypoint` and `entrypointArgs` during auto-containerization: the `command` field is moved to `entrypoint` as-is, and the original `args` are assigned to `entrypointArgs` unchanged without any prepending of the command. Both `command` and `args` are then cleared on the result struct to prevent double-interpretation downstream. This contract makes the relationship between the source workflow config and the generated Docker invocation unambiguous and testable.

### Alternatives Considered

#### Alternative 1: Keep command as process command, not container entrypoint

Instead of setting a container `entrypoint`, leave the command as the process command and pass `[command, ...args]` as the arguments to the default container entrypoint (e.g., `sh -c`). This would avoid the entrypoint/arg split entirely. It was not chosen because the container images used (`node:lts-alpine`, `python:alpine`) do not default to a shell entrypoint suitable for running npm packages — setting `entrypoint: npx` directly is the intended usage pattern for these images.

#### Alternative 2: Require explicit container configuration for all MCP servers

Remove auto-containerization entirely and require workflow authors to specify `container`, `entrypoint`, and `entrypointArgs` explicitly when they want containerized stdio servers. This would eliminate the implicit rewriting that caused the bug. It was not chosen because auto-containerization is a significant developer-experience feature that reduces boilerplate for the common case of `npx`- and `uvx`-based MCP servers; the correct fix is to make the implicit behavior correct, not to remove it.

### Consequences

#### Positive
- Docker receives the correct invocation (`npx @sentry/mcp-server`) and all declared MCP tools are exposed to the agent.
- The auto-containerization contract is now covered by a regression test (`TestNpxCommandAutoContainerization`) that explicitly asserts the command is not duplicated in `entrypointArgs`.

#### Negative
- All compiled lock files that referenced servers with `command: npx` or `command: uvx` must be recompiled to remove the erroneously prepended command argument; this is a one-time migration cost.

#### Neutral
- The `command` and `args` fields are both cleared to empty/nil after the rewrite; downstream code must not assume they retain their original values after `getMCPConfig()` runs for an auto-containerized server.
- The `@sentry/mcp-server` dependency was bumped from `0.32.0` to `0.33.0` as part of this fix; the version change is incidental to the architectural contract.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### MCP Auto-Containerization Entrypoint Contract

1. When `getMCPConfig()` auto-assigns a container for a well-known command (e.g., `npx`, `uvx`), implementations **MUST** set `result.Entrypoint` to the value of `result.Command` and **MUST NOT** also include that command value as the first element of `result.EntrypointArgs`.
2. Implementations **MUST** set `result.EntrypointArgs` to `result.Args` directly (the original args from the workflow config), without prepending the command.
3. Implementations **MUST** set `result.Command` to the empty string after assigning it to `result.Entrypoint`, so that the command is not interpreted twice by downstream config serialization.
4. Implementations **MUST** set `result.Args` to `nil` after assigning it to `result.EntrypointArgs`, so that args are not serialized in both fields.
5. Implementations **SHOULD** include a regression test that asserts the first element of `entrypointArgs` in generated YAML is not the well-known command itself when that command is used as the `entrypoint`.

### Conformance Testing

1. Implementations **MUST** provide at least one test case per supported well-known command (`npx`, `uvx`) that parses a workflow with `command: <well-known>` and verifies the generated YAML `entrypointArgs` does not begin with the command string.
2. Test cases **SHOULD** cover both bare package arguments (e.g., `["@sentry/mcp-server@0.33.0"]`) and flag-prefixed arguments (e.g., `["-y", "@modelcontextprotocol/server-memory"]`).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Any implementation where the container entrypoint and the first element of `entrypointArgs` are identical (both equal to the well-known command) is non-conformant.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25058262146) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
