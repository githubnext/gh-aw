# ADR-34008: Manifest-Scoped Source Tracking and Manifest-Aware Update Orchestration

**Date**: 2026-05-22
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

Workflows installed from `aw.yml` repository package manifests were previously tracked in each workflow's `source` frontmatter using a per-file address (e.g. `owner/repo/workflows/triage.md@<sha>`). This made the `update` command unable to reason at manifest scope: it could refresh each file independently against its pinned path, but it could not see that several local workflows belonged to a single manifest, nor detect that the manifest itself had since added or removed workflow entries. A manifest package is intended to be installed and evolved as a *set* (the maintainer curates a list of workflow files under one address), so per-file tracking lost the grouping that defines the package. Filenames are also the natural key for a manifest-managed workflow (used in the local destination path), so two manifest entries that collapse to the same filename make the install ambiguous.

### Decision

We will track manifest-installed workflows by the **manifest address** (`repo` or `repo/package-path`, plus ref) in their `source` frontmatter, and we will orchestrate `update` for those workflows as a **manifest-scoped group**. Concretely: (1) `add` / `add-wizard` write the manifest address (not the per-file path) into `source` whenever a workflow originated from a repository package manifest; (2) `update` parses `source`, groups workflows by manifest address, resolves the latest manifest, and reconciles the local set against the latest entries — updating workflows still listed, adding newly listed ones, and removing workflows the manifest no longer lists; (3) manifest resolution rejects packages whose `files:` entries collapse to duplicate markdown filenames; and (4) the local-modification comparison normalizes the `source` field so a pure source-format change does not register as a user modification.

### Alternatives Considered

#### Alternative 1: Keep per-file `source` and infer the manifest at update time

We could have left `source` as a per-file path and reconstructed the manifest grouping at update time by probing each workflow's host repository for an `aw.yml` manifest containing that file. This was rejected because (a) it requires speculative network calls for every workflow on every update just to discover whether it is manifest-managed, (b) it cannot distinguish a workflow that was installed *via* a manifest from one that was installed by direct path even though the same file exists in a manifest, and (c) it provides no signal at install time, so `add` cannot record the user's intent ("I installed this as part of package X"). Persisting the manifest address at install time makes that intent first-class and lets `update` operate on it without inference.

#### Alternative 2: Introduce a separate side-car manifest lockfile

We could have left workflow `source` alone and instead written a top-level lockfile (e.g. `.aw/manifests.lock.yml`) listing each installed manifest and the workflow files it manages. This was rejected because it duplicates information that already lives in workflow frontmatter, creates a second source of truth that can drift from per-workflow `source`, and adds a new file lifecycle (creation, migration, conflict resolution under merges) for a problem that only needed an address-format change. Reusing the existing `source` field keeps one source of truth per workflow and avoids new repo-level state.

### Consequences

#### Positive
- `update` can now add and remove workflows as the upstream manifest changes, matching the package author's intent for the installed set.
- The `source` value for manifest-installed workflows directly identifies the manifest (`owner/repo` or `owner/repo/package-path`), which is easier for humans to read and easier to match against the manifest at update time.
- Branch-tracking semantics are preserved: when `source` references a branch, the branch name is retained after update so the workflow continues to follow that branch.
- Duplicate-filename manifests are now rejected at resolve time with a targeted error, surfacing a class of authoring mistakes that previously would have silently overwritten files locally.

#### Negative
- `source` is no longer a uniform "path to upstream file" — readers and tooling must now interpret two shapes (per-file path vs. manifest address) and dispatch on which one is present. This complicates anything that wants a single mental model for `source`.
- `update` for manifest groups performs **deletion** of local files (and their compiled `.lock.yml` siblings) when the upstream manifest drops an entry; users who customized those files lose them at update time without an explicit prompt.
- Two manifest resolutions per group per update are now required (current ref + latest ref) to compute the add/remove/update diff, increasing network calls and time relative to the prior per-file update path.
- Manifest packages that previously relied on having two markdown files with the same basename in different subdirectories are now invalid and will fail to install.

#### Neutral
- Existing per-file `source` strings still resolve and update via the legacy code path; only workflows that were (re-)installed after this change carry the manifest-address form.
- The local-modification detector now substitutes `source` with a fixed placeholder before comparing content, so any future format changes to the `source` field will not falsely register as a local modification.
- The `FromRepositoryManifest` flag on `WorkflowSpec` becomes a new piece of state that downstream consumers of the spec may need to consider.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Source-string generation

1. When a workflow is resolved from a repository package manifest, `add` and `add-wizard` **MUST** write a `source` value of the form `<repo>[@<ref>]` (manifest at repository root) or `<repo>/<package-path>[@<ref>]` (nested package), and **MUST NOT** include the per-file workflow path.
2. When a workflow is resolved directly by file path (not via a manifest), implementations **MUST** continue to write the per-file form (`<repo>/<path>@<ref>`); the manifest form **MUST NOT** be used.
3. Implementations **MUST** populate the `FromRepositoryManifest` field on `WorkflowSpec` when, and only when, the spec was produced by resolving a repository package manifest.

### Manifest validation

1. When resolving a repository package manifest, implementations **MUST** reject the manifest with an error if two or more entries in `files:` reduce to the same markdown filename (case-insensitive, extension stripped), regardless of the directory they live in.
2. The validation error message **MUST** name the manifest path and both conflicting entries.

### Update orchestration

1. `update` **MUST** parse each workflow's `source` field and classify it as either a manifest source or a per-file source.
2. Workflows whose `source` is a manifest source **MUST** be grouped by the trimmed source string and reconciled as a group against the latest manifest; they **MUST NOT** be updated via the per-file update path.
3. For each manifest group, implementations **MUST** resolve the manifest at both the currently recorded ref and the latest resolved ref, and **MUST** compute the set difference of workflow filenames between the two resolutions.
4. For each filename present locally but absent from the latest manifest, implementations **MUST** remove the local workflow markdown file and its sibling `.lock.yml` file.
5. For each filename present in the latest manifest but absent locally, implementations **MUST** download and write the new workflow into the same target directory as the existing manifest-managed workflows in that group.
6. For each filename present in both, implementations **MUST** update the existing local file using the latest manifest's path/ref, applying the same merge/security/compile pipeline used for per-file updates.
7. When the currently recorded ref is a branch reference, implementations **MUST** preserve the branch name in the written `source` after update; otherwise implementations **MUST** write the resolved latest ref.
8. When comparing source-resolved content against local content to detect local modifications, implementations **MUST** normalize the `source` frontmatter field on both sides to a fixed sentinel value before comparison, so that a change in `source` format alone does not register as a local modification.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
