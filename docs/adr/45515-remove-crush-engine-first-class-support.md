# ADR-45515: Remove Crush Engine First-Class Built-In Support

**Date**: 2026-07-14
**Status**: Draft
**Deciders**: Unknown

---

### Context

GitHub Agentic Workflows (gh-aw) previously supported the Crush coding agent (by Charmbracelet) as a first-class built-in engine alongside Copilot, Claude, Codex, and Gemini. This required dedicated Go constants (`CrushEngine`, `EnvVarModelAgentCrush`, etc.), version pinning (`DefaultCrushVersion`), engine registration in `agentic_engine.go`, validation paths, domain configuration, a gateway config conversion script, a smoke test workflow, and extensive documentation. Crush now supports the same MCP/HTTP interface that gh-aw's custom engine path exposes, meaning users can run Crush workflows without any first-class engine support — they can instead implement it as a shared agentic workflow custom engine definition.

### Decision

We will remove Crush as a built-in engine from gh-aw. All Crush-specific constants, registration code, validation branches, domain helpers, the `convert_gateway_config_crush.sh` setup script, the `smoke-crush` workflow, the `crush_engine_test.go` test file, and all Crush references in documentation and golden files will be deleted. Users who need Crush can adopt the custom engine pattern using a shared agentic workflow definition with `engine.behaviors` (the same mechanism available to any third-party engine), keeping the platform open without maintaining a dedicated integration.

### Alternatives Considered

#### Alternative 1: Keep Crush as a First-Class Built-In Engine

Crush would remain alongside the other built-in engines with continued version pinning, smoke testing, documentation, and Go constants. This was viable when Crush required special handling, but now that custom engine definitions cover the same integration surface, the dedicated code path imposes ongoing maintenance cost (version bumps, smoke test flakiness, documentation drift) with no user-facing benefit that the custom engine path cannot provide.

#### Alternative 2: Deprecate Crush Without Removal (Soft Deprecation)

Mark Crush as deprecated in documentation and emit a warning at compile time when `engine: crush` is used, but retain the code paths for one or more release cycles to give users a migration window. This reduces risk of breakage for existing workflows. It was not chosen because Crush-based workflows in the wild appear limited in scope (the feature was experimental), and maintaining deprecated code alongside custom engine documentation creates confusion about the preferred migration path.

### Consequences

#### Positive
- Reduces the engine constant surface in Go (removes `CrushEngine`, `EnvVarModelAgentCrush`, `EnvVarModelDetectionCrush`, `CrushCLIModelEnvVar`, `DefaultCrushVersion`, `CrushBaseDefaultDomains`, and related helpers), lowering cognitive overhead when adding future engines.
- Eliminates the `smoke-crush` CI workflow and its lockfile (~1 870 lines), the `convert_gateway_config_crush.sh` script (123 lines), and test fixtures, reducing the total lines of code to maintain.
- Engine documentation (feature comparison tables, CLI help strings, FAQ) becomes simpler and no longer requires a Crush column/row.

#### Negative
- Any existing `engine: crush` workflows in user repositories will fail to compile after upgrading to a version of gh-aw that includes this change; users must migrate to a custom engine definition referencing the Crush binary.
- The `smoke-crush` end-to-end validation coverage is removed; regression in the MCP gateway interaction that previously only Crush exercised may go undetected until a user report.

#### Neutral
- The Crush agent itself is unaffected; it can still be invoked via `engine.behaviors` in a shared workflow definition, and the Charmbracelet project continues independently.
- Golden test files that previously included `.crush` in `GH_AW_AGENT_FOLDERS`/`GH_AW_AGENT_FILES` are regenerated, which is a routine side-effect of removing the engine from the engine registry.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
