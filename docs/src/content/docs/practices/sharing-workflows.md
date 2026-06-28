---
title: Sharing Workflows in the Organization
description: Share, reuse, and govern workflows across repositories and organizations.
---

Sharing workflows across an organization involves several independent layers. Each layer can be adopted independently; teams do not need all of them at once.

The recommended enterprise pattern is to maintain one central `agentic-workflows` repository with versioned workflow templates and shared components. Consuming repositories then use `gh aw add` to install full workflows and `imports:` to pull in common modules.

## Sharing Layers

Organizations usually combine a few independent sharing mechanisms: install complete workflows with `gh aw add`, import shared modules with `imports:`, and choose a versioning strategy that matches how quickly consumers should receive updates.

### 1. Copy and install whole workflows

A repository can pull in a complete workflow from another repository:

```bash
gh aw add acme-org/agentic-workflows/ci-doctor@v1.2.0
```

The `source:` field is automatically added to the installed workflow's frontmatter so the origin and version are tracked. Use `gh aw add-wizard` for interactive installation with guided prompts. Use `gh aw add` for scripted or CI-driven installation.

See [Reusing Workflows](/gh-aw/guides/reusing-workflows/) for the full command reference and options.

### 2. Reusable workflow components

Shared building blocks — tool configurations, MCP server definitions, safety policies, and prompt snippets — can be imported into any workflow:

```yaml
imports:
  - acme-org/shared-workflows/shared/security-setup.md@v2.1.0
  - acme-org/shared-workflows/shared/mcp/tavily.md@v1.0.0
```

Remote imports are cached under `.github/aw/imports/` by commit SHA after the first fetch. This enables reproducible offline compilation and avoids redundant downloads when multiple refs point to the same commit.

See [Imports Reference](/gh-aw/reference/imports/) for path formats, merge semantics, and field-specific behavior.

### 3. Parameterized templates

Shared workflows that declare an `import-schema` accept runtime parameters via `uses`/`with`:

```yaml
imports:
  - uses: acme-org/shared-workflows/shared/reviewer.md@v1
    with:
      languages: ["go", "typescript"]
      severity: "high"
```

This lets a single shared component serve multiple consuming workflows with different configurations without requiring separate copies.

See [Imports Reference](/gh-aw/reference/imports/#calling-a-parameterized-shared-workflow) for schema declaration and validation details.

### 4. Versioning and update flow

Enterprise workflow sharing needs a clear versioning model:

| Ref type | Behavior |
| --- | --- |
| Exact release tag (`@v1.2.0`) | Pins to one immutable release until you change `source:` explicitly. |
| Moving release ref (`@v1`) | Follows newer compatible releases in that major line when you run `gh aw update`. |
| Branch ref (`@develop`) | Tracks the latest commit on a branch for development integration. |
| SHA pin (`@abc123def`) | Gives strict reproducibility and never moves without an explicit change. |

To pull upstream changes into an already-installed workflow:

```bash
gh aw update ci-doctor          # update one workflow
gh aw update                    # update all tracked workflows
```

Updates use a 3-way merge by default to preserve local edits. Use `--no-merge` to replace the local copy with the upstream version without merging. When the recorded `source:` uses a moving major ref such as `@v1`, `gh aw update` stays within that major line unless `--major` is passed.

### 5. Private and internal sharing controls

Not all workflows are safe to share across organizations. Use `private: true` in frontmatter to block installation into other repositories via `gh aw add`, rely on repository visibility to control discoverability, and keep org-internal catalogs in private or internal repositories so only authorized members can install them.

See [Private Workflows](/gh-aw/reference/frontmatter/#private-workflows-private) for configuration details.

### 6. Import caching and lock behavior

When a workflow is compiled, remote imports are resolved and locked. The compiled `.lock.yml` file records the exact commit SHA for every remote import, making runs reproducible regardless of upstream branch movement.

Imports are cached locally under `.github/aw/imports/` by commit SHA. Cached imports are used for all subsequent compilations until you explicitly update them. This means the lock file and the import cache together form the reproducibility guarantee for shared workflows.

### 7. Cross-repository execution model

Separate from sharing workflow definitions, workflows can operate across repositories at runtime by reading files and metadata from other repositories, checking out target code for analysis or modification, and writing safe outputs to target repositories with explicit authentication and allowlists.

```yaml
safe-outputs:
  create-issue:
    target-repo: "acme-org/target-repo"
    allowed-repos: ["acme-org/repo1", "acme-org/repo2"]
```

Cross-repository operations require appropriate GitHub token permissions and explicit `allowed-repos` declarations. See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for authentication, permissions, and safe output configuration.

## Recommended Enterprise Pattern

For most organizations, one central `agentic-workflows` repository should hold versioned workflow templates and shared components under `workflows/` and `shared/`. Consuming repositories install complete workflows with `gh aw add acme-org/agentic-workflows/<workflow>@<version>`, import common modules such as MCP configurations and safety policies through `imports:`, use tags for stable production consumers and branches for development integration, and mark internal-only workflows with `private: true`.

This model gives platform teams centralized ownership and update control while giving consuming teams reproducibility through version pins and the ability to preserve local customizations through 3-way merge.

## Governance Questions

When workflows are shared across an organization, the important decisions are usually operational rather than technical: who owns the source workflow and reviews changes, how updates are tested and promoted, which repositories may consume or dispatch shared workflows, how secrets and permissions are standardized, and when a team may fork instead of staying on the shared version. Those decisions affect reliability more than the file format does.

## Related Documentation

- [Reusing Workflows](/gh-aw/guides/reusing-workflows/)
- [Imports Reference](/gh-aw/reference/imports/)
- [Cross-Repository Operations](/gh-aw/reference/cross-repository/)
- [Private Workflows](/gh-aw/reference/frontmatter/#private-workflows-private)
- [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/)
