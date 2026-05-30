# ADR-26377: Use GITHUB_REPOSITORY Env Var for MCP Subprocess Repo Resolution

**Date**: 2026-04-15
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `logs` and `audit` MCP tools registered by `registerLogsTool` and `registerAuditTool` in `pkg/cli/mcp_tools_privileged.go` spawn subprocesses that call `gh run list` and related commands. When `--repo` is not explicitly supplied, `gh` falls back to git to auto-detect the repository from the local working directory. MCP server containers in the agentic-workflows sandbox do not have git installed, causing these tool invocations to fail with `unable to find git executable in PATH`. The `GITHUB_REPOSITORY` environment variable (`owner/repo` format) is already forwarded to the MCP server container via `env_vars` in the agentic-workflows configuration, making it a zero-cost source of truth for the target repository.

### Decision

We will conditionally append `--repo <owner/repo>` to the subprocess argument list of the `logs` and `audit` MCP tools whenever the `GITHUB_REPOSITORY` environment variable is non-empty. This allows `gh` to resolve the repository without invoking git, making the tools functional in sandboxed environments that lack a git binary. The injection is conditional so that the tools continue to behave identically in local development environments where `GITHUB_REPOSITORY` is typically unset.

### Alternatives Considered

#### Alternative 1: Install git in the MCP server container

Git could be added as a dependency of the MCP server container image so that `gh`'s auto-detection works as designed. This was rejected because it adds a heavyweight, security-relevant binary to a minimal sandbox container for the sole purpose of supporting a fallback path in `gh`; it also increases the container attack surface and image size without addressing the root cause.

#### Alternative 2: Add an explicit `repo` parameter to the MCP tool schema

The `logs` and `audit` tool schemas could expose a `repo` parameter that callers must supply. This was rejected because it breaks backward compatibility for every existing caller and shifts repository-resolution responsibility to the AI agent (which already has `GITHUB_REPOSITORY` available implicitly). The env-var approach achieves the same result transparently.

#### Alternative 3: Always require `--repo` by reading from configuration at server startup

The MCP server startup code could resolve the repository once at initialization time and bake it into the tool handlers as a closure. This was not chosen because it complicates the server initialization path and makes unit-testing harder; the conditional env-var read at call time is simpler and equally correct.

### Consequences

#### Positive
- MCP tools work correctly in git-less sandbox containers without any changes to caller code or tool schemas.
- `GITHUB_REPOSITORY` is a standard GitHub Actions environment variable, so the fix is idiomatic and requires no new infrastructure.
- The change is backward-compatible: when `GITHUB_REPOSITORY` is empty, behavior is unchanged.

#### Negative
- The tools now have an implicit, undocumented dependency on the `GITHUB_REPOSITORY` environment variable; callers in non-Actions environments must be aware that omitting this variable forces git-based fallback.
- If `GITHUB_REPOSITORY` is set incorrectly (e.g., wrong value injected), it will be silently propagated to the subprocess without validation.

#### Neutral
- Two new integration-style unit tests (`TestLogsToolPassesGithubRepositoryAsRepoFlag`, `TestAuditToolPassesGithubRepositoryAsRepoFlag`) use `mcp.NewInMemoryTransports()` and a mock `execCmd` to capture subprocess arguments, establishing a test pattern for future MCP tool behavior verification.
- The same fix is applied symmetrically to both the `logs` and `audit` tool handlers; any future MCP tools that spawn `gh` subprocesses should follow the same pattern.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Repository Resolution in MCP Tool Subprocess Calls

1. MCP tool handlers that spawn `gh` subprocesses **MUST** append `--repo <owner/repo>` to the subprocess argument list when the `GITHUB_REPOSITORY` environment variable is non-empty.
2. MCP tool handlers **MUST NOT** append `--repo` when `GITHUB_REPOSITORY` is empty or unset, preserving existing git-based fallback behavior.
3. The value appended as `--repo` **MUST** be the verbatim value of `GITHUB_REPOSITORY` (in `owner/repo` format) without modification or validation.
4. Implementations **MUST NOT** install or invoke `git` solely to work around the absence of the `--repo` flag; the env-var injection pattern is the approved resolution mechanism for sandboxed environments.

### Environment Variable Contract

1. The `GITHUB_REPOSITORY` environment variable **MUST** be forwarded to MCP server containers via the agentic-workflows `env_vars` configuration when the server is deployed in a GitHub Actions sandbox.
2. Callers running MCP tools outside of GitHub Actions **SHOULD** set `GITHUB_REPOSITORY` explicitly if git is unavailable in the execution environment.
3. Implementations **MAY** log a debug-level message when `GITHUB_REPOSITORY` is used for `--repo` injection to aid in diagnosing unexpected repository resolution.

### Testing

1. Every MCP tool handler that conditionally injects `--repo` **MUST** have a unit test that verifies the flag is appended when `GITHUB_REPOSITORY` is set and omitted when it is empty.
2. Tests **SHOULD** use `mcp.NewInMemoryTransports()` with a mock `execCmd` to capture subprocess arguments without spawning a real `gh` process.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24439752420) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
