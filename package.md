# Configure Repository as Agentic Workflows Package

This prompt guides you, a coding agent, to convert the current repository (unless the request explicitly specifies a different target repository) that contains **agentic-workflows** and/or **shared agentic-workflows** into a reusable package repository.

## Your Task

Configure the repository so others can install workflows from it with `gh aw add` by:

1. Standardizing package structure when needed
2. Creating an `aw.yml` repository package manifest at the package root
3. Including all package workflows in `aw.yml`
4. Generating a clear package description
5. Updating `README.md` with installation instructions for consumers

## Step 1: Discover Package Contents

Identify all workflow markdown files and classify them:

- Agentic workflows (`.github/workflows/*.md`, excluding lock files)
- Shared workflows (`workflows/*.md`)
- Shared workflow assets used by those workflows

Also detect:

- Multiple potential package roots (repo root, nested folders, or both)
- Existing `workflows/` folders in one or more locations
- Repo-specific dependencies (hardcoded repo names, labels, branches, teams, secrets, file paths, or assumptions)

## Step 2: Standardize Structure

If structure is inconsistent, organize it into one clear package root:

- Keep installable shared workflows in `workflows/`
- Keep repository-owned runnable workflows in `.github/workflows/` only when they are intentionally repo-local
- Avoid duplicate or conflicting copies across multiple subfolders
- If the repo has multiple candidate package folders, choose one canonical package root and document that choice in README

Do not break existing references; update relative imports/paths when files move.

## Step 3: Create `aw.yml` Package Manifest

Create `aw.yml` in the package root using the supported manifest format:

```yaml
manifest-version: "1"
min-version: v0.88.5
name: Repo Assist
description: Reusable agentic workflows for <domain/use-case>.
emoji: 🤖
includes:
  - workflows/*
resources:
  - source: templates/bug.yml
    destination: .github/ISSUE_TEMPLATE/bug.yml
```

Requirements:

- `manifest-version`: use `"1"` (or omit and rely on default `"1"`)
- `min-version`: use `v0.88.5` or newer when `includes` contains a trailing `/*` wildcard
- `name`: human-readable package name
- `description`: concise and relevant to the actual workflows
- `emoji`: optional package emoji (string)
- `includes`: installable package entries; use `workflows/*` when every supported direct child belongs in the package
- `resources`: optional package-root-relative assets copied to allowlisted destinations such as `.github/ISSUE_TEMPLATE/*.yml`, `.github/CODEOWNERS`, or `.github/aw/**`
- Include paths must be package-root-relative. Trailing `/*` wildcards are non-recursive and are supported only for workflow, skill, and agent directories such as `workflows/*`, `skills/*`, and `agents/*`; `files/*` is not a supported include path.
- Use `.github/workflows/*` only when every matching repository workflow is intentionally redistributable.

Do not invent custom package metadata fields.

### Minimal manifest (caveman optimization)

Keep package manifest simple:

- Use only `manifest-version`, conditional `min-version`, `name`, `description`, optional `emoji`, and `includes`
- Keep `description` short
- Prefer `workflows/*` when all supported direct children are installable; otherwise list the intended entries explicitly in `includes`

Documentation links:

- https://github.github.com/gh-aw/reference/repository-package-manifest/
- https://github.github.com/gh-aw/specs/repository-package-manifest-specification/

## Step 4: Dependency Cleanup for Reusability

Make workflows generic and package-ready:

- Replace hardcoded repository-specific values with parameters, inputs, or neutral defaults
- Identify required external services/tools and clearly declare them
- Remove references that only work in the source repository unless explicitly required
- Ensure workflows can be consumed from another repository without manual path rewrites

If full cleanup is too large for one PR, include a prioritized follow-up list in README.

## Step 5: Update `README.md` for Consumers

Update the package README to include:

1. What this package provides
2. Prerequisites/dependencies
3. Exact install command(s), for example:
   - `gh aw add owner/repo`
   - `gh aw add owner/repo/path/to/package` (for nested packages)
4. How to compile/use the added workflows
5. Any required configuration after installation

The README instructions must be clear enough that another repository can adopt the package without guesswork.

## Step 6: Validate and Deliver

Before finishing:

- Verify `aw.yml` exists and is valid YAML
- Verify each explicit `includes` path exists and each wildcard selects only intended direct children
- Confirm README instructions match the final package structure
- Summarize what was standardized, what dependencies were cleaned up, and any remaining follow-up items
