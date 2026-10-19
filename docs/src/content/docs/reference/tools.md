---
title: Tools
description: Configure GitHub API tools, browser automation, and AI capabilities available to your agentic workflows, including GitHub tools and custom MCP servers.
sidebar:
  order: 700
---

[Tools](/gh-aw/reference/glossary/#tools) are defined in the frontmatter to specify which GitHub API calls, browser automation, and AI capabilities are available to your workflow:

```yaml wrap
tools:
  edit:
  bash: true
```

Some tools are available by default. All tools declared in imported components are merged into the final workflow.

## Built-in Tools

### Ecosystem Profile (`profile:`)

:::caution[Experimental]
Ecosystem tool profiles are experimental and their configuration and behavior
may change. Compiling a workflow that uses `tools.profile`, directly or through
an import, emits `Using experimental feature: tools.profile`.
:::

`tools.profile` configures an ecosystem's runtime, sandbox paths, shell commands,
and MCP tools. The first supported profile is `go`:

```yaml wrap
tools:
  profile: go
```

The Go profile installs Go 1.26 through the runtime manager, plus the pinned
`gopls` v0.23.0 language server and its built-in MCP server. It enables `go version`,
`go env`, `go list`, `go doc`, `go build`, `go test`, `go vet`, and `gofmt` in the
agent's sandboxed shell when no explicit bash policy is configured.

The shell entries are matched by command prefix, not by argument. In particular,
`go build`, `go test`, and `go vet` can execute arbitrary programs through
supported flags or test code. Treat workflows that use these commands as trusted
code execution inside the agent sandbox; this list is not an argument-level
security boundary.

The `gopls` MCP sidecar exposes only `go_workspace`, `go_search`,
`go_file_context`, `go_package_api`, `go_diagnostics`, and `go_symbol_references`.
It mounts the workspace, runner tool cache, and installed profile binaries
read-only; only the profile's isolated caches are writable. Builds and tests
execute inside the agent sandbox, never in a host-side MCP shell runner.
Sidecar analysis disables cgo, telemetry, and automatic Go toolchain downloads.
Setup preloads workspace module dependencies before starting the sidecar, which
sets `GOPROXY=off` and `GOSUMDB=off` to prevent module lookups from reaching
external Go servers. The sidecar container itself is not isolated by the agent
firewall, so these settings do not restrict unrelated outbound network traffic.
Before exporting its runtime environment, setup verifies gopls semantic readiness
against a disposable, dependency-free Go module with module-network access
disabled and a 60-second deadline. Missing binaries, installation failures, and
missing probe symbols fail setup rather than falling back to text-only analysis.

Setup stores binaries and caches under `$RUNNER_TEMP/gh-aw/tool-profiles/go` and
exports `GH_AW_TOOL_PROFILE_GO_DIR`, `GH_AW_TOOL_PROFILE_GO_ROOT`, and
`GH_AW_TOOL_PROFILE_GO_PATH`.
The latter includes both the installed gopls directory and the selected Go
toolchain's `bin` directory. Other containerized code-analysis servers can
forward that PATH when they declare the same read-only binary and toolchain
mounts. Forwarding only Go's `bin` directory does not make gopls available.
The profile also sets `GOBIN` to its writable, sandbox-mounted `cache/bin`
directory and adds it to PATH. Additional `go install` commands use that
directory rather than an unmounted `$HOME/go/bin`.

Profiles work with Copilot, Claude, Gemini, and Codex, including `tools.cli-proxy`, and
can be declared in shared workflow imports. Repeated declarations install the
profile once. Explicit tool disabling is preserved; for example, `bash: false`
keeps shell access disabled while retaining the gopls MCP tools. Set
`cli-proxy: false` alongside it.

Codex cannot enforce the profile's shell-command allowlist, so it requires an
explicit shell choice even when the profile comes from an import. Builds, tests,
and CLI proxies require unrestricted sandboxed shell access:

```aw wrap
engine: codex
tools:
  profile: go
  bash: true
```

MCP-only Go analysis disables both the shell and shell-based CLI proxies:

```aw wrap
engine: codex
tools:
  profile: go
  bash: false
  cli-proxy: false
```

Profile runtime settings are defaults, so `go.mod` cannot select an older
toolchain accidentally. Use `runtimes.go` to override the Go version or setup
action (Go 1.26 or newer is
required by the pinned gopls). The profile requires
an enabled Docker agent sandbox (`docker` or `docker-sudo-iptables`).
Cloud Hypervisor's runner-temp export is read-only; NVX exports only the workspace
and optional tool cache. Both reject additional host volume mounts, so neither
currently provides the profile's read-only binaries and writable cache contract.
Support is tracked in
[github/gh-aw-firewall#10231](https://github.com/github/gh-aw-firewall/issues/10231).

Profile definitions are embedded shared-workflow Markdown files under
`pkg/workflow/data/tool-profiles/`. Their frontmatter declares `runtimes`,
`sandbox.agent.mounts`, `pre-agent-steps`, `tools`, `network`, and `mcp-servers`,
so new ecosystems reuse the same compilation and validation paths.

### Edit Tool (`edit:`)

Allows file editing in the GitHub Actions workspace.

```yaml wrap
tools:
  edit:
```

### GitHub Tools (`github:`)

Configure GitHub API operations including toolsets, remote/local modes, and authentication.

```yaml wrap
tools:
  github:
    toolsets: [repos, issues]
```

See **[GitHub Tools Reference](/gh-aw/reference/github-tools/)** for complete configuration options.

### Linear Tools (`linear:`)

Connect to [Linear's official hosted MCP server](https://linear.app/docs/mcp) with the `LINEAR_API_KEY` GitHub Actions secret:

```yaml wrap
tools:
  linear: {}
```

Set `token` to use a different secret, `toolsets` to enable groups such as `issues` and `projects`, `allowed` to further restrict tool names, and `required: false` to make connectivity best-effort:

```yaml wrap
tools:
  linear:
    token: ${{ secrets.CUSTOM_LINEAR_TOKEN }}
    toolsets: [issues, projects]
    allowed: ["*"]
    required: true
```

Supported toolsets are `all`, `attachments`, `comments`, `customers`, `cycles`, `diffs`, `documentation`, `documents`, `initiatives`, `issues`, `milestones`, `projects`, `status_updates`, `teams`, and `users`. The compiler expands toolsets into the gateway's allowed-tool list, and any `allowed` names or wildcards must match a tool in the selected toolsets.

Linear always uses Linear's server-enforced read-only endpoint. The credential is passed to the gateway as an environment variable and sent as an `Authorization: Bearer` header, not embedded in MCP configuration. Like other remote MCP servers, Linear also works with `tools.cli-proxy: true`.

### Jira Tools (`jira:`)

Connect to Atlassian's official remote Rovo MCP endpoint from non-interactive GitHub Actions workloads. Browser OAuth, device login, and user-consent flows are not supported.

Use an Atlassian service account API key:

```yaml wrap
tools:
  jira:
    auth:
      type: service-account
      token: ${{ secrets.ATLASSIAN_SERVICE_ACCOUNT_API_KEY }}
    allowed:
      - getJiraIssue
      - searchJiraIssuesUsingJql
```

Or use an Atlassian account email and API token:

```yaml wrap
tools:
  jira:
    auth:
      type: api-token
      email: ${{ secrets.ATLASSIAN_EMAIL }}
      token: ${{ secrets.ATLASSIAN_API_TOKEN }}
    allowed:
      - getJiraIssue
      - searchJiraIssuesUsingJql
```

The `allowed` list is required and accepts only these read-only Jira tools: `getIssueLinkTypes`, `getJiraIssue`, `getJiraIssueRemoteIssueLinks`, `getJiraIssueTypeMetaWithFields`, `getJiraProjectIssueTypesMetadata`, `getTransitionsForJiraIssue`, `getVisibleJiraProjects`, `lookupJiraAccountId`, and `searchJiraIssuesUsingJql`.

`allowed: ["*"]` is shorthand for enabling that fixed list at compile time; it never grants access to the full, unrestricted MCP tool set. Omitting `allowed` or naming a write-capable tool is rejected.

The endpoint defaults to `https://mcp.atlassian.com/v1/mcp`. Set `url` only when your organization uses another HTTPS Atlassian MCP endpoint. Credentials must be direct GitHub Actions secret expressions; service account keys use the HTTP bearer scheme while API tokens use HTTP Basic authentication generated at runtime.

### Bash Tool (`bash:`)

Enables shell command execution in the workspace. Defaults to safe commands (`echo`, `printf`, `ls`, `pwd`, `cat`, `head`, `tail`, `grep`, `wc`, `sort`, `uniq`, `date`, `yq`).

```yaml wrap
tools:
  bash:                              # Default safe commands
  bash: []                           # Disable all commands
  bash: ["echo", "ls", "git status"] # Specific commands only
  bash: [":*"]                       # All commands (use with caution)
```

Use wildcards like `git:*` for command families or `:*` for unrestricted access.

### Web Tools

Enable web content fetching and search capabilities:

```yaml wrap
tools:
  web-fetch:   # Fetch web content
  web-search:  # Search the web (engine-dependent)
```

**Note:** Some engines require third-party Model Context Protocol (MCP) servers for web search. See [Using Web Search](/gh-aw/reference/web-search/).

Web search is disabled by default for **Codex** and **Claude**. Claude enables it only when `web-search:` is declared. Codex enables a shared native search/browsing tool when either `web-search:` or `web-fetch:` is declared; the two capabilities cannot be disabled independently. With neither tool, Codex runs with `-c web_search="disabled"`. Use `network.hosted-web` to restrict Codex's hosted retrieval.

**Copilot** runs in offline BYOK mode, which disables its native web tools. Compilation rejects `web-search:` and, in CLI mode, `web-fetch:`. Use an MCP server for search or fetch. With `engine.copilot-sdk: true`, `web-fetch:` remains supported by a custom proxy-aware tool that follows `network.allowed`; this does not enable native web search.

The **Gemini** and **Pi** engines have no built-in web search. Declaring `web-search:` with either engine has no effect, and `gh aw compile` prints a warning. To search the web with these engines, configure a third-party MCP server as described in [Using Web Search](/gh-aw/reference/web-search/).

### Playwright Tool (`playwright:`)

Configure Playwright for browser automation and testing:

```yaml wrap
tools:
  playwright:
    version: "1.56.1"  # Optional: specify version
```

See **[Playwright Reference](/gh-aw/reference/playwright/)** for complete configuration options, network access, browser support, and example workflows.

### Cache Memory (`cache-memory:`)

Persistent memory storage across workflow runs for trends and historical data.

```yaml wrap
tools:
  cache-memory:
```

See **[Cache Memory Reference](/gh-aw/reference/cache-memory/)** for complete configuration options and usage examples.

### Drive Memory (`drive-memory:`) — Private Preview

Drive memory is an experimental, feature-gated GitHub Drives integration. Do not
configure it unless GitHub has explicitly enrolled the repository in the private
preview.

The [Drive Memory Reference](/gh-aw/experimental/drive-memory/) records the preview
behavior for enrolled repositories; it is not a recommendation for general use.

### Repo Memory (`repo-memory:`)

Repository-specific memory storage for maintaining context across executions.

```yaml wrap
tools:
  repo-memory:
```

See **[Repo Memory Reference](/gh-aw/reference/repo-memory/)** for complete configuration options and usage examples.

### QMD Documentation Search (`qmd:`) — Experimental

Build a local vector search index over documentation files and expose it as an MCP search tool. The index is built in a dedicated indexing job (no `contents: read` needed in the agent job):

```yaml wrap
tools:
  qmd:
    checkouts:
      - pattern: "docs/**/*.md"
```

See **[QMD Documentation Search](/gh-aw/experimental/qmd/)** for complete configuration options, checkout support, GitHub search integration, and cache key usage.

### Introspection on Agentic Workflows (`agentic-workflows:`)

Provides workflow introspection, log analysis, and debugging tools. Requires `actions: read` permission:

```yaml wrap
permissions:
  actions: read
tools:
  agentic-workflows:
```

See [GH-AW as an MCP Server](/gh-aw/reference/gh-aw-as-mcp-server/) for available operations.

The built-in container runs with the runner's UID and GID unless a container user is explicitly configured. This keeps downloaded logs runner-owned without widening their owner-only permissions, so the post-agent secret-redaction step can read them. Its default `HOME` is the backend container's ephemeral `/tmp`, not `/root`. Redaction failures still remove unsanitized sources and fail the step.

The `compile` tool uses a dedicated read-only mode with `--no-emit` by default. Remote imports and nested body-level includes must already be cached; container validation does not pull missing Docker images. Ordinary CLI `--no-emit` compilation can still populate import caches. Workflow-writing tools (`add`, `update`, and `fix`), emitted compilation, dry-run compilation, and external scanners require an explicit opt-in at server startup:

```aw wrap
tools:
  agentic-workflows:
    allow-write-tools: true
```

This sets `--allow-write-tools` while preserving actor validation. It permits local workflow modifications and compiler scanner execution, not additional GitHub permissions or workflow dispatch. Enable it only for trusted clients and workflow sources.

### MCP CLI Mounting (`cli-proxy:`)

Set `tools.cli-proxy: true` to mount each user-facing MCP server as a standalone CLI tool on `PATH`, so the agent can invoke it from shell instead of through the MCP protocol:

```yaml wrap
tools:
  cli-proxy: true
```

With CLI mounting enabled, workflow-accessible servers such as `safeoutputs` and `mcpscripts` are wrapped as executables:

```bash
safeoutputs add_comment '{"item_number":42,"body":"### Result\n\nIt\u0027s ready.\nQuoted text: \"hello\".\nLiteral path: C:\\temp\\file.txt"}'
mcpscripts mcpscripts-gh --args "issue list --limit 5"
```

For `add_comment`, use `item_number` rather than `issue_number`; schema validation strips the latter. Prefer one single-quoted JSON object argument for safe outputs. Inside JSON strings, encode newlines as `\n`, double quotes as `\"`, backslashes as `\\`, and apostrophes as `\u0027`; Unicode can remain literal. Stdin is also supported: `safeoutputs <tool_name> .` reads a complete JSON object from stdin. This does not grant permission for helper commands or redirections. No auxiliary tools, pipelines, temporary files, or shell expansion are needed for inline JSON.

CLI mounting changes only the agent-facing interface: the MCP gateway still starts normally, but mounted servers are removed from the MCP tool list and accessed via shell. This can reduce token use from large tool schemas and simplify prompts when shell-style invocation is preferred.

Defaults to `false`.

CLI mounting requires shell access because the wrappers are ordinary executables invoked from bash. GitHub `gh-proxy` mode is also shell-backed because GitHub reads are performed with the `gh` CLI. When `tools.bash` is disabled (`bash: false` or `bash: []`), `cli-proxy: true` and `tools.github.mode: gh-proxy` are rejected at compile time, and strict mode requires `cli-proxy: false` to be stated explicitly:

```yaml wrap
tools:
  bash: false
  cli-proxy: false
  github:
    mode: local
```

With `cli-proxy: false` and an MCP-backed GitHub mode (`local` or `remote`), MCP servers (including `safeoutputs`) remain available through the MCP protocol, and the CLI-only instructions are omitted from the generated prompt. Run `gh aw fix` to add the explicit setting to existing workflows.

## Tool Timeout Configuration

### Tool Operation Timeout (`tools.timeout`)

Sets the per-operation timeout in seconds when supported by the engine. Claude and Codex default to 60 seconds. For Codex, this configures native MCP tool calls, not shell-command duration; use `timeout-minutes` for the overall execution limit.

```yaml wrap
tools:
  timeout: 120   # seconds
```

### MCP Server Startup Timeout (`tools.startup-timeout`)

Sets the timeout in seconds for MCP server initialization. Default is 120 seconds.

```yaml wrap
tools:
  startup-timeout: 60   # seconds
```

Both fields accept either an integer or a GitHub Actions expression string, enabling `workflow_call` reusable workflows to parameterize these values:

```yaml wrap
tools:
  timeout: ${{ inputs.tool-timeout }}
  startup-timeout: ${{ inputs.startup-timeout }}
```

> [!NOTE]
> Expression values are passed through environment variables and resolved before execution. Codex's configuration renderer converts the resolved values into numeric TOML settings; expressions do not fall back to defaults. An explicit per-server timeout in `engine.config` takes precedence over `tools.timeout` or `tools.startup-timeout`.

## Custom MCP Servers (`mcp-servers:`)

Integrate custom Model Context Protocol servers for third-party services:

```yaml wrap
mcp-servers:
  slack:
    command: "npx"
    args: ["-y", "@slack/mcp-server"]
    env:
      SLACK_BOT_TOKEN: "${{ secrets.SLACK_BOT_TOKEN }}"
    allowed: ["send_message", "get_channel_history"]
```

**Options**: `command` + `args` (process-based), `container` (Docker image), `url` + `headers` (HTTP endpoint), `registry` (MCP registry URI), `env` (environment variables), `allowed` (tool restrictions), `required` (startup criticality). See [MCPs Guide](/gh-aw/guides/mcps/) for setup.

### Required Field

MCP servers must pass a startup connectivity check before the agent starts. By default every server is startup-critical: if it cannot be reached, the workflow fails. Set `required: false` to mark a server as best-effort, so an unreachable server logs a warning and the workflow continues without it:

```yaml wrap
mcp-servers:
  datadog:
    type: http
    url: "https://mcp.datadoghq.com/api/unstable/mcp-server/mcp"
    required: false
```

Use this for optional integrations whose transient outages (for example an HTTP 503 from a hosted endpoint) should not take down the other configured servers. At least one server must still connect successfully for startup to proceed.

### Registry Field

The `registry` field specifies the source URI of an MCP server in a registry. It is informational — useful for documenting server origin and enabling registry-aware tooling — and does not affect execution. gh-aw does not enforce registry usage. Works with both stdio and HTTP servers:

```yaml wrap
mcp-servers:
  filesystem:
    registry: "https://api.mcp.github.com/v0/servers/modelcontextprotocol/filesystem"
    command: "npx"
    args: ["-y", "@modelcontextprotocol/server-filesystem"]
```

## Learn More

See [GitHub Tools](/gh-aw/reference/github-tools/) for GitHub API operations, toolsets, and modes; [Playwright](/gh-aw/reference/playwright/) for browser automation; [Cache Memory](/gh-aw/reference/cache-memory/) and [Repo Memory](/gh-aw/reference/repo-memory/) for persistent context; [MCP Scripts](/gh-aw/reference/mcp-scripts/) for custom inline tools; and [MCPs](/gh-aw/guides/mcps/) for end-to-end Model Context Protocol setup.
