# ADR-28482: Centralize Cache-Memory Directory Path Computation via Constants and Helper Function

**Date**: 2026-04-25
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The cache-memory feature provides agents with persistent read/write storage by mounting directories at deterministic runtime paths (`/tmp/gh-aw/cache-memory` for the default cache, `/tmp/gh-aw/cache-memory-{id}` for named caches). The rule governing path construction — an if/else branch on whether the cache ID equals `"default"` — was duplicated more than ten times across three files (`cache.go`, `claude_tools.go`, and `copilot_engine_execution.go`) with no single authoritative definition. This duplication made the naming convention implicit, difficult to audit, and prone to silent drift if one copy was updated without the others.

### Decision

We will extract the cache-memory path convention into two package-level constants (`defaultCacheMemoryDir`, `cacheMemoryDirPrefix`) and a single helper function `cacheMemoryDirFor(cacheID string) string` in `pkg/workflow/cache.go`. All call sites across the package will be refactored to call this helper rather than constructing paths inline. This creates a single source of truth: any future change to the path scheme requires a one-line edit in one place.

### Alternatives Considered

#### Alternative 1: Document the Convention with Comments, Keep Inline Construction

Each call site could receive an explanatory comment pointing to a canonical example, establishing the convention by documentation rather than code. This was rejected because documentation-only approaches do not prevent future contributors from introducing new call sites that deviate from the pattern, and the duplicated logic would continue to diverge silently under refactoring.

#### Alternative 2: Embed Path Logic in a Centralized Configuration Struct

The `CacheMemoryEntry` struct could expose a `Dir() string` method, moving the path logic closer to the data it describes. This was considered but rejected for this PR because the path is fundamentally an infrastructure constant (it is baked into compiled workflow YAML and agent prompts), not a runtime-configurable value. Placing it on the struct would imply configurability that does not exist and would require passing the struct to every call site that currently only needs the string.

### Consequences

#### Positive
- All ten scattered if/else path blocks are replaced with a single call, eliminating the risk of the copies drifting apart.
- Future changes to the path scheme (e.g., moving from `/tmp/` to a different base) require modifying exactly one function.
- The helper is independently testable and its contract (empty/`"default"` ID → default path; any other ID → prefixed path) can be exercised in isolation.

#### Negative
- The helper is package-private (`pkg/workflow`); code outside the package cannot reuse it without either exporting it or duplicating the logic, which limits future cross-package reuse if other engines are added.
- Call sites no longer show the concrete path at a glance; readers must follow the indirection into `cacheMemoryDirFor` to verify the exact value.

#### Neutral
- The PR also aligns prompt templates and reference documentation with the existing implementation (adding the `__GH_AW_ALLOWED_EXTENSIONS__` placeholder, correcting key-format descriptions). These are correctness fixes that do not affect the architectural decision above.
- No changes to the compiled workflow YAML format; the runtime paths emitted remain identical.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Cache-Memory Directory Path Computation

1. All code within `pkg/workflow` that resolves a cache ID to a filesystem path **MUST** call `cacheMemoryDirFor(cacheID)` rather than constructing the path inline.
2. New call sites **MUST NOT** introduce a new if/else branch that re-implements the `default`-vs-named-cache path logic.
3. `cacheMemoryDirFor` **MUST** return `/tmp/gh-aw/cache-memory` for the ID `"default"` and for the empty string.
4. `cacheMemoryDirFor` **MUST** return `/tmp/gh-aw/cache-memory-{id}` (with no trailing slash) for any non-default, non-empty cache ID.
5. Callers that require a trailing slash for display or directory-argument context **SHOULD** append `"/"` at the call site rather than relying on the helper to do so.

### Constants

1. The canonical default path **MUST** be defined as the package-level constant `defaultCacheMemoryDir`.
2. The named-cache path prefix **MUST** be defined as the package-level constant `cacheMemoryDirPrefix`.
3. These constants **MUST NOT** be duplicated as string literals elsewhere in the package.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24936314671) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
