---
description: Complete reference for gh aw CLI commands and their MCP tool equivalents for restricted environments
---

# gh aw CLI Commands Reference

This document describes all `gh aw` commands, when to use the CLI versus the **agentic-workflows** MCP tool, and common debugging scenarios.

## CLI vs MCP Tool — When to Use Each

| Environment | Use |
|---|---|
| **Local development** (terminal with `gh` auth) | `gh aw <command>` CLI |
| **GitHub Copilot Cloud** (coding agent, Copilot Chat) | `agentic-workflows` MCP tool |
| **GitHub Actions workflow step** | `gh aw <command>` after installing `github/gh-aw/actions/setup-cli` |
| **CI runner without gh auth** | `agentic-workflows` MCP tool |

> [!NOTE]
> **agentic-workflows MCP tool availability**
>
> The MCP tool is available when `agentic-workflows:` is added to a workflow's `tools:` section. In Copilot Chat / Copilot coding agent, it is pre-configured and always available.
>
> In a GitHub Actions workflow step, install the CLI first:
> ```yaml
> - uses: github/gh-aw/actions/setup-cli@<version>
> - run: gh aw compile
> ```

---

## Command Reference

### `gh aw init`

Initialize a repository for agentic workflows.

```bash
gh aw init
```

Creates `.github/agents/agentic-workflows.agent.md` and supporting files.

**MCP equivalent**: Not available — run from a local terminal or use the `upgrade` tool for updates.

---

### `gh aw compile`

Compile workflow `.md` files into GitHub Actions `.lock.yml` files.

```bash
gh aw compile                     # Compile all workflows
gh aw compile <workflow-name>     # Compile a specific workflow
gh aw compile --strict            # Compile with strict mode validation
gh aw compile --validate          # Validate without emitting lock files
gh aw compile --fail-fast         # Stop at first error
gh aw compile --purge             # Remove orphaned .lock.yml files
gh aw compile --approve           # Approve new secrets / action changes
```

**MCP equivalent**: `compile` tool

```
Use the compile tool with workflow_name: "my-workflow"
```

---

### `gh aw run`

> [!IMPORTANT]
> **Always prefer `gh aw run` over `gh workflow run <file>.lock.yml`** — it handles workflow resolution by short name, validates inputs, and enables correct run-tracking with `gh aw audit` and `gh aw logs`.

Trigger a workflow on demand using `workflow_dispatch`. Capabilities:
- Workflow resolution by short name (no need to remember `.lock.yml`)
- Input parsing and validation against declared inputs
- Correct run-tracking so `gh aw audit` and `gh aw logs` work immediately after

```bash
gh aw run                           # Interactive mode — pick workflow and fill inputs
gh aw run <workflow-name>           # Run by short name
gh aw run <workflow-name>.md        # Alternative: explicit .md extension
gh aw run <workflow-name> --ref main              # Run on a specific branch/tag/SHA
gh aw run <workflow-name> --repeat 3              # Run 4 times total (1 + 3 repeats)
gh aw run <workflow-name> --input key=value       # Pass a specific input
```

**MCP equivalent**: Not available in the agentic-workflows MCP tool. If you cannot access the CLI, use the GitHub MCP server to dispatch the workflow:

```
Use the github MCP server tool "create_workflow_dispatch" with:
  - owner: <org>
  - repo: <repo>
  - workflow_id: <workflow-name>.lock.yml
  - ref: main
  - inputs: { ... }
```

---

### `gh aw logs`

Download and analyze workflow execution logs.

```bash
gh aw logs                          # Logs for all agentic workflows
gh aw logs <workflow-name>          # Logs for a specific workflow
gh aw logs <workflow-name> --json   # JSON output for programmatic use
gh aw logs --engine copilot         # Filter by engine
gh aw logs -c 10                    # Last 10 runs
gh aw logs --start-date -1w         # Last week's runs
gh aw logs --start-date 2024-01-01 --end-date 2024-01-31
gh aw logs -o ./workflow-logs       # Save to directory
```

**MCP equivalent**: `logs` tool

```
Use the logs tool with workflow_name: "my-workflow"
```

---

### `gh aw audit`

Investigate a specific workflow run in detail (missing tools, safe outputs, metrics).

