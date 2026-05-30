# ADR-27387: Filter Non-Frontmatter Markdown Files During compile-all Discovery

**Date**: 2026-04-20
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw `compile-all` command discovers workflow sources by listing every `*.md` file (excluding `README.md`) found in `.github/workflows/`. Some repositories legitimately place non-workflow documentation — notes, runbooks, guides — as Markdown files in that same directory. These files have no YAML frontmatter, so the compiler encounters them, fails to parse them, and emits noisy `no frontmatter found` errors. The result is inflated compatibility-failure counts and degraded signal-to-noise for developers running compile-all.

### Decision

We will add a `filterMarkdownFilesWithFrontmatter` step that runs immediately after Markdown file discovery and before any compilation attempt. The filter reads the first line of each candidate file and keeps only those whose first line is exactly `---` — the YAML frontmatter delimiter used by every gh-aw workflow. Files that are empty or whose first line is anything other than `---` are silently skipped. The filter is applied in both `compileAllWorkflowFiles` (in `compile_file_operations.go`) and `compileAllFilesInDirectory` (in `compile_pipeline.go`); the underlying `getMarkdownWorkflowFiles` listing function is left unchanged so that non-compile callers are not affected.

### Alternatives Considered

#### Alternative 1: Suppress the "no frontmatter" error in the parser

The parser could downgrade the `no frontmatter found` diagnostic from an error to a debug-level log, letting compile-all silently continue. This approach avoids the additional I/O of a pre-scan. However, it only masks the symptom: the compiler still attempts a full parse of every non-workflow file, wasting CPU, and the file-discovery boundary between "any Markdown" and "workflow Markdown" remains blurred. It was rejected because it papers over the root cause rather than establishing a clean separation.

#### Alternative 2: Enforce a separate directory for non-workflow docs

Repositories could be required to keep documentation Markdown in a directory other than `.github/workflows/`. This would make the directory semantics unambiguous and eliminate the need for a filter entirely. It was rejected because it is a breaking change for existing repositories that already co-locate docs and workflows, and it imposes a structural constraint on users that has no benefit beyond compile-time convenience.

#### Alternative 3: Name-based convention for workflow files

Workflow Markdown files could be required to follow a specific naming pattern (e.g., end with `-workflow.md`), and compile-all would only process matching files. This would make the filter a simple `filepath.Match` with no I/O. It was rejected because it would require renaming every existing workflow, making it a major backwards-incompatible change.

### Consequences

#### Positive
- `compile-all` no longer generates false `no frontmatter found` errors for documentation files co-located with workflows.
- Compatibility-failure counts become more accurate, improving the signal value of compile-all output.
- The fix is surgically scoped: `getMarkdownWorkflowFiles` and all non-compile callers are untouched.

#### Negative
- A workflow file whose frontmatter block is accidentally missing or malformed (e.g., starts with a BOM or a blank line before `---`) will be silently skipped rather than producing a clear error, potentially hiding real authoring mistakes.
- The filter reads file contents for every discovered Markdown file before compilation begins, adding one extra `os.ReadFile` per file over the previous behavior.

#### Neutral
- The first-line check is intentionally strict (`bytes.Equal(firstLine, []byte("---"))`); any leading whitespace or UTF-8 BOM before `---` will cause the file to be skipped. This is consistent with how YAML frontmatter is defined in the gh-aw spec.
- Both compile pipelines now share a single filtering function, reducing future drift risk.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Markdown File Discovery

1. Implementations **MUST** apply the frontmatter filter to the list of Markdown files produced by `getMarkdownWorkflowFiles` before passing any file to the compiler.
2. Implementations **MUST** retain a Markdown file for compilation if and only if the file's first line (the bytes before the first `\n`) is exactly the three-byte sequence `---`.
3. Implementations **MUST NOT** pass a Markdown file to the compiler when the file is empty (zero bytes).
4. Implementations **MUST NOT** pass a Markdown file to the compiler when its first line contains any bytes other than `---` (including leading whitespace or BOM characters).
5. Implementations **SHOULD** emit a debug-level log entry naming each skipped file so that developers can diagnose unexpected omissions.

### Compile Pipeline Integration

1. Implementations **MUST** apply the frontmatter filter in every code path that calls `getMarkdownWorkflowFiles` and subsequently compiles the resulting files.
2. Implementations **MUST NOT** modify `getMarkdownWorkflowFiles` to incorporate the filter; the filter **MUST** remain a separate, composable step.
3. Implementations **MAY** cache file-read results within a single compile-all invocation to avoid reading the same file twice if the filter and the compiler would otherwise both read it.

### Error Handling

1. Implementations **MUST** propagate any `os.ReadFile` error that occurs during filtering as a compile-time error; they **MUST NOT** silently skip a file whose content cannot be read due to an I/O error.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24679557357) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
