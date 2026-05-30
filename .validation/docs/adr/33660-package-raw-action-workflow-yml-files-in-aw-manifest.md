# ADR-33660: Package Raw GitHub Actions YAML Files (.yml) in aw.yml Manifests

**Date**: 2026-05-21
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

`aw.yml` package manifests historically listed only agentic workflow markdown files (`.md`) under `workflows/` or `.github/workflows/`, which the `gh aw add` command compiles into a `.lock.yml` GitHub Actions workflow at install time. Pack authors who also want to distribute traditional GitHub Actions workflows (raw `.yml` files such as `ci.yml` or `deploy.yml`) alongside their agentic workflows had no first-class way to bundle them — they had to ship them in a separate repository or instruct users to copy them by hand. The `.github/workflows/` directory is the canonical home for action workflows in any GitHub repo, but the manifest validator rejected anything that did not end in `.md`. Frontmatter extraction, dependency fetching, and lock-file generation are inappropriate for raw action YAML — those files are meant to be consumed by GitHub Actions verbatim.

### Decision

We will extend `aw.yml` to accept `.yml` files under `.github/workflows/` as a second, distinct file kind that is installed **as-is**: copied verbatim to the target `.github/workflows/<name>.yml` with no frontmatter injection, no dependency fetch, no compilation, and no `.lock.yml`. The two kinds are kept structurally separate via a new `IsActionWorkflow bool` field on `ResolvedWorkflow`; the resolution and installation pipelines branch on this flag rather than retrofitting the markdown pipeline. Markdown agentic workflows remain the only file kind permitted under `workflows/`; raw action YAML is restricted to `.github/workflows/` to match GitHub Actions' own discovery rules. Generated artifacts (`.lock.yml`) are explicitly excluded so a re-pack of an installed workflow cannot accidentally redistribute its compiled output.

### Alternatives Considered

#### Alternative 1: Treat `.yml` as another input to the agentic compilation pipeline

We could feed `.yml` files through the same `ResolveWorkflows` path used for markdown, attempting to parse a (likely empty) frontmatter block and skip downstream steps when none exists. This was rejected because the markdown pipeline implicitly assumes the input is an agentic workflow definition (frontmatter, body, lock generation, dependency resolution) — every step would need a "skip when raw YAML" guard. The result would be a single pipeline with two divergent behaviors at every node, instead of one explicit branch at the top. The new `IsActionWorkflow` flag isolates the new code path and keeps the markdown pipeline unchanged.

#### Alternative 2: Allow `.yml` files anywhere the manifest lists them, including `workflows/`

The manifest could accept `.yml` files under both `workflows/` and `.github/workflows/`. This was rejected because `workflows/` is the project's reserved namespace for **shareable agentic markdowns** — mixing raw action YAML into the same directory would break the convention that "everything under `workflows/` is a portable agentic workflow." Restricting raw `.yml` to `.github/workflows/` keeps each directory single-purpose and matches the path users already use for hand-authored action workflows.

#### Alternative 3: Require a separate manifest field (e.g., `action_files:`) instead of mixing kinds in `files:`

We could add a second top-level field to `aw.yml` (`action_files:`) so the file kind is declared by the field name rather than inferred from extension and path. This was rejected because the path itself (`.github/workflows/*.yml` vs. `workflows/*.md`) is already unambiguous, and a unified `files:` list keeps the manifest minimal — pack authors do not have to learn a second field, and tooling that walks `files:` continues to see one list. Extension-and-path inference also matches how the rest of the codebase classifies workflow files.

### Consequences

#### Positive
- A single `aw.yml` can bundle agentic workflows and traditional action workflows together, eliminating the need for a separate distribution channel for raw `.yml` files.
- Raw action YAML is preserved byte-for-byte at install time (`os.WriteFile(destFile, resolved.Content, ...)`), so signed workflows, comments, and formatting survive the round trip through a pack.
- The new install path (`addActionWorkflowWithTracking`) is fully isolated from the markdown pipeline: frontmatter extraction, dependency fetching, and lock generation are unreachable for action workflows, eliminating an entire class of compilation-time failure modes.
- `.lock.yml` files are explicitly rejected by `isActionWorkflowPath`, so a pack author cannot accidentally republish a compiled artifact as a "source" file.
- The interactive wizard correctly shows `<name>.yml` (and not a spurious `<name>.lock.yml`) for action workflows in the pre-confirmation file list, so users see exactly what will land on disk.

#### Negative
- The `ResolvedWorkflow` struct now carries a flag (`IsActionWorkflow`) that callers must inspect; every future code path that consumes `ResolvedWorkflow` must explicitly handle both kinds or risk treating action workflows as agentic markdown.
- The asymmetry — `.md` allowed under both `workflows/` and `.github/workflows/`, but `.yml` allowed only under `.github/workflows/` — is a convention that users must learn and that the validator must enforce; manifest authors who put `.yml` under `workflows/` will see a warning and a silently-dropped entry.
- `gh aw update` semantics for raw action workflows are not addressed by this PR; updates will overwrite the installed `.yml` verbatim with whatever the source repo currently has, with no diff-aware merging of local edits.
- Action workflows installed via this path have no `# gh-aw-metadata:` header, so tooling that reads pack-installation provenance from the lock-file header cannot recover the source spec from an installed `.yml`.

