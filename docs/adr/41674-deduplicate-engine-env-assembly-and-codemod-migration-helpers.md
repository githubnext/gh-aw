# ADR-41674: Deduplicate Engine Env Assembly and Codemod Migration Helpers

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: Unknown (automated refactor by copilot-swe-agent)

---

### Context

The three agentic engine implementations (Claude, Codex, and Copilot) each contained near-identical ~20-line inline blocks responsible for setting tool-timeout env vars (`GH_AW_STARTUP_TIMEOUT`, `GH_AW_TOOL_TIMEOUT`), max-turns env resolution (`GH_AW_MAX_TURNS`), engine/agent env merges, and MCP-scripts secret passthrough. Similarly, two codemods for migrating deprecated `engine.max-runs` and `engine.max-turns` to top-level YAML keys shared ~80 lines of near-identical frontmatter transformation logic. Any change to env-assembly semantics — such as adding a new runtime env var — required the same edit in three separate engine files. The `FormatStepWithCommandAndEnv` function also used a bespoke ad-hoc sort instead of the existing `sliceutil.SortedKeys` utility.

### Decision

We will extract the repeated env-assembly logic into four shared helper functions in `pkg/workflow/engine_helpers.go` (`applyOptionalEngineToolTimeouts`, `applyEngineMaxTurnsEnv`, `applyEngineAndAgentEnv`, `applyMCPScriptsSecretEnv`) and introduce a single `migrateEngineFieldToTopLevel` helper in `pkg/cli/codemod_engine_to_top_level_helpers.go` that both codemods delegate to. We will also replace the ad-hoc key-sort in `FormatStepWithCommandAndEnv` with `sliceutil.SortedKeys`. This centralises shared behavior without changing runtime semantics or user-visible output.

### Alternatives Considered

#### Alternative 1: Keep Inline Per-Engine Logic (Status Quo)

Each engine file retains its own copy of the env-assembly blocks. The approach is maximally self-contained: reading any single engine file gives the full picture of what env vars it sets. This was rejected because the duplication made it error-prone to keep the three engines in sync when semantics change, as demonstrated by the three identical blocks already being the dominant maintenance burden called out in the PR description.

#### Alternative 2: Interface-Based Polymorphism

Define an `EngineEnvApplier` interface with an `ApplyCommonEnv(env map[string]string, wd *WorkflowData)` method, then have each engine embed a common implementation. This is a valid Go pattern for shared behavior across types, but it adds interface indirection for what is essentially stateless, side-effecting helper logic with no need for substitutability. Package-level functions are simpler and sufficient here.

### Consequences

#### Positive
- Single source of truth for each env-assembly concern; future additions (e.g., a new `GH_AW_*` env var) require exactly one code change.
- Net reduction of ~160 lines of duplicated code across the three engine files and two codemod files.
- `FormatStepWithCommandAndEnv` uses the shared `sliceutil.SortedKeys` utility consistently with other callers in the codebase.
- `migrateEngineFieldToTopLevel` is parameterised and reusable for any future `engine.*` → top-level migration codemods.

#### Negative
- New engine implementations must know to call the shared helpers; this implicit contract is not enforced by the type system.
- The generic `migrateEngineFieldToTopLevel` signature (9 parameters) is harder to read in isolation than each of the original self-contained codemods, increasing cognitive overhead for understanding a single migration path.

#### Neutral
- The refactor is behavior-preserving: no change to env-var names, values, or precedence order; existing tests continue to exercise the same code paths via the new helper call sites.
- The `docs/adr/` directory convention is being adopted concurrently with this PR, so this ADR is the first to use the PR-number-as-ADR-number naming scheme.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
