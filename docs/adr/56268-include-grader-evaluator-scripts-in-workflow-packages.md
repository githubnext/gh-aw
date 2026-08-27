# ADR-56268: Include Grader Evaluator Scripts in Workflow Packages

**Date**: 2026-08-27
**Status**: Draft
**Deciders**: Unknown

---

### Context

When users install a packaged workflow via `gh aw` that declares grader evaluators under `graders.*.run`, the evaluator shell scripts referenced by those keys were not co-installed alongside the workflow files. This caused workflow compilation to fail immediately in clean repositories because the compiler expects those scripts to be present. The root cause was that the package resource resolver (`extractResources`) only scanned the `resources:` frontmatter field and had no awareness of the `graders:` stanza. Evaluator scripts live at repository-root-relative paths (under `.github/graders/`) rather than relative to the workflow file, and the same evaluator may be shared across multiple workflows in one package.

### Decision

We will extend `extractResources` to also parse grader frontmatter via `workflow.ParseGradersFromFrontmatter` and append each non-empty `run` path to the resource list, then deduplicate the combined set before returning it. Within `fetchAndSaveRemoteResources` we will introduce an `isGraderEvaluator` branch that (a) resolves evaluator paths repository-root-relative rather than workflow-directory-relative, (b) installs them relative to the git repository root rather than the workflows target directory, and (c) allows silent no-op re-installs when the existing file content matches the incoming download, blocking only on content divergence. This routes evaluator scripts through the existing resource installation and ownership infrastructure so that `gh aw update` can restore or update them without additional code paths.

### Alternatives Considered

#### Alternative 1: Require explicit listing in `resources:` field

Workflow authors would be required to manually duplicate the evaluator path in both `graders.*.run` and `resources:`. This was the implicit status-quo before this fix. It is rejected because it creates a footgun: authors writing a `graders:` stanza naturally expect the referenced script to be packaged, and the silent omission produces a compilation failure that is hard to diagnose in clean repositories.

#### Alternative 2: Make missing evaluators a non-fatal warning at compile time

The compiler could treat a missing evaluator as a warning rather than an error, allowing workflows to install without the script. This is rejected because evaluators are required for grader execution — silencing the error would allow users to install broken workflows that fail at runtime rather than at installation/compilation time, which is a worse developer experience.

### Consequences

#### Positive
- Workflows with grader evaluators install successfully in clean repositories without any manual `resources:` duplication.
- `gh aw update` automatically restores or updates missing evaluator scripts using the existing resource lifecycle.
- Evaluator scripts shared across multiple packaged workflows are deduplicated, avoiding redundant downloads.

#### Negative
- `fetchAndSaveRemoteResources` now contains two distinct path-resolution conventions (workflow-dir-relative for ordinary resources, repo-root-relative for grader evaluators), increasing the function's complexity. A `//nolint:largefunc` suppression is needed.
- Content-based conflict detection for grader evaluators (byte-by-byte comparison before allowing overwrite) is deferred until after the download, adding a network round-trip even when the file would ultimately be skipped. Ordinary resources use a pre-download existence check.
- The `isGraderEvaluator` heuristic relies on the `.github/graders/` path prefix convention; evaluators stored elsewhere would not be detected and would still require manual `resources:` listing.

#### Neutral
- The `downloadResourceFileFromGitHub` function is extracted as a package-level variable to allow test injection, which is a minor testability refactor with no production behavior change.
- The `absTargetDir` pre-computation is moved from the function entry point into the per-resource loop body to support per-resource target base switching.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
