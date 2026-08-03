# ADR-0001: Conditional OIDC Environment Variable Forwarding to MCP Gateway Container

**Date**: 2026-04-11
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw compiler generates a runner-owned `Start MCP Gateway` step that launches `mcpg` via `docker run`. The gateway is not launched by AWF. For HTTP MCP servers using GitHub OIDC (`auth.type: "github-oidc"`), the gateway needs runtime access to the Actions OIDC request variables (`ACTIONS_ID_TOKEN_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN`) so it can mint audience-bound JWTs per request.

This credential path is runner → gateway, not runner → AWF agent → gateway. AWF must not receive or relay these values.

### Decision

We detect HTTP MCP GitHub OIDC auth at compile time and conditionally append:

- `-e ACTIONS_ID_TOKEN_REQUEST_URL`
- `-e ACTIONS_ID_TOKEN_REQUEST_TOKEN`

to the runner-owned MCP gateway `docker run` command only when required.

In parallel, we harden the AWF boundary by excluding both variables from the AWF agent container environment (via `--exclude-env`) when HTTP MCP GitHub OIDC is configured.

### Alternatives Considered

#### Alternative 1: Always forward OIDC env vars unconditionally

Forward `ACTIONS_ID_TOKEN_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN` to the gateway container in all cases, regardless of whether any MCP server uses OIDC auth. This is simpler — no detection logic required. However, it unnecessarily exposes the OIDC token endpoint to the gateway in workflows that have no need for it, which violates the principle of least privilege. Token minting from these endpoints is only safe when the specific permission (`id-token: write`) has been deliberately granted by the workflow author.

#### Alternative 2: Let the user configure OIDC var forwarding explicitly in the workflow frontmatter

Add a top-level `forward-oidc-vars: true` option to the workflow configuration that users must set manually. This avoids any detection heuristics but creates a footgun: users configuring `auth.type: "github-oidc"` on an MCP server would have to separately remember to set a second flag. Given that the compiler already has access to the full tool configuration at compile time, auto-detection is strictly better UX and eliminates a class of configuration errors.

#### Alternative 3: Forward OIDC vars via the firewall/agent-container layer

Rejected. The gateway is launched separately by the runner with `docker run`; it does not inherit the AWF agent process environment.

### Consequences

#### Positive
- HTTP MCP servers with `auth.type: "github-oidc"` can mint audience-bound tokens in gateway runtime.
- OIDC runtime variables are only forwarded to the gateway when needed.
- AWF agent boundary is explicit: the sandboxed agent does not receive these variables.
- Existing lock workflows that already include conditional gateway forwarding remain compatible.

#### Negative
- OIDC detection relies on MCP tool parsing; malformed tool config can suppress detection.

#### Neutral
- Workflows without HTTP MCP GitHub OIDC are unaffected.
- For pinned legacy AWF versions that do not support `--exclude-env`, gh-aw cannot enforce agent-side exclusion via CLI flags; users should recompile and/or pin a modern AWF version to get hardened boundary behavior.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### OIDC Environment Variable Forwarding

1. The compiler **MUST** inspect tools at compile time to detect HTTP MCP servers with `auth.type: "github-oidc"`.
2. The compiler **MUST** append `-e ACTIONS_ID_TOKEN_REQUEST_URL` and `-e ACTIONS_ID_TOKEN_REQUEST_TOKEN` to the runner-owned MCP gateway `docker run` command iff that auth is present.
3. The compiler **MUST NOT** append those flags when no HTTP MCP server uses `auth.type: "github-oidc"`.
4. The gateway stdin/config payload **MUST** contain only OIDC auth metadata (`type` and optional `audience`) and **MUST NOT** embed Actions OIDC runtime variable names or values.
5. When HTTP MCP GitHub OIDC auth is configured, gh-aw **MUST** exclude `ACTIONS_ID_TOKEN_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN` from the AWF agent container environment when `--exclude-env` support is available.

### OIDC Detection Logic

1. The detection helper **MUST** skip tools that are not configurable HTTP MCP servers (i.e., built-in tools: `github`, `playwright`, `cache-memory`, `agentic-workflows`, `safe-outputs`, `mcp-scripts`).
2. The detection helper **MUST** check only tools whose configuration resolves to a valid MCP config with `type: "http"`.
3. The detection helper **SHOULD** log a message at the MCP environment log level when a tool with GitHub OIDC auth is found, to aid in debugging.
4. The detection helper **MAY** return early (`true`) as soon as the first matching tool is found, without inspecting remaining tools.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: the MCP Gateway `docker run` command includes `-e ACTIONS_ID_TOKEN_REQUEST_URL -e ACTIONS_ID_TOKEN_REQUEST_TOKEN` when and only when at least one HTTP MCP server in the compiled workflow uses `auth.type: "github-oidc"`. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
