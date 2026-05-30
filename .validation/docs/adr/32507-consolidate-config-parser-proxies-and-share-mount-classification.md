# ADR-32507: Consolidate Workflow Config Parser Proxies and Share Mount Classification

**Date**: 2026-05-16
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

Semantic clustering of `pkg/workflow` surfaced two sources of churn. First, a family of thin parser proxies (`parseTitlePrefixFromConfig`, `parseTargetRepoFromConfig`, `parseAllowedLabelsFromConfig`, `parseAllowedReposFromConfig`, `parseRequiredLabelsFromConfig`, `parseRequiredTitlePrefixFromConfig`, and `extractStringSliceFromConfig`) existed only to call one of two shared primitives — `extractStringFromMap(...)` or `ParseStringArrayFromConfig(...)` — with a fixed key string. Each added a layer of indirection without adding validation or transformation. Second, mount-string validation duplicated the same "format-error / mode-error / empty-source / empty-destination" branching across `sandbox_validation.go` and `mcp_mount_validation.go`, even though each caller needed to produce a different error surface (`NewValidationError` vs. `fmt.Errorf`). The two duplications had different shapes but the same root cause: shared classification logic was either being re-implemented per caller or hidden behind a thin wrapper instead of exposed directly.

### Decision

We will inline calls to the shared primitives at safe-output parsing call sites and remove the fixed-key wrapper helpers, while introducing a single shared classifier — `parseMountEntry(mount) (mountParts, mountValidationKind)` — for mount validation. Call sites that previously used a parser proxy now invoke `extractStringFromMap(configMap, "key", log)` or `ParseStringArrayFromConfig(configMap, "key", log)` directly, which makes the configuration key explicit at every call site. Sandbox and MCP mount validators both call `parseMountEntry` and then `switch` on the returned `mountValidationKind` enum to construct their respective error types, eliminating duplicated parsing while preserving each caller's error contract.

### Alternatives Considered

#### Alternative 1: Keep the Thin Wrappers

Leave `parseTitlePrefixFromConfig`, `parseTargetRepoFromConfig`, etc. in place and accept the indirection as the cost of "named" call sites. This was rejected because the wrappers carry no validation or transformation logic — they only encode a key string and a logger — and they accumulate faster than they are removed, fragmenting the surface area of `config_helpers.go` over time.

#### Alternative 2: Share Mount Validation Behind a Single Error-Returning Function

Replace `parseMountEntry` with a function that returns a fully formatted error directly, removing the `switch` from each caller. This was not chosen because the two callers genuinely need different error types (structured `NewValidationError` for sandbox, plain `fmt.Errorf` for MCP) and different field paths (`sandbox.mounts[i].source` vs. `tool 'X' mcp configuration mounts[i] source path`). Pushing formatting into the shared helper would either require leaking caller-specific knobs into its signature or accepting a less informative error message in one of the two surfaces.

#### Alternative 3: Extract a Higher-Level "Config Field" Abstraction

Introduce a type that represents a single config field (key + type + logger + validation) and have call sites declare fields, then ask a registry to parse them. This was rejected as scope creep: the immediate need is to remove dead indirection and de-duplicate mount classification, not to design a new declarative configuration framework on top of the existing helpers.

### Consequences

#### Positive
- The configuration key for each parsed field is now visible at the call site, making it easier to grep for which fields are read where.
- `config_helpers.go` and `safe_outputs_parser.go` shrink and stop accumulating fixed-key wrappers, reducing helper sprawl.
- Mount validation logic lives in one place; adding a new `mountValidationKind` (e.g., a future empty-mode check) requires changes to a single classifier rather than two parallel validators.
- Each mount validator keeps its caller-specific error contract (structured vs. plain) intact.

#### Negative
- Call sites are slightly longer than they were with the wrapper helpers, because the key string and the logger appear inline at every read.
- Callers must remember to pass the appropriate logger argument; previously the wrapper hard-coded `configHelpersLog`, now mixed loggers (`updateIssueLog`, `pushToPullRequestBranchLog`, `submitPRReviewLog`, etc.) appear at the call sites and can be misrouted.
- The two mount validators are now coupled to the `mountValidationKind` enum: adding a new kind requires updating every `switch` statement, or risk a silent fall-through.

#### Neutral
- No external behaviour change: error messages, validation outcomes, and configuration semantics are preserved.
- Test coverage shifts onto the shared primitives (`extractStringFromMap`, `ParseStringArrayFromConfig`, `parseMountEntry`) rather than the removed wrappers; the test rename `TestParseAllowedReposFromConfig → TestParseAllowedRepos` reflects this.
- Empty-source and empty-destination cases for MCP mount validation are now explicitly covered by tests, where previously they relied on the shared parser's behaviour without dedicated assertions.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Config Parser Helper Design

1. New configuration-reading helpers in `pkg/workflow` **MUST NOT** be introduced solely to bind a fixed key string and logger to an existing shared primitive (`extractStringFromMap`, `ParseStringArrayFromConfig`, or `ParseStringArrayOrExprFromConfig`).
2. A new wrapper function over a shared primitive **MAY** be introduced when it performs additional validation, transformation, or normalisation beyond key + logger binding (for example, `parseTargetRepoWithValidation`, which rejects the wildcard `*`).
3. Call sites that read a configuration field **MUST** pass the configuration key as a string literal argument to the shared primitive and **MUST** pass a logger appropriate to the call site's package context.
4. Code paths that need silent (non-logging) extraction of a `[]string` value **MUST** call `ParseStringArrayFromConfig(configMap, key, nil)` rather than introducing a dedicated silent helper.

### Mount Validation Architecture

1. Mount string parsing and classification **MUST** be performed by the shared `parseMountEntry(mount) (mountParts, mountValidationKind)` function in `validation_helpers.go`.
2. Callers of `parseMountEntry` **MUST** branch on the returned `mountValidationKind` to construct caller-specific errors, and **MUST NOT** re-implement the format / mode / empty-source / empty-destination classification themselves.
3. The shared classifier **MUST NOT** format user-facing error messages; error message construction **MUST** remain at the call site so that the sandbox and MCP validators can preserve their distinct error surfaces.
4. New mount-validation failure modes **MUST** be expressed as new `mountValidationKind` constants, and every existing caller's `switch` over `mountValidationKind` **MUST** be updated to handle the new constant.
5. Sandbox mount validation **MUST** continue to return errors via `NewValidationError(...)`; MCP mount validation **MUST** continue to return errors via `fmt.Errorf(...)`.

### Test Coverage

1. The shared primitives `extractStringFromMap`, `ParseStringArrayFromConfig`, and `parseMountEntry` **MUST** have unit tests covering their primary success and failure cases.
2. Tests that previously exercised a removed thin wrapper **MUST** be re-pointed to the underlying shared primitive rather than deleted.
3. Each `mountValidationKind` constant **SHOULD** have at least one test case in `TestParseMountEntry` exercising the corresponding classification path.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25949009950) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
