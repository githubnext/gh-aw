---
name: audit-workflows
description: Use when the user wants to audit a completed agentic workflow run to understand what happened, diagnose failures, review tool usage, and get actionable fix suggestions.
---

# Audit Agentic Workflow Runs

Use this skill to audit a completed (or failed) agentic workflow run and provide a clear summary of what the agent did, what went wrong, and how to fix it.

## Steps

1. Obtain the workflow run ID or URL from the user. If not provided, run `gh run list --workflow <workflow-name>.lock.yml --limit 5` to show recent runs and ask the user to pick one.
2. Run `gh aw audit <run-id-or-url>` to fetch and analyze the run.
3. Parse the audit output:
   - **Overview** — Did the run succeed or fail? What was the agent's objective?
   - **Tool usage** — Which tools did the agent invoke? Were any calls blocked by the firewall?
   - **Errors** — What errors occurred? Include file paths, line numbers, and error messages where available.
   - **Safe-output chains** — Did the agent produce any outputs (PRs, issues, comments)?
4. Summarize findings in a concise report:
   - One-sentence verdict (success / partial success / failure)
   - Top 3 issues (if any) with proposed fixes
   - Recommended next step
5. If the audit reveals a fixable issue (e.g., a missing secret, a blocked domain, an incorrect workflow config), walk the user through the fix.

## Hand back to the user for

- Approving changes to the workflow's frontmatter or prompt
- Setting secrets identified as missing
- Triggering a re-run after a fix: `gh aw run <workflow-name>`

## Example CLI commands used by this skill

```bash
# Audit the most recent run of a workflow
gh run list --workflow <workflow-name>.lock.yml --limit 5
gh aw audit <run-id>

# Audit using a GitHub Actions run URL
gh aw audit https://github.com/<owner>/<repo>/actions/runs/<run-id>
```
