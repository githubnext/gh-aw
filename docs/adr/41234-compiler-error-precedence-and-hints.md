# ADR-41234: Compiler error reporting — validate engine before imports and enrich errors with hints

**Date**: 2026-06-24
**Status**: Accepted

## Context

The daily syntax-error quality check scored compiler diagnostics at an average of 44/100, exposing three developer-experience defects in how `pkg/workflow` and `pkg/parser` surface frontmatter and import errors. First, an invalid `engine:` value (e.g. `copiilot`) was silently shadowed when the same workflow also had a failing import — the import error was reported and the engine typo never reached the user. Second, import-resolution errors carried no actionable fix hint, so a "file not found" told the author *what* failed but not *how* to fix it. Third, when a frontmatter error fell on the first frontmatter line (line 2), the YAML context snippet omitted the faulty line entirely because `sourceContextPattern` required leading whitespace and did not match the `>  N |` highlighted-error-line prefix. These are user-facing error-quality concerns in the compiler's hot path, where ordering and message content directly determine whether an author can self-correct.

## Decision

We will validate the resolved `engine:` setting against `engineCatalog.Resolve()` **early in the compiler pipeline** — immediately after `injectBuiltinEngineImportIfNeeded` and *before* `processEngineImportsAndMerge` — so that engine-name errors take precedence over import-resolution errors in user-facing output. The early check is intentionally skipped when `engineSetting` is empty (the engine may be supplied by an import) or when a command-line `--engine` override is active (it is validated later). Alongside this precedence decision, we will enrich import errors with cause-specific hints via a new `buildImportErrorHint` helper populating `CompilerError.Hint`, and broaden `sourceContextPattern` to `>?\s*\d+\s*\|` so the highlighted faulty line is always included in the context snippet.

## Alternatives Considered

### Alternative 1: Collect all errors and rank them at the end

Rather than failing fast on the first engine error, the compiler could accumulate every diagnostic (engine, imports, frontmatter) and sort them by a severity/priority ranking before reporting. This produces a complete picture in one pass but requires threading an error-aggregator through the pipeline and defining a stable priority ordering across error kinds. Rejected for this change as disproportionate: the immediate problem is a single precedence inversion (engine vs. import), which early fail-fast validation fixes with ~10 lines and no new aggregation machinery. *(Close call — revisit if more precedence inversions appear.)*

### Alternative 2: Improve messages only, leave ordering unchanged

We could keep the existing validation order and only add hints and richer messages, trusting authors to fix imports first and rediscover the engine typo on a later compile. Rejected because it leaves the core defect — engine typos being unreportable in the presence of an import failure — unaddressed, forcing an avoidable edit-compile iteration on the author.

## Consequences

### Positive

- An invalid `engine:` value now always surfaces (with a "Did you mean: copilot?" suggestion) even when imports also fail, eliminating a class of shadowed errors.
- Import errors carry a tailored, path-specific hint per cause variant (file not found, download failure, bad ref, invalid workflowspec), telling authors how to fix the problem rather than only what failed.
- Frontmatter errors on the first frontmatter line now include the faulty line in the context snippet, restoring the most important line of the diagnostic.

### Negative

- Early engine validation is deliberately skipped when the engine is empty or a `--engine` override is set, so coverage is partial — those paths still rely on later validation and could in principle re-introduce shadowing for override flows.
- Fail-fast precedence means a workflow with both an engine typo *and* a broken import shows only the engine error first; the author fixes it, recompiles, and only then sees the import error — two iterations instead of one combined report.
- `buildImportErrorHint` matches on substrings of the error message; if upstream message wording changes, hints can silently fall through to the default branch and become less specific.

### Neutral

- The change encodes "engine before imports" as a compiler error-precedence convention; future error kinds must decide where they slot into this ordering.
- Broadening `sourceContextPattern` from `\s+\d+\s*\|` to `>?\s*\d+\s*\|` makes the leading whitespace optional and the `>` marker explicit; the new pattern is slightly more permissive about what counts as a context line.
