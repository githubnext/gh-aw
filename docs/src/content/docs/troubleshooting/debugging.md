---
title: Debugging Workflows
description: How to run, debug, and investigate agentic workflow failures using the Copilot CLI, gh aw audit, and log analysis.
sidebar:
  order: 250
---

This guide shows you how to debug agentic workflow failures on **github.com** using the Copilot CLI, the `gh aw` debugging commands, and manual investigation techniques.

> [!TIP]
> The fastest path to a fix is to let an AI agent debug it for you. Launch the Copilot CLI, load the agentic-workflows agent, and paste the failing run URL.

## Debugging with the Copilot CLI

The Copilot CLI can audit logs, trace failures, and suggest fixes interactively. This is the recommended first step for any workflow failure.

### Step 1: Launch the Copilot CLI

```bash
copilot
```

### Step 2: Load the Agentic Workflows Agent

Once inside the Copilot CLI, run:

```text
/agent
```

Select **agentic-workflows** from the list. This gives Copilot access to the `gh aw audit`, `gh aw logs`, and other debugging tools.

### Step 3: Ask Copilot to Debug the Failure

Paste the failing run URL and ask Copilot to investigate:

```text
Debug this workflow run: https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

Copilot will:

- Download and audit the run logs
- Identify the root cause (missing tools, permission errors, network blocks, etc.)
- Suggest targeted fixes or open a pull request with the fix

You can also ask follow-up questions:

```text
What domains were blocked by the firewall?
Show me the safe-outputs from this run.
Why did the MCP server fail to connect?
```

### Alternative: Copilot Chat on GitHub.com

If your repository is [configured for agentic authoring](/gh-aw/guides/agentic-authoring/), you can use Copilot Chat directly on GitHub.com:

```text
/agent agentic-workflows debug https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

### Alternative: Any Coding Agent

For coding agents that don't have the agentic-workflows agent pre-configured, use the standalone debug prompt:

```text
Debug this workflow run using https://raw.githubusercontent.com/github/gh-aw/main/debug.md

The failed workflow run is at https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

The agent will install `gh aw`, analyze logs, identify the root cause, and suggest a fix.

## Debugging with CLI Commands

### Auditing a Specific Run

`gh aw audit` gives a comprehensive breakdown of a single run — overview, metrics, tool usage, MCP failures, firewall analysis, behavior fingerprint, and artifacts:

```bash
# By run ID
gh aw audit 12345678

# By full URL
gh aw audit https://github.com/OWNER/REPO/actions/runs/12345678

# By job URL (extracts first failing step)
gh aw audit https://github.com/OWNER/REPO/actions/runs/123/job/456

# By step URL (extracts a specific step)
gh aw audit https://github.com/OWNER/REPO/actions/runs/123/job/456#step:7:1

# Parse to markdown for sharing
gh aw audit 12345678 --parse
```

Audit output includes:

- **Failure analysis** with error summary and root cause
- **Behavior fingerprint** — multi-dimensional characterization of the run's network, tool, and cost profile
- **Tool usage** — which tools were called, which failed, and why
- **MCP server status** — connection failures, timeout errors, and per-server health
- **Firewall analysis** — blocked domains, allowed traffic, and policy attribution
- **Token/cost metrics** — per-run inference spend and token usage
- **Safe-outputs** — structured outputs the agent produced

To compare behavior between two runs and detect regressions across firewall, MCP, and metrics dimensions, use `audit diff`:

```bash
gh aw audit diff 12345678 12345679
gh aw audit diff 12345678 12345679 --format markdown
```

For security and performance trends across multiple runs, use `gh aw logs --format`:

```bash
gh aw logs my-workflow --format markdown --count 10
gh aw logs my-workflow --format markdown --last 5 --json
```

See [Audit Commands](/gh-aw/reference/audit/) for complete flag documentation.

### Analyzing Workflow Logs

`gh aw logs` downloads and analyzes logs across multiple runs with tool usage, network patterns, errors, and warnings:

```bash
# Download logs for a workflow
gh aw logs my-workflow

# Filter by count and date range
gh aw logs my-workflow -c 10 --start-date -1w

# Include firewall analysis
gh aw logs my-workflow --firewall

# Include safe-output details
gh aw logs my-workflow --safe-output

# JSON output for scripting
gh aw logs my-workflow --json
```

Results are cached locally for 10–100× speedup on subsequent runs.

### Checking Workflow Health

`gh aw health` gives a quick overview of workflow status across all workflows in a repository:

```bash
gh aw health
```

### Inspecting MCP Configuration

If you suspect MCP server issues, inspect the compiled configuration:

```bash
# List all workflows with MCP servers
gh aw mcp list

# Inspect MCP servers for a specific workflow
gh aw mcp inspect my-workflow

