# ADR-25822: Version-Gated --no-ask-user Flag for Autonomous Copilot Agent Runs

**Date**: 2026-04-11
**Status**: Draft
**Deciders**: pelikhan, Copilot SWE Agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The Copilot agentic engine orchestrates CLI-driven runs on GitHub Actions. Interactive prompts in the Copilot CLI can block fully automated runs, since there is no human operator to respond. The `--no-ask-user` flag was introduced in Copilot CLI v1.0.19 to suppress all interactive prompts, enabling headless/autonomous execution. However, workflows may pin to an explicit Copilot CLI version (e.g., for stability or testing), and emitting `--no-ask-user` against a CLI version that does not recognize the flag causes the run to fail immediately at startup. A mechanism is therefore needed to conditionally emit the flag based on the effective CLI version.

### Decision

We will add a version-gate helper (`copilotSupportsNoAskUser`) that inspects the workflow's `EngineConfig.Version` and returns `true` only when the configured CLI version is ≥ 1.0.19 (the minimum version that accepts `--no-ask-user`). The flag is emitted during argument construction in `GetExecutionSteps` for both agent and detection jobs when the version supports it. Unspecified versions and `"latest"` are treated as always supported; non-semver strings (e.g., branch names) are treated conservatively as unsupported. This pattern mirrors the existing `awfSupportsExcludeEnv` and `awfSupportsCliProxy` helpers, extending the established version-gate convention in this codebase.

### Alternatives Considered

#### Alternative 1: Always emit --no-ask-user unconditionally

Emit the flag for all agent jobs regardless of the configured CLI version. This is simpler (no version check) and ensures all runs benefit from autonomous mode. It was rejected because it would immediately break any workflow pinned to a CLI version older than 1.0.19: the CLI treats unknown flags as fatal errors, which would block those users from running at all without a forced version upgrade.

#### Alternative 2: Require callers to opt in via an explicit workflow configuration key

Add an `noAskUser: true` field to the workflow YAML or `EngineConfig` struct, requiring workflow authors to enable the flag explicitly. This gives authors full control. It was rejected because the flag is unconditionally desirable for any automated run — there is no scenario where a modern CLI version should retain interactive prompts in a CI context — making explicit opt-in unnecessary friction without safety benefit.

#### Alternative 3: Use a runtime CLI capability probe (--help parsing)

Detect support by invoking the CLI with `--help` at workflow generation time and parsing the output for `--no-ask-user`. This would be perfectly accurate without hard-coding a version number. It was rejected because it introduces a subprocess execution dependency at step-generation time (which currently has no external side effects), adds latency, and creates a brittle dependency on CLI help text format. A semver comparison against a well-known constant is simpler, auditable, and easy to update.

### Consequences

#### Positive
- Fully autonomous agent runs are enabled by default for all workflows using Copilot CLI ≥ 1.0.19, eliminating the risk of runs hanging on interactive prompts.
- The version-gate pattern is consistent with existing helpers (`awfSupportsExcludeEnv`, `awfSupportsCliProxy`), reducing cognitive overhead for contributors familiar with the codebase.
- Conservative handling of non-semver version strings ensures the flag is never emitted in ambiguous cases, protecting against unexpected CLI behavior.

#### Negative
- A version constant (`CopilotNoAskUserMinVersion = "1.0.19"`) must be kept accurate as the CLI evolves; if a regression causes an older version to also accept the flag, the constant would be unnecessarily restrictive (though not harmful).
- The detection of whether a job is an "agent job" vs. a "detection job" relies on the sentinel `SafeOutputs == nil`, which is an implicit convention rather than an explicit job-type enum. Future refactors of `WorkflowData` must preserve this invariant or update the guard.

#### Neutral
- The `isDetectionJob` variable declaration was hoisted earlier in `GetExecutionSteps` to be shared by both the `--no-ask-user` block and the existing `--autopilot` block. This is a pure refactor with no behavioral change.
- Lock files were regenerated as part of the dependency graph update triggered by the code change.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Flag Emission

1. Implementations **MUST** emit `--no-ask-user` to the Copilot CLI argument list for both agent and detection jobs when the effective CLI version is ≥ `CopilotNoAskUserMinVersion` (currently `1.0.19`).
2. Implementations **MUST NOT** emit `--no-ask-user` when the effective CLI version is a semver string strictly less than `CopilotNoAskUserMinVersion`.
3. Implementations **MUST NOT** emit `--no-ask-user` when the effective CLI version is a non-semver string (e.g., a branch name), treating such cases conservatively as unsupported.
4. Implementations **MUST** treat an unspecified version (nil `EngineConfig` or empty `EngineConfig.Version`) as supported, since `DefaultCopilotVersion` is always ≥ `CopilotNoAskUserMinVersion`.
5. Implementations **MUST** treat the literal string `"latest"` (case-insensitive) as supported, since `latest` always resolves to a current release.

### Version Gate Helper

1. The version-gate logic **MUST** be encapsulated in a dedicated helper function (`copilotSupportsNoAskUser`) following the same pattern as `awfSupportsExcludeEnv` and `awfSupportsCliProxy`.
2. The minimum supported version **MUST** be defined as the named constant `CopilotNoAskUserMinVersion` in `pkg/constants/version_constants.go`; it **MUST NOT** be inlined as a string literal at call sites.
3. When `CopilotNoAskUserMinVersion` is updated, the corresponding unit tests **SHOULD** be updated to reflect the new boundary version.

### Testing

1. The flag emission behavior **MUST** be covered by integration-style tests that exercise `GetExecutionSteps` with the full matrix of version and job-type combinations.
2. The version-gate helper **MUST** be covered by unit tests that include: nil config, empty version, `"latest"`, the exact minimum version, one version above the minimum, one version below the minimum, and a non-semver string.
3. Tests **MUST NOT** use mocks for the version comparison logic; the actual `compareVersions` function **MUST** be exercised.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. In particular: `--no-ask-user` **MUST** be gated on CLI version (≥ 1.0.19 or unspecified/latest). Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
