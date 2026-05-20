---
title: GH-AW as an MCP Server
description: Use the gh-aw MCP server to expose CLI tools to AI agents via Model Context Protocol, enabling secure workflow management.
sidebar:
  order: 400
---

The `gh aw mcp-server` command exposes GitHub Agentic Workflows CLI commands as MCP tools, allowing chat systems and workflows to manage agentic workflows programmatically.

Start the server:

```bash wrap
gh aw mcp-server
```

Or configure for any Model Context Protocol (MCP) host:

```yaml wrap
command: gh
args: [aw, mcp-server]
```

## Configuration Options

### HTTP Server Mode

Run with HTTP/SSE transport using `--port`:

```bash wrap
gh aw mcp-server --port 8080
```

### Actor Validation

Control access to logs and audit tools based on repository permissions using `--validate-actor`:

```bash wrap
gh aw mcp-server --validate-actor
```

When enabled, the logs and audit tools require write/maintain/admin repository access. The server reads `GITHUB_ACTOR` and `GITHUB_REPOSITORY` env vars and caches permission check results for 1 hour. Without validation (default), all tools are available without checks.

## Configuring with GitHub Copilot Agent

Configure GitHub Copilot Agent to use gh-aw MCP server:

```bash wrap
gh aw init
```

This creates `.github/workflows/copilot-setup-steps.yml` that sets up Go, GitHub CLI, and gh-aw extension before agent sessions start, making workflow management tools available to the agent. MCP server integration is enabled by default. Use `gh aw init --no-mcp` to skip MCP configuration.

## Configuring with Copilot CLI

To add the MCP server in the interactive Copilot CLI session, start `copilot` and run:

```text
/mcp add github-agentic-workflows gh aw mcp-server
```

## Configuring with VS Code

Configure VS Code Copilot Chat to use gh-aw MCP server:

```bash wrap
gh aw init
```

This creates `.github/mcp.json` and `.github/workflows/copilot-setup-steps.yml`. MCP server integration is enabled by default. Use `gh aw init --no-mcp` to skip MCP configuration.

Alternatively, create `.github/mcp.json` manually:

```json wrap
{
  "mcpServers": {
    "github-agentic-workflows": {
      "command": "gh",
      "args": ["aw", "mcp-server"]
    }
  }
}
```

Reload VS Code after making changes.

## Configuring with Docker

If `gh` is not installed locally, use the `ghcr.io/github/gh-aw` Docker image. The image ships with the GitHub CLI and gh-aw pre-installed and uses `mcp-server` as the default command.

```json wrap
{
  "command": "docker",
  "args": [
    "run", "--rm", "-i",
    "-e", "GITHUB_TOKEN",
    "-e", "GITHUB_ACTOR",
    "ghcr.io/github/gh-aw:latest",
    "mcp-server"
  ]
}
```

Pass your GitHub token via the `GITHUB_TOKEN` environment variable. Add `--validate-actor` to the `args` array to enforce permission checks based on `GITHUB_ACTOR`.

## Available Tools

The MCP server exposes the following tools for workflow management:

### `status`

Show status of agentic workflow files and workflows.

- `pattern` (optional): Filter workflows by name pattern
- `jq` (optional): Apply jq filter to JSON output

Returns a JSON array with `workflow`, `agent`, `compiled`, `status`, and `time_remaining` fields.

### `compile`

Compile Markdown workflows to GitHub Actions YAML with optional static analysis.

- `workflows` (optional): Array of workflow files to compile (empty for all)
- `strict` (optional): Enforce strict mode validation (default: true)
- `fix` (optional): Apply automatic codemod fixes before compiling
- `zizmor`, `poutine`, `actionlint` (optional): Run security scanners/linters
- `jq` (optional): Apply jq filter to JSON output

Returns a JSON array with `workflow`, `valid`, `errors`, `warnings`, and `compiled_file` fields.

> [!NOTE]
> The `actionlint`, `zizmor`, and `poutine` scanners use Docker images that download on first use. If images are still being pulled, the tool returns a "Docker images are being downloaded. Please wait and retry the compile command." message. Wait 15–30 seconds, then retry the request.

### `logs`

Download and analyze workflow logs with timeout handling and size guardrails.

