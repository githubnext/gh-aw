---
name: compile-workflows
description: Use when the user wants to compile agentic workflow Markdown files into GitHub Actions YAML lock files, or when compilation errors need to be diagnosed and fixed.
---

# Compile Agentic Workflows

Use this skill to compile `.md` workflow files into `.lock.yml` GitHub Actions workflow files, and to surface and explain any compilation errors.

## Steps

1. Run `gh aw compile` to compile all workflows in `.github/workflows/`.
   - To compile a specific workflow, run `gh aw compile <workflow-name>`.
2. If compilation succeeds, report which `.lock.yml` files were generated or updated.
3. If compilation fails:
   a. Read the error output carefully — errors include file path, line number, and a description.
   b. Open the relevant `.md` file and identify the problematic frontmatter field or expression.
   c. Explain the error to the user in plain language and propose a fix.
   d. Apply the fix (with user approval for significant changes).
   e. Re-run `gh aw compile` to confirm the fix resolved the issue.
4. If there are validation warnings (non-fatal), summarize them and suggest whether they should be addressed.

## Common error categories

- **Schema validation errors** — invalid or missing frontmatter fields; fix by correcting the field value or adding the required key.
- **Expression errors** — oversized expressions or invalid GitHub Actions syntax; simplify the expression.
- **Secret/permission errors** — missing required secrets or invalid permission combinations; add secrets via `gh secret set`.
- **Network/image validation errors** — invalid container image references or blocked domains; correct the image tag or add the domain to the `network.allowed` list.

## Hand back to the user for

- Fixes that require changing business logic in the workflow prompt
- Secret values that need to be set interactively

## Example CLI commands used by this skill

```bash
# Compile all workflows
gh aw compile

# Compile a specific workflow
gh aw compile <workflow-name>

# Compile with detailed validation
gh aw compile --validate

# Auto-fix known issues before compiling
gh aw compile --fix
```
