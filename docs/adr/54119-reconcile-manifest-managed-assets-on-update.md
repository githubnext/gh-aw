# ADR-54119: Reconcile Manifest-Managed Assets During `gh aw update`

**Date**: 2026-08-20
**Status**: Draft
**Deciders**: Unknown

---

### Context

`gh aw update` previously updated only already-installed Markdown workflow files that carry a `source:` frontmatter field. Package manifests (`aw.yml`) can also declare non-Markdown assets—action workflows (`.yml`), skills, and agents—under their `includes` list. Because these assets do not carry per-file source tracking, new entries added to a package manifest after initial installation were silently skipped, leaving installed packages incomplete even as their existing workflows advanced to the latest source commit.

### Decision

We will re-resolve both the current and latest package manifests during `gh aw update` and install any package-owned assets (action workflows, skills, agents) that appear in the latest manifest but are absent from the current one. Installation is conservative: if the destination path already exists on disk the file is skipped to avoid overwriting local modifications. Ownership is derived from the manifest rather than per-file frontmatter.

### Alternatives Considered

#### Alternative 1: Require `source:` Frontmatter on All Package-Managed Assets

Package authors could be required to embed `source:` metadata into every asset file. This would let the existing per-file update logic handle all asset types uniformly without new manifest-diffing code. However, it couples the manifest format to each asset's content, imposes a documentation burden on package authors, and is impractical for binary or generated files that cannot carry YAML frontmatter.

#### Alternative 2: Track Installed Manifest Assets in a Dedicated Lockfile

A separate lockfile (e.g., `aw.lock`) could record which asset paths were installed from which package version, enabling precise reconciliation and future removal support. This would centralize ownership tracking and decouple it from both frontmatter and manifest diffing. The trade-off is added file-system overhead, a required migration step for existing installations that lack a lockfile, and increased complexity in the update flow.

### Consequences

#### Positive
- New assets declared in `aw.yml` are automatically installed when users run `gh aw update`, delivering complete package updates without manual intervention.
- The conservative install behavior (destination-exists check) prevents unintentional overwrites of locally modified files.

#### Negative
- Assets are keyed by destination path (action workflows) or source path (skills/agents). A package that renames an asset across versions will produce a duplicate installation rather than an in-place update, requiring manual cleanup.
- Reconciliation errors are appended to the failure list for the entire workflow group, so a single failed asset download can suppress success reporting for other updates in the same run.

#### Neutral
- The reconciliation logic runs unconditionally at the end of every `updateManifestWorkflowGroup` call, adding a GitHub API download per newly declared asset; no additional network calls are made for unchanged assets.
- Removal of assets dropped from the manifest is not addressed by this change; removal behavior remains conservative (unchanged from the prior implementation).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