#### Neutral
- A new `isActionWorkflowPath(path string) bool` helper is added to `add_package_manifest.go` and reused in `add_workflow_resolution.go` for the path-classification check, so future callers have a single source of truth for "is this a raw action workflow file?".
- The manifest validation warning message changes from "workflow files must be markdown (.md) files…" to a longer form that names both supported kinds; downstream consumers that grep on the old text will need to update.
- `appendRepositoryPackageWorkflowSpecs` now strips `.yml` or `.md` from the file's basename to derive the workflow name, instead of unconditionally stripping `.md`.
- The new code path is exercised by three test functions (`TestIsSupportedPackageInstallablePath`, `TestResolveRepositoryPackage_ActionWorkflowYML`, `TestResolveWorkflows_ActionWorkflowYML`, `TestAddWorkflowWithTracking_ActionWorkflow`, `TestAddWorkflowWithTracking_ActionWorkflow_Force`), covering manifest acceptance, resolution, install-as-is, and the `--force` overwrite case.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Manifest Acceptance

1. Implementations of `isSupportedPackageInstallablePath` **MUST** accept a path as installable if and only if it satisfies one of the following:
   (a) the path ends in `.md` (case-insensitive) AND begins with `workflows/` or `.github/workflows/`, or
   (b) the path is an action workflow path as defined by `isActionWorkflowPath` AND begins with `.github/workflows/`.
2. Implementations **MUST NOT** accept a `.yml` file located outside `.github/workflows/` (e.g., under `workflows/`) as installable from a package manifest.
3. Implementations **MUST NOT** accept `.lock.yml` files as installable; `isActionWorkflowPath` **MUST** return false for any path ending in `.lock.yml` (case-insensitive).
4. When the manifest contains an entry that fails the above checks, implementations **MUST** emit a warning that names both supported file kinds (markdown `.md` under `workflows/` or `.github/workflows/`, and action workflow `.yml` under `.github/workflows/`) and **MUST** continue resolving the remaining manifest entries.

### Resolution

1. Implementations of `ResolveWorkflows` **MUST**, for any spec whose `WorkflowPath` satisfies `isActionWorkflowPath`, set `ResolvedWorkflow.IsActionWorkflow = true` and **MUST NOT** perform agentic-workflow frontmatter extraction, description extraction, `HasWorkflowDispatch` detection, or `IsPrivate` detection for that workflow.
2. `appendRepositoryPackageWorkflowSpecs` **MUST** derive `WorkflowName` by stripping the `.yml` suffix when the installation source path satisfies `isActionWorkflowPath`, and by stripping the `.md` suffix otherwise.
3. The `ResolvedWorkflow` struct **MUST** include a boolean field named `IsActionWorkflow` whose semantics are "the source is a raw GitHub Actions YAML file installed verbatim, not an agentic workflow markdown."

### Installation

1. When `ResolvedWorkflow.IsActionWorkflow` is `true`, `addWorkflowWithTracking` **MUST** delegate installation to `addActionWorkflowWithTracking` and **MUST NOT** invoke any agentic-markdown installation logic (frontmatter injection, dependency fetch, compilation, or lock-file generation) for that workflow.
2. `addActionWorkflowWithTracking` **MUST** write the resolved content byte-for-byte to `<githubWorkflowsDir>/<workflowName>.yml` and **MUST NOT** create a corresponding `.md` file or `.lock.yml` file.
3. When the target `.yml` file already exists and `opts.Force` is `false`, `addActionWorkflowWithTracking` **MUST** return an error in the single-spec case or skip with a warning in the wildcard case (`opts.FromWildcard == true`); it **MUST NOT** silently overwrite an existing file.
4. When `opts.Force` is `true` and the target `.yml` file exists, `addActionWorkflowWithTracking` **MUST** overwrite the existing file and **MUST** report the file to the tracker as modified rather than created.
5. When a `FileTracker` is supplied, `addActionWorkflowWithTracking` **MUST** call `TrackCreated` for new files and `TrackModified` for overwrites.

### Wizard Display

1. The interactive wizard's `determineFilesToAdd` **MUST**, for each resolved workflow with `IsActionWorkflow = true`, list exactly one file (`<workflowName>.yml`) in the pre-confirmation file list and **MUST NOT** list a corresponding `.lock.yml` for that workflow.
2. The wizard **MUST** continue to list both `<workflowName>.md` and `<workflowName>.lock.yml` for resolved workflows with `IsActionWorkflow = false`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26202040962) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
