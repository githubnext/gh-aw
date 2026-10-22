# ADR-71194: Validate Gallery Docs by Compiling Linked Repository Workflows

**Date**: 2026-10-22
**Status**: Draft
**Deciders**: pelikhan (PR author); [TODO: verify additional reviewers]

---

### Context

Documentation gallery pages (for example `docs/src/content/docs/gallery/linter-factory.md`) were previously validated by integration tests that either fetched a workflow named by a `gh aw add` directive from `githubnext/agentics`, or extracted inline ```` ```aw ```` excerpts from the page, patched their frontmatter (`engine`, `permissions`, `on`), and compiled the synthesized file. After the deployment guide and factory pages were relocated into the gallery/patterns sections, those pages link directly to canonical workflows in this repository (`https://github.com/github/gh-aw/blob/main/.github/workflows/*.md`) instead of embedding excerpts, so `TestDocumentationGalleryWorkflowsCompile` failed with an extraction error and the excerpt-based work-queue test no longer matched the page. The gate needed a way to keep docs examples compile-checked without reintroducing duplicated workflow bodies in the docs, and the same PR also tightened related regression fixtures (approval-expiry quota stubs, native-delivery settings mocks, reserved job-name validation) and refactored `pkg/cli/audit_diff.go` without changing output.

### Decision

We will treat repository workflows linked from gallery pages as the source of truth for documentation examples and validate them by compiling each linked `.github/workflows/<name>.md` in an isolated temporary `.github/workflows` fixture. The fixture copies `.github/skills` and `.github/aw`, copies subdirectories, symlinks sibling workflow files (excluding the target and its `.lock.yml`), and compiles the target with `SetApprove(true)`. Pages that link to repository workflows fall back to this path when no `gh aw add` directive is present; the work-queue factory test now asserts the exact set of linked eslint workflows instead of compiling patched excerpts.

### Alternatives Considered

#### Alternative 1: Keep inline `aw` excerpts in the docs and compile them with patched frontmatter

This was the previous approach. It verifies exactly what readers copy from the page, but it duplicates workflow content in docs, requires the test to inject `engine`, `permissions`, and `on` values that do not match production workflows, and drifts from the canonical workflows. It was dropped because the docs restructuring removed the excerpts in favor of links.

#### Alternative 2: Compile linked workflows in place inside the repository's `.github/workflows`

Compiling the real files in place is simpler and needs no fixture copying, but it would rewrite tracked `.lock.yml` files during tests and couple the test to the developer's working tree state. `make check-workflow-drift` already covers in-place compilation of all workflows, so this would duplicate coverage while adding side effects. The isolated fixture keeps tests hermetic.

#### Alternative 3: Only assert that documentation links resolve to existing files

This would be cheapest, but it would not detect a linked workflow that stops compiling under the import graph the docs describe. The PR uses this lighter check only for the factory page, where the full-compile path is already covered by the gallery test.

### Consequences

#### Positive
- Documentation no longer duplicates workflow bodies; the gallery links the canonical workflows that are actually maintained and compiled.
- Gallery tests compile real repository workflows with their real imports (shared files, skills, `aw` assets), so they catch import-graph breakage that excerpts with patched frontmatter could miss.
- Excluding the target's `.lock.yml` from the fixture avoids stale-lock interference and keeps the test hermetic.

#### Negative
- Readers no longer see a compile-verified inline snippet on the page; they must follow a link to `main`, which may differ from the docs version they are reading.
- The fixture copies `.github/skills` and `.github/aw` and symlinks every sibling workflow for each linked workflow, which increases test I/O and runtime as the repository grows.
- The link regex is hard-coded to `github/gh-aw/blob/main/.github/workflows/`; links to other branches, forks, or path layouts are silently ignored and not compile-checked.

#### Neutral
- `pkg/cli/audit_diff.go` uses promoted-field composite literals (`Status:` instead of `DiffEntryBase{Status: ...}`), which rely on the module's Go 1.27 toolchain, and extracts helpers (`mcpToolsForDiff`, `appendMCPToolDiff`, `runSummaryDiffMetrics`) to keep functions within size limits. The PR description states that diff output is unchanged.
- The related test-fixture fixes (quota stubs, settings mocks, reserved job names) follow existing patterns and do not change runtime behavior.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
