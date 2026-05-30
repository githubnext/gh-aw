# ADR-28282: Relocate Outlier Functions to Their Semantic Homes

**Date**: 2026-04-24
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

Several functions in `pkg/workflow` and `pkg/cli` had accumulated in files that did not own their semantic domain. `computeIntegrityCacheKey` lived in `cache.go` despite calling three helpers (`cacheIntegrityLevel`, `computePolicyHash`, `generateIntegrityAwareCacheKey`) that all resided in `cache_integrity.go`. Three npm/Node.js step-generation functions (`BuildStandardNpmEngineInstallSteps`, `BuildNpmEngineInstallStepsWithAWF`, `GetNpmBinPathSetup`) lived in `engine_helpers.go` alongside non-npm helpers rather than with the rest of npm logic in `nodejs.go`. In `pkg/cli`, `safePercent` and `formatPercent` were split across `audit_cross_run_render.go` and `audit_diff.go` despite being a compute–format pair consumed by both audit rendering files. This placement made the call graph opaque and complicated discoverability for contributors.

### Decision

We will relocate each outlier function to the file that already owns its semantic domain: `computeIntegrityCacheKey` moves to `cache_integrity.go`, the three npm functions move to `nodejs.go` (updating the logger reference from `engineHelpersLog` to `nodejsLog`), and `safePercent`/`formatPercent` are consolidated in a new `pkg/cli/audit_math_helpers.go`. No behavior is changed; this is a pure relocation. This extends the file-organization convention established in ADR-27325 beyond the specific files it addressed.

### Alternatives Considered

#### Alternative 1: Leave Functions in Place, Add Cross-File Documentation

Inline comments could document that a function in file A primarily supports file B. This was rejected because comments do not enforce boundaries — over time contributors would continue adding similar "outlier" functions to the nearest convenient file rather than the semantically correct one. Documentation without structural enforcement degrades.

#### Alternative 2: Extract a Shared `helpers.go` Umbrella File

All small utilities could be consolidated into a single `pkg/workflow/helpers.go`. This reduces the number of files and requires no judgment about semantic ownership. It was rejected because it recreates the catch-all umbrella pattern that ADR-27325 explicitly discourages; a function's location would again fail to communicate its purpose.

#### Alternative 3: Move Functions to a Sub-Package

Creating sub-packages (e.g., `pkg/workflow/cacheintegrity`) would enforce hard import boundaries. This was rejected because the functions are used exclusively within their parent packages, making the added indirection and forced symbol export an unnecessary cost for what is logically internal rearrangement.

### Consequences

#### Positive
- `computeIntegrityCacheKey` and all its transitive callees now reside in `cache_integrity.go`, making the entire integrity cache key computation readable in a single file.
- All npm/Node.js step-generation logic is consolidated in `nodejs.go`, simplifying discovery and reducing cognitive load when working on engine installation steps.
- `safePercent` and `formatPercent` share a single definition in `audit_math_helpers.go`, enabling consistent `formatPercent(safePercent(a, b))` composition across all audit rendering code.
- Extends and reinforces the semantic file-organization convention from ADR-27325 to new areas of the codebase.

#### Negative
- Increases file count in `pkg/workflow/` and `pkg/cli/`, adding marginal navigation overhead.
- The move adds churn to `git log` for the relocated functions; `git blame` will show the relocation commit rather than the original authorship without `--follow`.

#### Neutral
- No public API surface changes; all moved functions retain their signatures.
- The `engine_helpers.go` header comment is updated to remove stale npm function references, keeping the file's documented scope accurate.
- The logger reference change (`engineHelpersLog` → `nodejsLog`) is a cosmetic alignment with the new file's existing logger; no log output changes.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Cache Integrity

1. `computeIntegrityCacheKey` and all helpers it calls (`cacheIntegrityLevel`, `computePolicyHash`, `generateIntegrityAwareCacheKey`) **MUST** reside in `pkg/workflow/cache_integrity.go`.
2. New cache key computation helpers that depend on integrity level or policy hash **MUST** be added to `cache_integrity.go` and **MUST NOT** be placed in `cache.go` or other cache-adjacent files.
3. `cache.go` **MUST NOT** contain integrity-aware cache key computation logic; it **MUST** be limited to cache entry parsing and default key generation.

### npm / Node.js Step Generation

1. `BuildStandardNpmEngineInstallSteps`, `BuildNpmEngineInstallStepsWithAWF`, and `GetNpmBinPathSetup` **MUST** reside in `pkg/workflow/nodejs.go`.
2. New npm package installation or Node.js path helpers **MUST** be added to `nodejs.go` and **MUST NOT** be placed in `engine_helpers.go`.
3. `engine_helpers.go` **MUST NOT** contain npm package installation or Node.js binary path logic; its scope **MUST** be limited to base installation steps, secret validation, environment variable filtering, and other engine-agnostic helpers.
4. npm-related log statements within `nodejs.go` **MUST** use `nodejsLog` and **MUST NOT** use `engineHelpersLog`.

### Audit Math Helpers

1. `safePercent` and `formatPercent` **MUST** be defined in `pkg/cli/audit_math_helpers.go` and **MUST NOT** be duplicated in other files within `pkg/cli`.
2. Audit rendering code that requires percentage computation or formatting **SHOULD** compose `formatPercent(safePercent(a, b))` using the canonical helpers from `audit_math_helpers.go`.
3. New percentage or ratio computation helpers used across multiple audit files **MUST** be placed in `audit_math_helpers.go`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement — in particular, placing integrity cache key logic in `cache.go`, placing npm helpers in `engine_helpers.go`, or duplicating `safePercent`/`formatPercent` outside `audit_math_helpers.go` — constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24893632435) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
