# ADR-70029: Fast Authored JavaScript Analysis with Reviewed, Hash-Verified API Summaries

**Date**: 2026-10-18
**Status**: Draft
**Deciders**: pelikhan, Unknown

---

### Context

The workflow linter's JavaScript secret-flow (taint) analysis (see ADR-69813) traverses JavaScript through a native AST parser. When every lint pass walks the generated workflow runtime code (`actions/setup/js/*.cjs` helpers and generated JavaScript in compiled lock files), analysis becomes prohibitively expensive. At the same time, the analyzer is a security control: it must not weaken the existing compiled shell/job/output checks, and incomplete coverage must never be reported as "safe". User-authored scripts live in many schema locations (steps, custom jobs, safe-output jobs/handlers, MCP tools, graders, memory validators, engine drivers, harnesses, log parsers), each with its own sandbox, environment, and output contracts.

### Decision

We will add a `fast` JavaScript analysis mode, made the CLI default when native analysis is configured via `--javascript-module`, that analyzes only user-authored scripts discovered at known executable schema locations, and replaces traversal of runtime helpers with a versioned, reviewed API summary catalog (`pkg/workflow/taint/javascript/summaries.json`). A summary is used only when the module's SHA-256 and the hashes of its transitive dependencies match the reviewed values; otherwise a coverage gap is reported. Native parsing is batched and deduplicated across documents (bounded to 64 sources / 2 MiB per batch) with cached ASTs and per-source syntax failures. `--javascript-mode deep` retains full runtime traversal, and an unset mode in the Go API preserves the previous deep behavior.

### Alternatives Considered

#### Alternative 1: Keep deep runtime traversal as the only mode

Always traverse runtime helper bodies and generated JavaScript. This gives the most precise flows without a maintained catalog, but its cost is the problem this PR addresses. It remains available as `--javascript-mode deep` instead of being the default.

#### Alternative 2: Trust runtime helpers by name/path without hash verification

Model well-known helpers (e.g., `maskSecret`, `buildGitAuthEnv`) by module path or export name only. This is simpler and needs no hash maintenance, but a changed or substituted helper could silently invalidate the model and produce false "safe" results. That conflicts with the requirement that unreviewed or changed APIs remain visible coverage warnings.

#### Alternative 3: Cache deep-analysis results instead of using summaries

Memoize the results of deep runtime traversal between runs. This keeps precision, but the first run stays expensive, cache invalidation across runtime versions adds complexity, and per-call-site field sensitivity is harder to reuse than an explicit transfer summary. [TODO: verify whether this was evaluated]

### Consequences

#### Positive
- Default lint passes avoid traversing generated runtime code. Batching and deduplication further reduce native parser invocations (measured by `BenchmarkNativeJavaScriptBatch` and `BenchmarkJavaScriptFastSummary`).
- Hash-checking each module and its transitive dependencies ensures summaries apply only to the exact reviewed source. Unknown or changed APIs surface as coverage gaps rather than being assumed safe.
- Schema-driven discovery of executable script locations replaces matching any property named `script`, which reduces false positives and missed entry points.

#### Negative
- `summaries.json` must be updated (and re-reviewed) whenever a summarized helper or its dependencies change. Otherwise fast mode regresses to coverage warnings.
- Fast mode explicitly excludes generated and cross-job JavaScript flows. Precision for these flows requires opting in to the slower `deep` mode.
- The CLI default (`fast`) differs from the Go API default (unset → deep), which creates two behaviors that maintainers must keep in mind.

#### Neutral
- Native JavaScript analysis remains opt-in through `--javascript-module`; the change only affects behavior once it is enabled.
- Compiled shell/job/output checks remain active for paired documents; only the JavaScript checks switch to authored sources.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
