# ADR-28366: Two-Phase Resolution of Body-Level `{{#import}}` Directives

**Date**: 2026-04-25
**Status**: Draft
**Deciders**: Unknown (copilot-swe-agent, pelikhan)

---

## Part 1 — Narrative (Human-Friendly)

### Context

Workflow markdown files support two ways to pull in shared content: frontmatter `imports:` entries (resolved at compile time) and inline `{{#import filepath}}` / `{{#runtime-import filepath}}` directives (resolved at runtime by `runtime_import.cjs`). Until this change, `{{#import}}` directives placed directly in the workflow *body* (rather than in frontmatter) were silently ignored at runtime — the agent received the raw macro string instead of the imported file's content. Additionally, the compiled lock file gave no visibility into which sibling-directory files (e.g. `.github/shared/editorial.md`) would be pulled in at runtime, making lock-file integrity checks incomplete.

### Decision

We will resolve body-level `{{#import}}` directives via a two-phase approach. At **compile time**, the Go compiler (`pkg/workflow/compiler_orchestrator_tools.go`) scans the markdown body for `{{#import:}}` directives (the colon form) and promotes them to explicit `{{#runtime-import}}` macros in the compiled lock file, giving the lock file full visibility into the import graph before runtime. At **runtime**, `runtime_import.cjs` normalises all remaining `{{#import}}` variants (colon, no-colon, optional `?` forms) to `{{#runtime-import}}` at the start of `processRuntimeImports`, ensuring any directive that was not promoted at compile time is still resolved. A deduplication `Set` in `runtime_import.cjs` prevents double-importing when both phases have emitted the same macro.

### Alternatives Considered

#### Alternative 1: Compile-Time Inline Expansion Only

Expand `{{#import}}` directives fully at compile time and inline the file content directly into the compiled prompt, matching the behaviour of frontmatter `imports:` entries that carry `inputs:`. This would eliminate the runtime dependency and make the compiled artefact self-contained. It was rejected because it creates a static snapshot of imported content that goes stale when the shared file is updated without recompiling the workflow — the runtime-import model exists precisely to get fresh shared-file content on every run.

#### Alternative 2: Runtime Normalisation Only (No Compile-Time Promotion)

Add the `{{#import}} → {{#runtime-import}}` normalisation solely inside `runtime_import.cjs` and leave the Go compiler unchanged. This is the simpler path and fixes the agent prompt bug. It was rejected because it leaves the lock file blind to body-level imports: the `Includes:` manifest header would not list `.github/shared/editorial.md`, so lock-file content-hash checks and dependency auditing tools would miss those files.

### Consequences

#### Positive
- Body-level `{{#import}}` directives (all four syntax variants: colon, no-colon, optional, optional-colon) are now correctly resolved and injected into the agent prompt at runtime.
- The compiled lock file's `Includes:` header now explicitly lists body-level imported files, enabling accurate lock-file integrity checks and dependency tracking.
- The `importedFiles` deduplication set in `runtime_import.cjs` prevents the same file being imported twice when both the compile-time promotion and the runtime normalisation emit the same macro.
- The repo-root-relative path helper (`findGitHubRepoRoot`) ensures that files in sibling `.github/` subdirectories are recorded with clean paths (e.g. `.github/shared/editorial.md`) rather than absolute system paths.

#### Negative
- Two separate codepaths now handle `{{#import}}`: compile-time promotion in Go (`ExtractBodyLevelImportPaths`) and runtime normalisation in JavaScript (`processRuntimeImports`). Keeping these in sync when the directive syntax evolves requires changes in both languages.
- The compile-time phase only promotes the `{{#import:}}` colon form (as used by `ParseImportDirective`). The no-colon `{{#import filepath}}` form is handled only at runtime, creating a syntax asymmetry that is not immediately obvious to workflow authors.
- Adding `ExtractBodyLevelImportPaths` introduces a second scan of the markdown body at compile time (the first scan is `ExpandIncludesWithManifest`), adding minor overhead for workflows with large bodies.

#### Neutral
- All daily agentic workflow `.lock.yml` files are regenerated to include the new `{{#runtime-import .github/workflows/shared/noop-reminder.md}}` macro — this is a mechanical, non-semantic change to the compiled artefacts.
- The new `findGitHubRepoRoot` helper is a pure function with no side effects; it is already unit-tested via `TestManifestIncludePathRelativeToRepoRoot`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Compile-Time Promotion

1. The Go compiler **MUST** call `ExtractBodyLevelImportPaths` on the markdown body after frontmatter has been stripped and before generating the lock file's `{{#runtime-import}}` macro list.
2. Each path returned by `ExtractBodyLevelImportPaths` **MUST** be emitted as an explicit `{{#runtime-import <path>}}` (or `{{#runtime-import? <path>}}` for optional) macro in the compiled lock file, appearing before the main workflow-file macro.
3. `ExtractBodyLevelImportPaths` **MUST** convert relative paths to workspace-root-relative form (e.g. `.github/workflows/shared/tools.md`) using `findGitHubRepoRoot` before returning them.
4. `ExtractBodyLevelImportPaths` **MUST NOT** process legacy `@include` or `@import` directives; those are handled by `ExpandIncludesWithManifest`.
5. The `Includes:` manifest header in the lock file **MUST** list every file referenced by a body-level `{{#import:}}` directive using a repo-root-relative path, not an absolute system path.

### Runtime Normalisation

1. `processRuntimeImports` **MUST** normalise all `{{#import}}` directive variants (with and without colon separator, with and without `?` optional marker) to their `{{#runtime-import}}` equivalents before the main macro processing loop executes.
2. The normalisation regex **MUST** match `{{#import filepath}}`, `{{#import: filepath}}`, `{{#import? filepath}}`, and `{{#import?: filepath}}`, but **MUST NOT** match tokens such as `{{#importantthing}}` that lack a whitespace or colon separator after `import`.
3. `processRuntimeImports` **MUST** apply deduplication via an `importedFiles` Set so that a file promoted at compile time and also referenced via a body-level `{{#import}}` at runtime is imported exactly once.
4. Recursive calls to `processRuntimeImports` on imported file content **MUST** pass the same `importedFiles` Set and `importCache` Map to propagate deduplication state.

### Path Resolution

1. `findGitHubRepoRoot` **MUST** walk up the directory tree from `baseDir` until it finds a directory named `.github` and return its parent, or return an empty string if no `.github` ancestor is found.
2. When building the `Includes:` manifest, the compiler **MUST** prefer repo-root-relative paths over `baseDir`-relative paths for files located in sibling `.github/` subdirectories (e.g. `.github/shared/`), falling back to absolute paths only when neither relative form is computable without a `..` prefix.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24919963187) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
