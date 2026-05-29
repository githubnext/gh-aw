---
title: Debugging Workflows
description: How to run, debug, and investigate agentic workflow failures using the Copilot CLI, gh aw audit, and log analysis.
sidebar:
  order: 250
---

This guide shows you how to debug agentic workflow failures on **github.com** using the Copilot CLI, the `gh aw` debugging commands, and manual investigation techniques.

> [!TIP]
> The fastest path to a fix is to let an AI agent debug it for you. Launch the Copilot CLI, invoke the `agentic-workflows` skill, and paste the failing run URL.

## Debugging with the Copilot CLI

The Copilot CLI is the recommended first step: it audits logs, traces failures, and suggests fixes interactively.

Launch `copilot`, invoke **agentic-workflows** to enable `gh aw audit`, `gh aw logs`, and related debugging tools. Then paste the failing run URL:

```text
Debug this workflow run: https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

Copilot downloads the logs, identifies the root cause (missing tools, permission errors, network blocks), and suggests fixes or opens a PR. Follow-up questions like "What domains were blocked?" or "Why did the MCP server fail?" work too.

### Alternative entry points

- **Copilot Chat on GitHub.com** (requires [agentic authoring setup](/gh-aw/guides/agentic-authoring/)): `agentic-workflows debug <run-url>`
- **Any coding agent**: paste this prompt to install `gh aw` and run the standalone analysis:

  ```text
  Debug this workflow run using https://raw.githubusercontent.com/github/gh-aw/main/debug.md

  The failed workflow run is at https://github.com/OWNER/REPO/actions/runs/RUN_ID
  ```

## Debugging with CLI Commands

### Auditing a Specific Run

`gh aw audit` breaks down a single run — failure analysis and root cause, behavior fingerprint (network/tool/cost profile), tool usage, MCP server status, firewall analysis, token/cost metrics, and safe-outputs. Accepts a run ID, run URL, job URL (extracts first failing step), or step URL.

```bash
gh aw audit 12345678
gh aw audit https://github.com/OWNER/REPO/actions/runs/123/job/456#step:7:1
gh aw audit 12345678 --parse                          # parse to markdown
gh aw audit 12345678 12345679                         # compare two runs
gh aw audit 12345678 12345679 --format markdown
```

For security and performance trends across multiple runs, use `gh aw logs --format`:

```bash
gh aw logs my-workflow --format markdown --count 10
gh aw logs my-workflow --format markdown --last 5 --json
```

See [Audit Commands](/gh-aw/reference/audit/) for complete flag documentation.

### Analyzing Workflow Logs

`gh aw logs` downloads and analyzes logs across multiple runs (tool usage, network patterns, errors). Results are cached for 10–100× speedup on later runs.

```bash
gh aw logs my-workflow
gh aw logs my-workflow -c 10 --start-date -1w   # filter by count and date
gh aw logs my-workflow --firewall               # include firewall analysis
gh aw logs my-workflow --safe-output            # include safe-output details
gh aw logs my-workflow --json                   # JSON for scripting
```

### Checking Workflow Health

`gh aw health` reports workflow status across all workflows in a repository.

### Inspecting MCP Configuration

If you suspect MCP server issues, inspect the compiled configuration:

```bash
gh aw mcp list                              # list workflows with MCP servers
gh aw mcp inspect my-workflow               # inspect a workflow
gh aw mcp inspect my-workflow --inspector   # web-based inspector
```

## Common Errors

### "Authentication failed"

The Copilot token is missing, expired, or lacks the required permissions. Confirm you have an active Copilot subscription, that the token has **Copilot Requests** permission (fine-grained PATs), and that `gh auth status` reports it valid. See [Authentication Reference](/gh-aw/reference/auth/).

### "Tool not found" or Missing Tool Calls

The workflow references a tool that isn't configured or the MCP server failed to connect. Verify the `tools:` section in frontmatter, check the MCP server version, then run `gh aw mcp inspect my-workflow` and `gh aw audit <run-id>` to compare available vs. requested tools.

### Network / Firewall Blocks

```text
DENIED CONNECT registry.npmjs.org:443
```

The agent reached a domain outside the firewall allow-list. Add it explicitly, or use an ecosystem shorthand:

```aw
network:
  allowed:
    - defaults
    - node        # npm, yarn, pnpm registries
    - python      # PyPI, conda registries
    - registry.npmjs.org
```

See [Network Configuration](/gh-aw/guides/network-configuration/) for common domain configurations.

### Safe-Outputs Not Creating Issues / Comments

The safe-outputs job failed, the agent didn't produce the expected output, or permissions are missing. Inspect the safe-outputs section in `gh aw audit <run-id>` and review the [Safe Outputs Reference](/gh-aw/reference/safe-outputs/).

### Compilation Errors

The frontmatter has schema validation errors or unsupported fields. Use `--verbose` to diagnose, `gh aw fix --write` to auto-correct, and `--validate` to check without writing the lock file. See [Error Reference](/gh-aw/troubleshooting/errors/) for specific messages.

```bash
gh aw compile my-workflow --verbose
gh aw fix --write
gh aw compile --validate
```

## Advanced Debugging

### Enable Debug Logging

`DEBUG` enables detailed internal logging for any `gh aw` command. Output goes to `stderr` — capture it with `2>&1 | tee debug.log`.

```bash
DEBUG=* gh aw compile my-workflow              # all logs
DEBUG=cli:* gh aw audit 12345678               # CLI-specific
DEBUG=workflow:*,cli:* gh aw compile           # multiple packages
```

### Enable GitHub Actions Debug Logging

Add an `ACTIONS_STEP_DEBUG` repository secret set to `true` (**Settings → Secrets and variables → Actions**), then re-run the workflow for verbose step-level logging in the Actions UI.

### Inspecting Firewall Logs

Workflow run artifacts include `sandbox/firewall/logs/access.log`. Each line shows allowed (`TCP_TUNNEL`) or blocked (`DENIED`) traffic:

```text
TCP_TUNNEL/200 api.github.com:443
DENIED CONNECT blocked-domain.com:443
```

Or use the CLI: `gh aw logs my-workflow --firewall`, or `gh aw audit <run-id>` for combined firewall analysis.

### Inspecting Artifacts

Workflow runs produce several artifacts useful for debugging:

| Artifact | Location | Contents |
|----------|----------|----------|
| `prompt.txt` | `/tmp/gh-aw/aw-prompts/` | Full prompt sent to the AI agent |
| `agent_output.json` | `/tmp/gh-aw/safeoutputs/` | Structured safe-output data |
| `agent-stdio.log` | `/tmp/gh-aw/` | Raw agent stdin/stdout log |
| `firewall-logs/` | `/tmp/gh-aw/firewall-logs/` | Network access logs |

Download from the Actions run page or `gh run download <run-id> --repo OWNER/REPO`.

### Recompiling for a Quick Fix

After editing the `.md` file, recompile, commit both files, and push:

```bash
gh aw compile my-workflow
git add .github/workflows/my-workflow.md .github/workflows/my-workflow.lock.yml
git commit -m "fix: update workflow configuration"
git push
```
