# ADR-34797: Fast-Path YAML Bypass for Permission Scope-Name Validation

**Date**: 2026-05-26
**Status**: Draft
**Deciders**: pelikhan (copilot-swe-agent)

---

## Part 1 — Narrative (Human-Friendly)

### Context

A validation microbenchmark regressed to ~29.7µs/op (+17.9% vs historical baseline). Profiling identified `ValidatePermissionScopeNames` on the hot path of workflow compilation, dominated by two costs: (a) per-call construction of the full known-scope list (`GetAllPermissionScopes` + `GetAllGitHubAppOnlyScopes` + meta-key map + scope preview string), and (b) `yaml.Unmarshal` on the permissions block, which allocates a generic `map[string]any` even when the input is the trivial common case — a flat block-style mapping of scope → access level. Because permissions blocks in real workflows are nearly always written in block style with a small number of keys, the YAML decoder is doing far more work than the validation actually needs.

### Decision

We will short-circuit `ValidatePermissionScopeNames` with a fast extraction path that reads top-level scope keys directly from block-style YAML without invoking `yaml.Unmarshal`, falling back to the existing YAML-based parsing only when the content uses flow mappings, sequences, or other non-trivial YAML structures (detected by presence of `{`, `}`, `[`, `]`). Reusable validation state — the full scope list, the meta-key set (`all`, `read-all`, `write-all`, `none`), and the preformatted scope preview string used in error messages — is hoisted to package-level `var` initialization so it is built once at startup rather than on every call. Single-key validation logic is centralized in `validateSinglePermissionScopeKey` so the fast and fallback paths share identical error semantics.

### Alternatives Considered

#### Alternative 1: Keep `yaml.Unmarshal` and cache only the package-level state

Hoisting the scope list, meta-key set, and preview string to `var` initialization alone would eliminate the per-call list construction (a meaningful chunk of the regression) without touching the YAML parsing path. This was considered because it is a strictly smaller change with no behavioral risk: every input continues through the same `yaml.Unmarshal` code path. It was rejected because profiling showed `yaml.Unmarshal` of the permissions block was itself a material cost (allocating `map[string]any` and decoder state on every call), and the common-case input is so structurally simple — a flat list of `key: level` lines — that parsing it as full YAML is disproportionate work.

#### Alternative 2: Replace YAML parsing entirely with a hand-rolled parser

Removing `yaml.Unmarshal` from this function altogether and parsing every input — including flow mappings and quoted keys with embedded colons — with custom code would maximize the speed-up. This was rejected because the GitHub Actions YAML spec allows several non-block forms (flow mappings like `{contents: read, issues: write}`, anchors, multi-line scalars) that are tedious and error-prone to reimplement. The hybrid approach — fast path for the easy common case, real YAML decoder for everything else — captures most of the speed-up while preserving exact behavior for edge cases that already work today.

#### Alternative 3: Cache parsed results across calls

A keyed cache from permissions-YAML string to validation result would skip both the YAML parse and the validation work on repeat inputs. This was not chosen because the validator is called from a compile pipeline that already memoizes higher-level work (see ADR-28560), and the inputs vary enough per workflow that a cache hit rate would not justify the cache plumbing, eviction policy, and concurrent-access surface area.

### Consequences

#### Positive
- The hot common case (block-style permissions with a handful of keys) avoids `yaml.Unmarshal` entirely, removing the dominant per-call allocation and decoder cost.
- Package-level cached scope list, meta-key set, and preview string remove the repeated `GetAllPermissionScopes`/`GetAllGitHubAppOnlyScopes` traversals and slice allocations from every validation call.
- `validateSinglePermissionScopeKey` centralizes the meta-key check, exact-match check, case-only suggestion, and fuzzy-match error message — both the fast path and the YAML fallback emit identical errors, removing a class of drift bugs.
- Flow-mapping inputs (`{contents: read, issues: write}`) and quoted-key inputs (`"contents": read`) are covered by added regression tests, so the dual-path design has explicit behavioral pinning.

