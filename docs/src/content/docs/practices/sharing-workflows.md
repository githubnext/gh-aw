---
title: Sharing Workflows
description: Share, reuse, and govern workflows across repositories and organizations.
---

Sharing workflows across an organization involves several independent layers. Each layer can be adopted independently; teams do not need all of them at once.

The recommended enterprise pattern is to maintain one central `agentic-workflows` repository with versioned workflow templates and shared components. Consuming repositories then use `gh aw add` to install full workflows and `imports:` to pull in common modules.

## Sharing Layers

### 1. Copy and install whole workflows

A repository can pull in a complete workflow from another repository:

```bash
gh aw add acme-org/agentic-workflows/ci-doctor@v1.2.0
```

The `source:` field is automatically added to the installed workflow's frontmatter so the origin and version are tracked. Use `gh aw add-wizard` for interactive installation with guided prompts. Use `gh aw add` for scripted or CI-driven installation.

See [Reusing Workflows](/gh-aw/guides/packaging-imports/) for the full command reference and options.

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

- **Exact release tags** (`@v1.2.0`) pin to a specific immutable release. They do not move on their own, so `gh aw update` will keep fetching that same tagged version unless you change the `source:` ref explicitly.
- **Moving release refs** (`@v1`) follow the latest compatible release within that stream. These are the typical refs to use when you want `gh aw update` to pick up newer upstream releases automatically.
- **Branch refs** (`@develop`) track the latest commit on a branch — useful for development integration.
- **SHA pins** (`@abc123def`) provide strict reproducibility and never move without an explicit change.

To pull upstream changes into an already-installed workflow:

```bash
gh aw update ci-doctor          # update one workflow
gh aw update                    # update all tracked workflows
```

Updates use a 3-way merge by default to preserve local edits. Use `--no-merge` to replace the local copy with the upstream version without merging. When the recorded `source:` uses a moving major ref such as `@v1`, `gh aw update` stays within that major line unless `--major` is passed.

### 5. Private and internal sharing controls

Not all workflows are safe to share across organizations. GitHub Agentic Workflows provides controls at multiple levels:

- **`private: true`** in frontmatter blocks a workflow from being installed into other repositories via `gh aw add`. Attempting to add a private workflow from another repository fails with an error.
- **Repository visibility** controls which workflows are discoverable. Private repositories require access before any workflow can be fetched.
- **Org-internal catalogs** can be implemented by placing workflows in a private or internal organization repository, ensuring only organization members can install them.

See [Private Workflows](/gh-aw/reference/frontmatter/#private-workflows-private) for configuration details.

### 6. Import caching and lock behavior

When a workflow is compiled, remote imports are resolved and locked. The compiled `.lock.yml` file records the exact commit SHA for every remote import, making runs reproducible regardless of upstream branch movement.

Imports are cached locally under `.github/aw/imports/` by commit SHA. Cached imports are used for all subsequent compilations until you explicitly update them. This means the lock file and the import cache together form the reproducibility guarantee for shared workflows.

### 7. Cross-repository execution model

Separate from sharing workflow definitions, workflows can operate across repositories at runtime:

- Read files and metadata from other repositories during execution.
- Check out code from target repositories for analysis or modification.
- Write safe outputs to target repositories with explicit authentication and allowlists.

```yaml
safe-outputs:
  create-issue:
    target-repo: "acme-org/target-repo"
    allowed-repos: ["acme-org/repo1", "acme-org/repo2"]
```

Cross-repository operations require appropriate GitHub token permissions and explicit `allowed-repos` declarations. See [Cross-Repository Operations](/gh-aw/reference/cross-repository/) for authentication, permissions, and safe output configuration.

## Recommended Enterprise Pattern

The recommended pattern for organizations sharing workflows at scale:

1. **One central `agentic-workflows` repository** holds versioned workflow templates and shared components under `workflows/` and `shared/`.
2. **Consuming repositories** use `gh aw add acme-org/agentic-workflows/<workflow>@<version>` to install complete workflows.
3. **Common modules** (MCP configurations, safety policies, shared prompts) live in `shared/` and are imported via `imports:` in consuming workflows.
4. **Version tags** on the central repository provide stable anchors for production consumers while branches support development integration.
5. **`private: true`** marks internal-only workflows that should not be exported outside the organization.

This model gives platform teams centralized ownership and update control while giving consuming teams reproducibility through version pins and the ability to preserve local customizations through 3-way merge.

## Governance Questions

When workflows are shared across an organization, the important decisions are usually operational rather than technical:

- Who owns the source workflow and reviews proposed changes.
- How updates are tested, tagged, and promoted to consuming repositories.
- Which repositories may consume or dispatch to shared workflows.
- How secrets, permissions, and safe outputs are standardized across consumers.
- When a consuming team may fork a workflow rather than stay on the shared version.

Those decisions affect reliability more than the file format does.

## Related Documentation

- [Reusing Workflows](/gh-aw/guides/packaging-imports/)
- [Imports Reference](/gh-aw/reference/imports/)
- [Cross-Repository Operations](/gh-aw/reference/cross-repository/)
- [Private Workflows](/gh-aw/reference/frontmatter/#private-workflows-private)
- [SideRepoOps](/gh-aw/patterns/side-repo-ops/)
- [MultiRepoOps](/gh-aw/patterns/multi-repo-ops/)
