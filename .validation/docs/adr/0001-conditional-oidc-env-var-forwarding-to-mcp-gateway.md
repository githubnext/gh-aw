# ADR-0001: Conditional OIDC Environment Variable Forwarding to MCP Gateway Container

**Date**: 2026-04-11
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw compiler generates a `docker run` command that launches the MCP Gateway container. The host GitHub Actions runner has the OIDC token endpoint variables (`ACTIONS_ID_TOKEN_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN`) available in its environment. The firewall layer (gh-aw-firewall#1796) was previously fixed to forward these variables into the agent container, but the second hop — from the agent container to the MCP Gateway container — was never wired up. As a result, HTTP MCP servers that require GitHub OIDC authentication (`auth.type: "github-oidc"`) fail to mint tokens because the gateway cannot reach the OIDC endpoint. These two variables are only meaningful when at least one configured HTTP MCP server uses OIDC auth; forwarding them unconditionally would expose the token endpoint unnecessarily.

### Decision

We will detect whether any HTTP MCP server in the workflow tools configuration uses `auth.type: "github-oidc"` at compile time, and only append `-e ACTIONS_ID_TOKEN_REQUEST_URL -e ACTIONS_ID_TOKEN_REQUEST_TOKEN` to the MCP Gateway `docker run` command when that condition is true. This detection is performed by the new `hasGitHubOIDCAuthInTools()` helper, which iterates the tool map and checks each HTTP MCP server's auth configuration. The approach is consistent with the existing pattern of conditionally adding other environment variables (e.g., OTEL tracing vars) to the docker command only when the corresponding feature is active.

### Alternatives Considered

#### Alternative 1: Always forward OIDC env vars unconditionally

Forward `ACTIONS_ID_TOKEN_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN` to the gateway container in all cases, regardless of whether any MCP server uses OIDC auth. This is simpler — no detection logic required. However, it unnecessarily exposes the OIDC token endpoint to the gateway in workflows that have no need for it, which violates the principle of least privilege. Token minting from these endpoints is only safe when the specific permission (`id-token: write`) has been deliberately granted by the workflow author.

#### Alternative 2: Let the user configure OIDC var forwarding explicitly in the workflow frontmatter

Add a top-level `forward-oidc-vars: true` option to the workflow configuration that users must set manually. This avoids any detection heuristics but creates a footgun: users configuring `auth.type: "github-oidc"` on an MCP server would have to separately remember to set a second flag. Given that the compiler already has access to the full tool configuration at compile time, auto-detection is strictly better UX and eliminates a class of configuration errors.

#### Alternative 3: Forward OIDC vars via the firewall/agent-container layer only, not the docker command

Rely on the firewall forwarding the variables from the host into the agent container and then have the MCP Gateway inherit them via the container's process environment rather than explicit `-e` flags. This would work only if the gateway process is spawned as a child process, which it is not — it runs inside a separate Docker container started with `docker run`. Environment inheritance does not cross a `docker run` boundary without explicit `-e` flags.

### Consequences

#### Positive
- HTTP MCP servers configured with `auth.type: "github-oidc"` can successfully mint OIDC tokens inside the gateway container.
- OIDC token endpoint variables are forwarded only when needed, following the principle of least privilege.
- The implementation is consistent with the existing conditional env-var-forwarding pattern used for OTEL tracing.
- No workflow author action is required; detection is automatic from existing tool configuration.

#### Negative
- The `hasGitHubOIDCAuthInTools()` function must maintain a hardcoded blocklist of standard tool names (`github`, `playwright`, `cache-memory`, `agentic-workflows`, `safe-outputs`, `mcp-scripts`) that are skipped during detection. This list must be kept in sync if new built-in tools are added.
- If a tool configuration is malformed (e.g., `getMCPConfig` returns an error), that tool is silently skipped rather than causing a compile error; OIDC auth on that tool will silently not work.

#### Neutral
- The `hasOIDCAuth` boolean is computed once and reused in both the `-e` flag section and the dedup map section of the docker command builder, so detection cost is O(n) over tools and paid only once per compile.
- Workflows that do not use OIDC auth are unaffected by this change.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### OIDC Environment Variable Forwarding

1. The compiler **MUST** inspect the tools configuration at compile time to determine whether any HTTP MCP server has `auth.type` equal to `"github-oidc"`.
2. The compiler **MUST** append `-e ACTIONS_ID_TOKEN_REQUEST_URL` and `-e ACTIONS_ID_TOKEN_REQUEST_TOKEN` to the MCP Gateway `docker run` command if and only if at least one HTTP MCP server with `auth.type: "github-oidc"` is present in the tools configuration.
3. The compiler **MUST NOT** append these environment variable flags when no HTTP MCP server uses `auth.type: "github-oidc"`.
4. The compiler **MUST** register `ACTIONS_ID_TOKEN_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN` in the deduplication map when they are forwarded, to prevent duplicate `-e` entries.

### OIDC Detection Logic

1. The detection helper **MUST** skip tools that are not configurable HTTP MCP servers (i.e., built-in tools: `github`, `playwright`, `cache-memory`, `agentic-workflows`, `safe-outputs`, `mcp-scripts`).
2. The detection helper **MUST** check only tools whose configuration resolves to a valid MCP config with `type: "http"`.
3. The detection helper **SHOULD** log a message at the MCP environment log level when a tool with GitHub OIDC auth is found, to aid in debugging.
4. The detection helper **MAY** return early (`true`) as soon as the first matching tool is found, without inspecting remaining tools.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: the MCP Gateway `docker run` command includes `-e ACTIONS_ID_TOKEN_REQUEST_URL -e ACTIONS_ID_TOKEN_REQUEST_TOKEN` when and only when at least one HTTP MCP server in the compiled workflow uses `auth.type: "github-oidc"`. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
