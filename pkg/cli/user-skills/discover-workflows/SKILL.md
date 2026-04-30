---
name: discover-workflows
description: Use when the user wants to browse or discover available agentic workflows from the gh-aw catalog for the current repository. Lists installed workflows and suggests relevant ones from the catalog based on the repo's language and framework.
---

# Discover Agentic Workflows

Use this skill to help users explore available agentic workflows from the gh-aw catalog and identify which ones are suitable for their repository.

## Steps

1. Run `gh aw list` to show all workflows currently installed in the repository.
2. If the user wants catalog suggestions, inspect the repository language and framework signals (check `package.json`, `go.mod`, `requirements.txt`, `Gemfile`, etc.).
3. Run `gh aw add --list` (or refer to the gh-aw catalog at https://github.com/github/gh-aw) to enumerate available catalog workflows.
4. Propose 3–5 relevant workflows based on the repo's technology stack and the user's goals.
5. For each proposed workflow, briefly describe:
   - What trigger activates it (e.g., on issue creation, on schedule)
   - What the agent does (e.g., triage issues, improve code quality)
   - Any required secrets (e.g., `ANTHROPIC_API_KEY`)
6. Ask the user which workflow(s) they want to install, or if they want more details on any option.

## Hand back to the user for

- Final workflow selection — always confirm before installing
- Ambiguous use-cases where multiple workflows could apply — present options

## Example CLI commands used by this skill

```bash
# List installed workflows
gh aw list

# Inspect catalog (catalog names are passed to gh aw add)
gh aw add --list
```
