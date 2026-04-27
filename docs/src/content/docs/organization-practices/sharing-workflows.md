---
title: Sharing Workflows
description: Share, reuse, and govern workflows across repositories and organizations.
sidebar:
  badge: { text: 'Platform', variant: 'tip' }
---

:::caution[Evolving guidance]
Enterprise workflow sharing capabilities are actively expanding. Details in this guide may change as the platform matures.
:::

Sharing workflows across repositories is an organization practice, not a single design pattern. GitHub Agentic Workflows supports multiple layers of sharing, from installing a complete workflow into a repository to parameterized imports and cross-repository execution.

The recommended enterprise pattern is one central `agentic-workflows` repository that publishes versioned workflow templates and shared components. Consuming repositories install full workflows with `gh aw add` and pull in shared modules through `imports:`.

## Sharing Layers

### Layer 1: Copy or install whole workflows

A repository can pull in a complete workflow from another repository using `gh aw add`:

```bash
gh aw add acme-org/agentic-workflows/ci-doctor@v1.2.0
```

`gh aw add-wizard` provides interactive guidance for the same operation. When a workflow is installed, a `source:` field is added to its frontmatter so the origin is tracked. Updates can then be applied later with `gh aw update`.

Version references support semantic tags (`@v1.2.0`), branches (`@main`), and commit SHAs for strict pinning.

### Layer 2: Reusable workflow components

Shared pieces such as common MCP server configuration, security setup steps, or reusable prompt fragments can be imported by any workflow:

```yaml
imports:
  - acme-org/shared-workflows/shared/security-setup.md@v2.1.0
  - acme-org/shared-workflows/shared/mcp/tavily.md@v1.0.0
```

Imports compose into the consuming workflow at compile time. Frontmatter fields such as `tools:`, `network:`, and `mcp-servers:` are merged so imported configuration is additive.

### Layer 3: Parameterized templates

Shared workflows can accept inputs so the same template is usable across teams with different requirements:

```yaml
imports:
  - uses: acme-org/shared-workflows/shared/reviewer.md@v1
    with:
      languages: ["go", "typescript"]
      severity: "high"
```

The `uses` / `with` syntax makes it possible to share workflows that have team-specific settings while keeping a single maintained source.

### Layer 4: Versioning and update flow

Enterprise sharing depends on a predictable versioning model:

- **Semantic versions** (`@v1.2.0`) for stable workflows that consuming teams can pin.
- **Branch refs** (`@main`, `@develop`) for pre-release versions during active development.
- **SHA pins** for strict reproducibility when drift must be ruled out.

Use `gh aw update` to pull upstream changes into installed workflows:

```bash
gh aw update                        # update all tracked workflows
gh aw update ci-doctor              # update a specific workflow
```

Updates apply a three-way merge that preserves local edits while incorporating upstream changes.

### Layer 5: Private and internal sharing controls

Not every workflow should be available for installation everywhere. GitHub Agentic Workflows supports access-based controls:

- **`private: true`** in workflow frontmatter blocks `gh aw add` from installing that workflow into other repositories.
- Repository and organization visibility settings control who can read the workflow sources at all.
- `gh aw add` performs access checks before installation and surfaces warnings for workflows from untrusted sources.
- Org-internal workflow catalogs can be created using organization repositories with appropriate visibility settings.

```yaml
---
private: true
---
```

### Layer 6: Import caching and lock behavior

Remote imports are resolved at compile time and cached in `.github/aw/imports/` by commit SHA. This means:

- Compiled `.lock.yml` files are fully reproducible: the exact import content is pinned at compile time.
- Offline compilation works once imports have been downloaded.
- The SHA cache is shared across refs that resolve to the same commit, reducing redundant network calls.

The `.lock.yml` file and the `.github/aw/imports/` directory should both be committed to the repository so workflow runs are reproducible across environments.

### Layer 7: Cross-repository execution model

Separate from sharing workflow definitions, workflows can operate across repositories at runtime:

- Read other repositories using GitHub tool access configured with appropriate permissions.
- Check out code from other repositories using cross-repository checkout.
- Create safe outputs (issues, pull requests, comments) in target repositories using `target-repo` and `allowed-repos`.
- Explicit authentication (PAT or GitHub App token) and allowlists control which repositories a workflow may write to.

This execution model is covered in detail in [Cross-Repository Workflows](/gh-aw/reference/cross-repository/) and [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/).

## Governance Questions

When workflows are shared across an organization, the important questions are usually operational rather than technical:

- Who owns the source workflow and approves changes.
- How updates are reviewed and promoted from the central repository to consuming repositories.
- Which repositories may consume or dispatch to shared workflows.
- How secrets, permissions, and safe outputs are standardized across teams.
- When teams may fork a workflow rather than stay on the shared source.

Those decisions affect reliability more than the file format does.

## Related Documentation

- [Reusing Workflows](/gh-aw/guides/packaging-imports/)
- [Imports Reference](/gh-aw/reference/imports/)
- [Frontmatter Reference](/gh-aw/reference/frontmatter/) (source, private, resources fields)
- [Cross-Repository Workflows](/gh-aw/reference/cross-repository/)
- [SideRepoOps](/gh-aw/patterns/side-repo-ops/)
- [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/)