```bash
gh aw audit <run-id>                # Audit a single run
gh aw audit <run-id> --json         # JSON output
gh aw audit <base-id> <compare-id>  # Diff two runs (regression detection)
gh aw audit <id1> <id2> <id3> --json  # Multi-run diff
```

**MCP equivalent**: `audit` tool (single run) / `audit-diff` tool (multi-run comparison)

```
Use the audit tool with run_id: 20135841934
```

---

### `gh aw status`

Show the status of all agentic workflows in the repository.

```bash
gh aw status
```

**MCP equivalent**: `status` tool

---

### `gh aw checks`

Show check run results for a workflow run.

```bash
gh aw checks <run-id>
```

**MCP equivalent**: `checks` tool

---

### `gh aw fix`

Apply automatic codemods to fix deprecated fields in workflow files.

```bash
gh aw fix                   # Preview changes (dry run)
gh aw fix --write           # Apply changes
```

**MCP equivalent**: `fix` tool

---

### `gh aw upgrade`

Upgrade the repository's agentic workflows configuration to the latest gh-aw version.

```bash
gh aw upgrade               # Upgrade agent files + codemods + compile
gh aw upgrade -v            # Verbose output
gh aw upgrade --no-fix      # Skip codemods and compilation
```

**MCP equivalent**: `upgrade` tool

---

### `gh aw add`

Add a new shared workflow component as an import.

```bash
gh aw add <workflow-url>
```

**MCP equivalent**: `add` tool

---

### `gh aw update`

Update imported shared workflow components.

```bash
gh aw update
```

**MCP equivalent**: `update` tool

---

### `gh aw mcp inspect`

Inspect and analyze MCP server configurations in workflows.

```bash
gh aw mcp inspect <workflow-name>
gh aw mcp inspect <workflow-name> --inspector   # Launch web-based inspector UI
gh aw mcp list                                   # List workflows with MCP servers
```

**MCP equivalent**: `mcp-inspect` tool

---

## Debugging Scenarios

For detailed debugging guidance, see **[debug-agentic-workflow.md](./debug-agentic-workflow.md)** (local) or the [canonical upstream](https://github.com/github/gh-aw/blob/main/.github/aw/debug-agentic-workflow.md).

### Scenario: Trigger a workflow and immediately audit it

```bash
# CLI
gh aw run my-workflow --ref main
# (gh aw run prints the run URL / ID)
gh aw audit <run-id> --json
```

```
# MCP tool (restricted environment)
Use the github tool "create_workflow_dispatch" to trigger my-workflow.lock.yml, then
use the audit tool with the returned run_id.
```

### Scenario: Investigate why a workflow failed

```bash
gh aw audit <run-id> --json
```

Focus on:
- `missing_tools` — tools the agent tried but couldn't call
- `safe_outputs.jsonl` — safe-output calls attempted
- Agent logs — the agent's own reasoning

### Scenario: Compare two runs for regressions

```bash
gh aw audit <baseline-run-id> <new-run-id> --json
```

### Scenario: Compile after editing the workflow

```bash
gh aw compile my-workflow        # Recompile
gh aw run my-workflow --ref main # Test the updated workflow
```

### Scenario: I'm in Copilot Cloud — how do I run a workflow?

The agentic-workflows MCP tool does not expose a `run` command. Use the GitHub MCP tool instead:

```
Use the github tool "create_workflow_dispatch" with:
  - owner: <org>
  - repo: <repo>
  - workflow_id: my-workflow.lock.yml
  - ref: main
```

Then immediately audit the run:

```
Use the audit tool with the run_id returned by the dispatch.
```

---

## MCP Tool ↔ CLI Quick Reference

| CLI command | MCP tool |
|---|---|
| `gh aw status` | `status` |
| `gh aw compile` | `compile` |
| `gh aw run` | *(use GitHub MCP `create_workflow_dispatch`)* |
| `gh aw logs` | `logs` |
| `gh aw audit` | `audit` |
| `gh aw audit <id1> <id2>` | `audit-diff` |
| `gh aw checks` | `checks` |
| `gh aw mcp inspect` | `mcp-inspect` |
| `gh aw add` | `add` |
| `gh aw update` | `update` |
| `gh aw fix` | `fix` |
| `gh aw upgrade` | `upgrade` |
| `gh aw init` | *(local only)* |