#### Negative
- There are now two code paths through `ValidatePermissionScopeNames` (hand-rolled fast extraction + YAML fallback). Future changes to validation semantics must be made in both `validateSinglePermissionScopeKey` callers and the fast path's structural assumptions.
- The fast-path detector keys off the literal presence of `{`, `}`, `[`, `]` anywhere in the content. A block-style permissions value containing these characters in a comment (e.g., `# example: {contents: read}`) will unnecessarily fall back to YAML parsing. This is a correctness-preserving false positive, not a behavior change, but it means the speed-up does not apply to inputs with such comments.
- Package-level initialization runs at import time, including `safeAllocationCapacity` and the join over the preview slice. Any future panic in those builders would now manifest at package init rather than first call.

#### Neutral
- The function continues to return `nil` for inputs that are not maps (shorthand strings like `read-all`), matching prior behavior.
- The fast path explicitly accepts quoted keys by trimming `"` and `'` from extracted keys; quoted-key inputs that previously went through `yaml.Unmarshal` are now handled by the fast path with no observable difference.
- `countLeadingSpacesAndTabs` and `extractTopLevelPermissionScopeKeys` are new package-private helpers; their scope is intentionally narrow to this one validator and they are not exported.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Fast-Path Extraction

1. `ValidatePermissionScopeNames` **MUST** attempt fast-path extraction of top-level scope keys via `extractTopLevelPermissionScopeKeys` before invoking `yaml.Unmarshal`.
2. The fast path **MUST** decline (return `parsedFast == false`) when the content contains any of the characters `{`, `}`, `[`, or `]`, so flow mappings and sequences are routed to the YAML fallback.
3. The fast path **MUST** decline when any non-empty, non-comment line does not contain a `:` character at column > 0, or begins with `-` after trimming, so non-mapping inputs are routed to the YAML fallback.
4. When the fast path succeeds, it **MUST** trim surrounding `"` and `'` from each extracted key before validation, so quoted keys are accepted as equivalent to their unquoted form.
5. When the fast path succeeds and all extracted keys validate, `ValidatePermissionScopeNames` **MUST** return `nil` without invoking `yaml.Unmarshal`.

### YAML Fallback Path

1. When the fast path declines, `ValidatePermissionScopeNames` **MUST** fall back to `yaml.Unmarshal` of the content into a `map[string]any`.
2. If `yaml.Unmarshal` returns an error, the function **MUST** return `nil` (treating the input as a non-map shorthand), preserving prior behavior.
3. The fallback **MUST** invoke `validateSinglePermissionScopeKey` for each key in the resulting map, so error messages and suggestions are identical to the fast path.

### Centralized Single-Key Validation

1. Both the fast path and the YAML fallback **MUST** route per-key validation through `validateSinglePermissionScopeKey`; neither path may inline its own meta-key, exact-match, or fuzzy-match logic.
2. `validateSinglePermissionScopeKey` **MUST** accept any key present in `permissionScopeValidationMeta` (currently `all`, `read-all`, `write-all`, `none`) without error.
3. `validateSinglePermissionScopeKey` **MUST** accept any key present in `validPermissionScopes` without error.
4. For unknown keys that differ from a valid scope only by letter case, `validateSinglePermissionScopeKey` **MUST** return an error suggesting the lowercase form.
5. For unknown keys with no case-only match, `validateSinglePermissionScopeKey` **MUST** consult `stringutil.FindClosestMatches` against `permissionScopeValidationAllScopes` with a limit of 3 suggestions, and **MUST** return `nil` (silent accept) when no suggestions are found.

### Package-Level Cached State

1. `permissionScopeValidationAllScopes`, `permissionScopeValidationMeta`, and `permissionScopeValidationValidScopesPreview` **MUST** be initialized once at package load time as package-level `var`s.
2. `ValidatePermissionScopeNames` and `validateSinglePermissionScopeKey` **MUST NOT** rebuild the full scope list, meta-key map, or preview string on each call.
3. The preview string **MUST** contain at most the first 10 entries of `permissionScopeValidationAllScopes` joined by `", "`, followed by `"..."`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26425418484) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
