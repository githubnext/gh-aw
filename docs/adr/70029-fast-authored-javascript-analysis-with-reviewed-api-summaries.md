# ADR-70029: Fast Authored JavaScript Analysis with Reviewed, Hash-Verified API Summaries

**Date**: 2026-10-18
**Status**: Accepted
**Deciders**: pelikhan

---

### Context

The workflow linter's JavaScript secret-flow (taint) analysis (see ADR-69813) traverses JavaScript through a native AST parser. When every lint pass walks the generated workflow runtime code (`actions/setup/js/*.cjs` helpers and generated JavaScript in compiled lock files), analysis becomes prohibitively expensive. At the same time, the analyzer is a security control: it must not weaken the existing compiled shell/job/output checks, and incomplete coverage must never be reported as "safe". User-authored scripts live in many schema locations (steps, custom jobs, safe-output jobs/handlers, MCP tools, graders, memory validators, engine drivers, harnesses, log parsers), each with its own sandbox, environment, and output contracts.

### Decision

Add a `fast` JavaScript analysis mode, made the CLI default when native analysis is configured via `--javascript-module`, that starts from user-authored scripts discovered at known executable schema locations. Modules reached through operator-verified runtime mappings use a versioned, reviewed API summary catalog (`pkg/workflow/taint/javascript/summaries.json`). Resolve the requested file before checking its identity, source SHA-256, and transitive dependency hashes. Unknown or changed mapped runtime modules report coverage gaps; ordinary repository modules remain traversable regardless of their directory name.

Summaries describe symbolic return values and side effects rather than cached call results. They preserve credential provenance, callback effects, and unshadowed runtime globals. Mask registration is not declassification, and unsupported coercion or callback behavior remains incomplete coverage.

Native parsing is batched and deduplicated across documents, bounded to 64 sources / 2 MiB per batch, with cached ASTs and per-source syntax failures. Cache priming is an optional optimization: failures are reported by normal per-file analysis, and only caller cancellation aborts preparation. Ordinary linting streams parsed Actions graphs instead of retaining the whole directory. Hook analysis follows the owning runtime's environment, invocation, and payload-encoding contracts.

`--javascript-mode deep` retains available runtime-body traversal, and an unset mode in the Go API preserves the previous deep behavior. Paired fast analysis retains compiled shell/job/output checks and adds authored JavaScript flows without duplicating ordinary authored findings.

### Alternatives Considered

#### Alternative 1: Keep deep runtime traversal as the only mode

Always traverse runtime helper bodies and generated JavaScript. This avoids maintaining a catalog, but repeated runtime interpretation is the cost this change addresses and still has unsupported-flow limits. It remains available as `--javascript-mode deep` instead of being the CLI default.

#### Alternative 2: Trust runtime helpers by name/path without hash verification

Model well-known helpers (e.g., `maskSecret`, `buildGitAuthEnv`) by module path or export name only. This is simpler and needs no hash maintenance, but a changed or substituted helper could silently invalidate the model and produce false "safe" results. That conflicts with the requirement that unreviewed or changed APIs remain visible coverage warnings.

#### Alternative 3: Cache deep-analysis results instead of using summaries

Retain AST caching while interpreting every runtime helper for each analysis. Investigation found that warm ASTs still incurred runtime evaluation cost; caching syntax alone does not remove repeated interpretation. Persisting whole-analysis results would additionally require keys for symbolic inputs, environments, callbacks, and runtime versions. Keep in-process AST caching as a complementary optimization, but use explicit transfer summaries for reusable runtime effects.

### Consequences

#### Positive
- Configured fast-mode lint passes avoid traversing mapped runtime bodies and generated JavaScript. Batching and deduplication further reduce native parser invocations (measured by `BenchmarkNativeJavaScriptBatch` and `BenchmarkJavaScriptFastSummary`).
- Hash-checking each module and its transitive dependencies ensures summaries apply only to the exact reviewed source. Unknown or changed APIs surface as coverage gaps rather than being assumed safe.
- Schema-driven discovery of executable script locations replaces matching any property named `script`, which reduces false positives and missed entry points.

#### Negative
- `summaries.json` must be updated (and re-reviewed) whenever a summarized helper or its dependencies change. Otherwise fast mode regresses to coverage warnings.
- Fast mode explicitly excludes generated and cross-job JavaScript flows. Precision for these flows requires opting in to the slower `deep` mode.
- The CLI default (`fast`) differs from the Go API default (unset → deep), which creates two behaviors that maintainers must keep in mind.
- Summary correctness requires reviewing runtime execution contracts, not only refreshing source hashes. Regression tests must run with the pinned native compiler installed.

#### Neutral
- Native JavaScript analysis remains opt-in through `--javascript-module`; the change only affects behavior once it is enabled.
- Compiled shell/job/output checks remain active for paired documents; only the JavaScript checks switch to authored sources.
