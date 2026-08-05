# ADR-50597: Inject Engine Version as Environment Variable into Behavior-Defined Engine Execution Steps

**Date**: 2026-08-05
**Status**: Accepted
**Deciders**: gh-aw maintainers

---

### Context

The gh-aw platform supports "behavior-defined" custom engines (Goose, Ader, Crush, etc.) where users specify an `engine.version` field in their workflow configuration. This version value is parsed and used during engine installation but was never propagated into the execution environment of the run step itself. As a result, engine scripts and install helpers had no reliable way to access the user-specified version at runtime without re-parsing the workflow configuration frontmatter directly. The platform already exposes other engine properties (`GH_AW_ENGINE_CWD`, `GH_AW_ENGINE_MAX_TURNS`, etc.) as environment variables following a consistent `GH_AW_*` naming convention. Audit information (`GH_AW_INFO_VERSION` / `GH_AW_INFO_AGENT_VERSION`) was already populated for safe-outputs jobs via `getInstallationVersion`, but this did not cover the execution step environment for behavior-defined engines.

### Decision

We will add a new `applyEngineVersionEnv` helper in `engine_helpers.go` that sets `GH_AW_ENGINE_VERSION` from `EngineConfig.Version` when the field is non-empty. The main agent job receives this environment variable so engine installation and execution steps share the same configured version. Behavior-defined engine execution retains the explicit environment injection alongside `applyEngineCwdEnv`. The helper passes the version value verbatim — including GitHub Actions expression syntax (e.g., `${{ inputs.engine-version }}`) — without attempting evaluation, consistent with how other environment helpers in this layer operate.

### Alternatives Considered

#### Alternative 1: Require Scripts to Parse Workflow Frontmatter Directly

Engine scripts could read the workflow YAML file and extract the `engine.version` field themselves. This is the status quo for consumers who need the version today. It couples engine scripts to the internal configuration schema, is fragile across schema changes, and duplicates parsing logic already handled by the platform. It also fails for GHA expression values, which are not resolved in workflow YAML at the time the script runs.

#### Alternative 2: Expose the Full EngineConfig as a Serialized JSON Environment Variable

Rather than a single `GH_AW_ENGINE_VERSION` variable, serialize the entire `EngineConfig` struct into a JSON-encoded env var (e.g., `GH_AW_ENGINE_CONFIG_JSON`). This would give scripts access to all engine configuration fields without requiring individual helper functions per field. However, it increases the attack surface for environment variable injection if the EngineConfig grows to include sensitive fields, leaks internal schema details to script authors, and creates a larger compatibility surface area to maintain. The targeted single-field approach follows the established pattern in the codebase.

### Consequences

#### Positive
- Engine scripts and install helpers can access the user-configured engine version at runtime without parsing configuration files.
- The implementation is consistent with the existing `GH_AW_*` environment variable convention, reducing cognitive overhead for engine authors.
- Both literal version strings and GHA expression pass-through values (`${{ inputs.engine-version }}`) are supported.
- Full unit and integration test coverage is included, matching the testing discipline of adjacent helpers.

#### Negative
- GHA expressions (e.g., `${{ inputs.engine-version }}`) are injected verbatim into the environment; they are not evaluated by the platform, which may surprise users who expect a resolved version string at runtime. This behavior must be documented for engine authors.
- Adds one more environment variable to the execution namespace. While well-namespaced under `GH_AW_`, this incrementally increases the surface of the implicit contract between the platform and engine scripts.

#### Neutral
- The `GH_AW_INFO_VERSION` / `GH_AW_INFO_AGENT_VERSION` audit fields in safe-outputs jobs are unaffected and continue to use `getInstallationVersion` — this change only fills the gap for the execution step environment.
- The pattern mirrors the `applyEngineCwdEnv` function added previously; no new architectural abstractions are introduced.

---
