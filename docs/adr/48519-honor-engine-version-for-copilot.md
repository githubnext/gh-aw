# ADR-48519: Honor engine.version for Copilot CLI Installation

**Date**: 2026-07-28
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `engine.version` field in workflow configuration allows users to pin the version of a coding engine (Copilot, Claude, codex, gemini) installed during workflow execution. Claude, codex, and gemini engines all honored this field correctly, using the user-specified value when present and falling back to a default pinned constant otherwise. However, the Copilot engine's `GetInstallationSteps` silently ignored `engine.version` and always installed `DefaultCopilotVersion`, logging only a message that it was ignoring the user-set value. This inconsistency made `engine.version` a no-op for Copilot, violating the principle of least surprise and breaking user expectations established by the other three engine implementations.

### Decision

We will make the Copilot engine honor `engine.version` exactly as Claude, codex, and gemini do: when `EngineConfig.Version` is set, use it as the installation target; when it is absent, fall back to `DefaultCopilotVersion` and normalize `EngineConfig.Version` to that default so all downstream consumers in the same compile flow observe the effective installed value. The install script already skips `compat.json` resolution when an explicit version is provided, so no additional bypass logic is required.

### Alternatives Considered

#### Alternative 1: Keep Copilot always pinned to DefaultCopilotVersion

The existing behavior guaranteed a stable, tested Copilot CLI version regardless of user configuration, which reduced risk of users accidentally installing untested or incompatible versions. This was rejected because it made `engine.version` silently ineffective for Copilot while working for all other engines, creating inconsistent and confusing behavior. The field's documented contract implies it controls the installed version across all engines.

#### Alternative 2: Introduce a Copilot-specific version override field

A dedicated field (e.g., `engine.copilot_version`) could allow Copilot-specific version control without changing the behavior of `engine.version`. This was rejected because it adds unnecessary API surface, conflicts with the established cross-engine convention of `engine.version`, and would require documentation and migration for any users who expect the unified field to work.

### Consequences

#### Positive
- `engine.version` now behaves consistently across all four supported engines (Copilot, Claude, codex, gemini), fulfilling user expectations.
- Users can pin or override the Copilot CLI version in their workflow definitions, enabling version-locked reproducible builds or controlled rollouts of new CLI versions.
- Test coverage explicitly asserts the honored-version contract, preventing silent regression back to the ignore-version behavior.

#### Negative
- `compat.json` compatibility-matrix resolution is skipped when an explicit `engine.version` is provided, meaning users who specify an incompatible or unsupported version will receive no automated correction and may encounter runtime failures.
- Any workflow that previously set `engine.version` for Copilot expecting it to be ignored (e.g., to document an intent without changing behavior) will now have that version actually installed — a silent behavioral change at the call site.

#### Neutral
- `EngineConfig.Version` is still normalized to the effective installed version when no explicit version is set, preserving the existing contract for downstream consumers in the compile flow.
- The change is contained to `copilot_engine_installation.go`; the install script and `compat.json` integration path are unchanged.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
