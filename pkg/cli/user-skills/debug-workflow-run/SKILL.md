---
name: debug-workflow-run
description: Use when the user wants to debug a failed or misbehaving agentic workflow run by fetching raw logs, identifying the root cause, and proposing a concrete fix.
---

# Debug a Failed Workflow Run

Use this skill to fetch logs for a failed agentic workflow run, diagnose the root cause, and propose a concrete fix.

## Steps

1. Obtain the workflow run ID or URL from the user. If not provided, run `gh run list --status failure --limit 10` to list recent failed runs and ask the user to pick one.
2. Run `gh aw logs <run-id-or-url>` to download and display the structured log output.
   - For detailed raw logs, use `gh aw logs <run-id-or-url> --verbose`.
3. Identify the failure category from the logs:
   - **Agent error** — the AI agent returned an error or exceeded the token/time limit
   - **Tool error** — an MCP tool call failed (authentication, network, permissions)
   - **Action error** — a GitHub Actions step failed (shell command, Docker pull, etc.)
   - **Firewall block** — the agent tried to access a domain not in the `network.allowed` list
   - **Safe-output failure** — the agent's output was rejected by the safe-output guard
4. Explain the root cause to the user in plain language, referencing specific log lines where possible.
5. Propose a concrete fix:
   - For agent errors: simplify the prompt, reduce scope, or increase `timeout-minutes`
   - For tool errors: check secrets (`gh secret list`), verify MCP server config
   - For firewall blocks: add the blocked domain to the workflow's `network.allowed` list and recompile
   - For safe-output failures: review the agent's output against the safe-output schema
6. Apply the fix (with user approval) and suggest re-running the workflow: `gh aw run <workflow-name>`.

## Hand back to the user for

- Setting secrets identified as missing (`gh secret set <NAME>`)
- Approving changes to the workflow prompt or frontmatter
- Auth flows required to re-authenticate MCP servers

## Example CLI commands used by this skill

```bash
# List recent failed runs
gh run list --status failure --limit 10

# Fetch structured logs for a specific run
gh aw logs <run-id>

# Fetch verbose raw logs
gh aw logs <run-id> --verbose

# Re-run the workflow after a fix
gh aw run <workflow-name>

# Compile after making frontmatter changes
gh aw compile <workflow-name>
```