- `workflow_name` (optional): Workflow name (empty for all)
- `count` (optional): Number of runs to download (default: 100)
- `start_date`, `end_date` (optional): Date range filter (YYYY-MM-DD or delta like `-1w`)
- `engine`, `firewall`, `no_firewall`, `branch` (optional): Run filters
- `after_run_id`, `before_run_id` (optional): Pagination by run ID
- `timeout` (optional): Max seconds to download (default: 50)
- `max_tokens` (optional): Output token guardrail (default: 12000)
- `jq` (optional): Apply jq filter to JSON output

Returns JSON with workflow run data and metrics, or continuation parameters if timeout occurred.

### `audit`

Investigate one or more workflow runs and generate a detailed report. With a single run, returns a full audit. With two or more runs, the first is the base and the rest are compared against it (diff mode).

At least one of the following run-identifier fields must be supplied:

- `run_ids_or_urls` (array of strings, preferred): One or more run IDs or URLs. Single item produces a detailed audit; multiple items produce a diff against the first.
- `run_id` (string or number): Alias for a single run identifier. Use this when only one run is being audited.
- `run_id_or_url` (string or number, deprecated): Original single-run field. Accepted for backward compatibility — prefer `run_ids_or_urls` or `run_id`.

Each identifier accepts a numeric run ID, run URL, job URL, or job URL with a step anchor (for example `https://github.com/owner/repo/actions/runs/123/job/456#step:7:1`).

Optional parameters:

- `artifacts` (array of strings): Artifact sets to download. Valid values: `all`, `activation`, `agent`, `detection`, `firewall`, `github-api`, `mcp`. Defaults to all sets.
- `experiment` (string): Filter to runs assigned to this experiment name.
- `variant` (string): Filter to runs assigned this variant. Requires `experiment`.
- `jq` (optional): Apply jq filter to JSON output.

Single-run returns JSON with `overview`, `metrics`, `jobs`, `downloaded_files`, `missing_tools`, `mcp_failures`, `errors`, `warnings`, `tool_usage`, `firewall_analysis`, and (when present) `experiments`. Multi-run diff returns JSON describing the changes between the base run and each comparison run.

### `checks`

Classify CI check state for a pull request and return a normalized result.

- `pr_number` (required): Pull request number to classify CI checks for
- `repo` (optional): Repository in `owner/repo` format (defaults to current repository)

Returns JSON with:
- `state`: Aggregate check state across all check runs and commit statuses
- `required_state`: State derived from check runs and policy commit statuses only (ignores optional third-party statuses like Vercel/Netlify deployments)
- `pr_number`, `head_sha`, `check_runs`, `statuses`, `total_count`

Normalized states: `success`, `failed`, `pending`, `no_checks`, `policy_blocked`.

Use `required_state` as the authoritative CI verdict in repos with optional deployment integrations.

### `mcp-inspect`

Inspect MCP servers in workflows and list available tools, resources, and roots.

- `workflow_file` (optional): Workflow file to inspect (empty to list all workflows with MCP servers)
- `server` (optional): Filter to specific MCP server
- `tool` (optional): Show detailed info about a specific tool (requires `server`)

Returns formatted text listing MCP servers, their tools/resources/roots, secret availability, and detailed tool info when `tool` is specified.

### `add`

Add workflows from remote repositories to `.github/workflows`.

- `workflows` (required): Array of workflow specs in `owner/repo/workflow-name[@version]` format
- `number` (optional): Create multiple numbered copies
- `name` (optional): Name for added workflow (without `.md` extension)

### `update`

Update workflows from their source repositories and check for gh-aw updates.

- `workflows` (optional): Array of workflow IDs to update (empty for all)
- `major` (optional): Allow major version updates
- `force` (optional): Force update even if no changes detected

### `fix`

Apply automatic codemod-style fixes to workflow files.

- `workflows` (optional): Array of workflow IDs to fix (empty for all)
- `write` (optional): Write changes to files (default is dry-run)
- `list_codemods` (optional): List available codemods and exit

Available codemods: `timeout-minutes-migration`, `network-firewall-migration`, `sandbox-agent-false-removal`, `mcp-scripts-mode-removal`, `steps-run-secrets-to-env`.

## Using GH-AW as an MCP from an Agentic Workflow

Use the GH-AW MCP server from within a workflow to enable self-management (status checks, compilation, log analysis):

```yaml wrap
---
permissions:
  actions: read  # Required for agentic-workflows tool
tools:
  agentic-workflows:
---

Check workflow status, download logs, and audit failures.
```
