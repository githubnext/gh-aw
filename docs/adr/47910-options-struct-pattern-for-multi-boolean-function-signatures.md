# ADR-47910: Options Struct Pattern for Multi-Boolean Function Signatures

**Date**: 2026-07-25
**Status**: Draft
**Deciders**: Unknown (automated lint-compliance PR)

---

### Context

The codebase enforces coding standards via a custom linter (`make golint-custom`) that includes rules for maximum function parameter count and maximum function length. Over time, `CheckAndPrepareDockerImages` in `pkg/cli/docker_images.go` accumulated 9 positional boolean parameters — one per Docker-based static-analysis tool (zizmor, poutine, actionlint, runner-guard, syft, grype, grant, yamllint). Positional booleans at call sites are indistinguishable without inspecting the signature, making misorderings silent bugs. Simultaneously, `cmd/gh-aw/main.go` had an `init()` function of ~447 lines and a `compileCmd.RunE` closure of ~129 lines, both far exceeding the linter's function-length threshold. These violations were surfaced as non-shared findings in `make golint-custom`, blocking CI.

### Decision

We will adopt the **Options Struct pattern** for functions that accept more boolean parameters than the linter's configured threshold, and decompose functions that exceed the length limit into focused named helpers.

Concretely:
- `CheckAndPrepareDockerImages` now accepts a single `DockerImagesOptions` struct instead of 9 positional booleans; all callers are updated.
- `compileCmd.RunE` is extracted to `runCompileCmd`, with flag parsing moved to `parseCompileFlags` (returning a `compileFlags` struct) and config assembly moved to `buildCompileConfig`.
- `init()` is decomposed into 10 focused helpers: `setupRootCmdGroups`, `setupRootCmdMeta`, `makeCustomHelpCmd`, `registerCompileFlags`, `setupSetupGroupCmds`, `setupDevelopmentGroupCmds`, `setupExecutionGroupCmds`, `setupAnalysisGroupCmds`, `setupUtilityGroupCmds`, `fixAllSubCmdHelpFlags`.
- The hardcoded container path `/tmp/gh-aw-grant-policy.yaml` is extracted to the named constant `grantContainerPolicyPath`.
- The `defer timer.Stop()` inside a `for` loop in `spawnMCPInspector` is moved outside the `select` block to fix a resource-leak bug.

### Alternatives Considered

#### Alternative 1: Lint suppression directives (`//nolint`)

Add `//nolint:param-count` or equivalent suppression comments at the offending functions to silence the linter without changing the code. This avoids churn and keeps call sites unchanged.

Why not chosen: Suppression discards the signal the lint rule is trying to send. The 9-boolean signature is a genuine readability and correctness hazard — callers cannot verify argument order without reading the signature. Suppression would also set a precedent for silencing violations rather than resolving them.

#### Alternative 2: Raise the linter thresholds

Increase the maximum parameter count and function length limits in the linter configuration so that the existing code passes without modification.

Why not chosen: The existing limits reflect deliberate standards for the project. Relaxing them to accommodate one function would weaken the rules for the entire codebase and invite future growth of already over-complex functions.

### Consequences

#### Positive
- Named struct fields at call sites are self-documenting; readers no longer need to look up parameter order to understand `DockerImagesOptions{Zizmor: true, Grype: true}`.
- Adding a new tool to `DockerImagesOptions` does not require updating every call site (zero-value defaults to `false`).
- Smaller, focused helper functions (`setupSetupGroupCmds`, etc.) are individually testable and easier to review in isolation.
- The defer-in-loop fix eliminates a resource-management bug where `timer.Stop()` would only run at function return rather than per iteration.

#### Negative
- Existing callers of `CheckAndPrepareDockerImages` must be updated to use the struct literal; this is a breaking API change within the package.
- The `compileFlags` struct and its associated `parseCompileFlags`/`buildCompileConfig` functions add an intermediate layer of indirection to the compile path that readers must traverse.
- The `init()` decomposition significantly increases line count in `main.go`, as each helper requires its own function signature, making the file longer despite each function being shorter.

#### Neutral
- The options struct pattern is idiomatic Go; future contributors familiar with the language will recognize it immediately.
- All lint findings fixed in this PR are enforcement of pre-existing rules, not the introduction of new rules or tooling.
- The named constant `grantContainerPolicyPath` is only used in one place today; its value as a constant will become apparent if the path needs to change or be referenced from tests.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