# Open the web-based MCP inspector
gh aw mcp inspect my-workflow --inspector
```

## Common Errors

### "Authentication failed"

```text
Error: Authentication failed
Your GitHub token may be invalid, expired, or lacking the required permissions.
```

**Cause**: The Copilot token is missing, expired, or lacks required permissions.

**Fix**:

1. Verify you have an active Copilot subscription
2. Check that the token has the **Copilot Requests** permission (for fine-grained PATs)
3. If using a custom `COPILOT_GITHUB_TOKEN`, verify it's valid:

   ```bash
   gh auth status
   ```

4. See [Authentication Reference](/gh-aw/reference/auth/) for token setup details

### "Tool not found" or Missing Tool Calls

**Cause**: The workflow references a tool that isn't configured or the MCP server failed to connect.

**Fix**:

1. Run `gh aw mcp inspect my-workflow` to verify tool configuration
2. Check that the MCP server version is compatible
3. Ensure `tools:` section in frontmatter includes the required tool
4. Run `gh aw audit <run-id>` to see which tools were available vs. requested

### Network / Firewall Blocks

```text
DENIED CONNECT registry.npmjs.org:443
```

**Cause**: The agent tried to reach a domain not in the firewall allow-list.

**Fix**: Add the domain to the `network.allowed` list in your workflow frontmatter:

```aw
network:
  allowed:
    - defaults
    - registry.npmjs.org
```

Or use an ecosystem shorthand:

```aw
network:
  allowed:
    - defaults
    - node        # Adds npm, yarn, pnpm registries
    - python      # Adds PyPI, conda registries
```

See [Network Configuration](/gh-aw/guides/network-configuration/) for common domain configurations.

### Safe-Outputs Not Creating Issues / Comments

**Cause**: The safe-outputs job failed, the agent didn't produce the expected output, or permissions are missing.

**Fix**:

1. Run `gh aw audit <run-id>` and check the safe-outputs section
2. See [Safe Outputs Reference](/gh-aw/reference/safe-outputs/) for configuration details

### Compilation Errors

**Cause**: The workflow frontmatter has schema validation errors or unsupported fields.

**Fix**:

1. Run the compiler with verbose output:

   ```bash
   gh aw compile my-workflow --verbose
   ```

2. Run the fixer for auto-correctable issues:

   ```bash
   gh aw fix --write
   ```

3. Validate without compiling:

   ```bash
   gh aw compile --validate
   ```

4. See [Error Reference](/gh-aw/troubleshooting/errors/) for specific error messages

## Advanced Debugging

### Enable Debug Logging

The `DEBUG` environment variable enables detailed internal logging for any `gh aw` command:

```bash
# All debug logs
DEBUG=* gh aw compile my-workflow

# CLI-specific logs
DEBUG=cli:* gh aw audit 12345678

# Workflow compilation logs
DEBUG=workflow:* gh aw compile my-workflow

# Multiple packages
DEBUG=workflow:*,cli:* gh aw compile my-workflow
```

> [!TIP]
> Debug output goes to `stderr`. Capture it with `2>&1 | tee debug.log`.

### Enable GitHub Actions Debug Logging

Set the `ACTIONS_STEP_DEBUG` secret to `true` in your repository to enable verbose step-level logging in GitHub Actions:

1. Go to **Settings → Secrets and variables → Actions**
2. Add a secret: `ACTIONS_STEP_DEBUG` = `true`
3. Re-run the workflow

This produces much more detailed logs in the Actions UI.

### Inspecting Firewall Logs

Download the workflow run artifacts and look for `sandbox/firewall/logs/access.log`. Each line shows whether a domain was allowed (`TCP_TUNNEL`) or blocked (`DENIED`):

```text
TCP_TUNNEL/200 api.github.com:443
DENIED CONNECT blocked-domain.com:443
```

You can also use the CLI:

```bash
gh aw logs my-workflow --firewall
gh aw audit <run-id>   # Includes firewall analysis
```

### Inspecting Artifacts

Workflow runs produce several artifacts useful for debugging:

| Artifact | Location | Contents |
|----------|----------|----------|
| `prompt.txt` | `/tmp/gh-aw/aw-prompts/` | The full prompt sent to the AI agent |
| `agent_output.json` | `/tmp/gh-aw/safeoutputs/` | Structured safe-output data |
| `agent-stdio.log` | `/tmp/gh-aw/` | Raw agent stdin/stdout log |
| `firewall-logs/` | `/tmp/gh-aw/firewall-logs/` | Network access logs |

Download artifacts from the GitHub Actions run page or via the CLI:

```bash
gh run download <run-id> --repo OWNER/REPO
```

### Recompiling for a Quick Fix

If you've identified the issue and made a change to the `.md` file, recompile and push:

```bash
gh aw compile my-workflow
git add .github/workflows/my-workflow.md .github/workflows/my-workflow.lock.yml
git commit -m "fix: update workflow configuration"
git push
```
