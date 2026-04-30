---
name: install-workflow
description: Use when the user wants to install an agentic workflow from the gh-aw catalog into the current repository. Handles workflow selection, secret wiring (including CLAUDE_CODE_OAUTH_TOKEN for Claude Pro/Max subscribers), and initial compile verification.
---

# Install an Agentic Workflow

Use this skill to install a workflow from the gh-aw catalog into the current repository.

## Steps

1. If the user hasn't named a workflow, run `gh aw list` and then `gh aw add --list` to show available catalog workflows; propose 3 candidates based on repo language/framework signals.
2. Confirm the workflow choice with the user before proceeding.
3. Run `gh aw add <name>` to install the chosen workflow.
4. Parse required secrets from the installed workflow's frontmatter (look for a `secrets:` block in the generated `.github/workflows/<name>.md`).
5. For each required secret, run `gh secret list` to check whether it is already set. For missing secrets, instruct the user to run:
   ```bash
   gh secret set <SECRET_NAME>
   ```
6. Run `gh aw compile` to verify the `.lock.yml` generates cleanly.
7. Summarize: workflow installed, secrets set (or still needed), recommended next step (push or trigger first run with `gh aw run <name>`).

## Hand back to the user for

- Interactive secret entry — `gh secret set` prompts for the value; the agent must not handle secret values directly
- Ambiguous workflow choices — always confirm, never guess
- Auth flows (`claude setup-token`, `gh auth login`, etc.)

## Example CLI commands used by this skill

```bash
# List installed workflows
gh aw list

# Browse the catalog
gh aw add --list

# Install a specific workflow
gh aw add <workflow-name>

# Check which secrets are already set
gh secret list

# Set a required secret (user will be prompted for the value)
gh secret set ANTHROPIC_API_KEY

# Compile to verify everything is in order
gh aw compile
```
