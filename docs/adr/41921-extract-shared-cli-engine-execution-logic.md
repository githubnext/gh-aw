# ADR-41921: Extract Shared CLI Engine Execution Logic into UniversalLLMConsumerEngine

**Date**: 2026-06-27
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `pkg/workflow` package hosts multiple agentic engine implementations (CrushEngine, OpenCodeEngine, PiEngine) that each compile to GitHub Actions workflow steps. CrushEngine and OpenCodeEngine had ~90 lines of nearly-identical `GetExecutionSteps` logic: building a CLI command with a `run` subcommand, handling firewall-aware AWF wrapping, injecting the standard set of AWF environment variables (prompt path, workspace, MCP config, safe-output, trace context, max-turns, model), and formatting the resulting GitHub Actions step. Maintaining two copies increased the risk of one engine receiving a bug fix or behaviour change that the other did not. Additionally, Pi's `resolvePiBackend` function special-cased the `github-copilot` provider alias with an inline `strings.EqualFold` check, making the alias pattern non-reusable for future engines.

### Decision

We will consolidate the duplicated `GetExecutionSteps` logic into a single `BuildCLIEngineExecutionSteps` method on `UniversalLLMConsumerEngine`, parameterised by a `UniversalCLIEngineExecutionConfig` struct. Engine-specific parameters (binary name, extra CLI flags, permissions config file, step name, model env var name, timestamp behaviour) are passed via the struct; the common firewall-aware command construction, env injection, and step formatting live once in the shared method. We will also extract Pi's `github-copilot` alias lookup into a new `resolveBackendWithAliases` helper that accepts a caller-supplied alias map, making the pattern available to any future engine without ad-hoc string comparisons.

### Alternatives Considered

#### Alternative 1: Keep duplication, fix both engines in lockstep

Each engine retains its own `GetExecutionSteps` implementation. Divergences are managed through code review discipline and cross-referencing comments. This avoids introducing an abstraction but perpetuates the maintenance burden: every future change to the execution pattern (e.g., a new env var, a firewall flag) must be applied in two places and reviewed for both. Given that the two implementations already diverged in minor ways (e.g., `WriteTimestamp` only in Crush), this approach has already shown its fragility and was rejected.

#### Alternative 2: Extract a package-level function rather than a method on UniversalLLMConsumerEngine

The shared logic could live in a standalone function `BuildCLIEngineExecutionSteps(e SomeInterface, workflowData, logFile, cfg)` instead of a method. This keeps the function decoupled from `UniversalLLMConsumerEngine` but requires defining an interface or passing the engine explicitly. Since all CLI engines already embed `UniversalLLMConsumerEngine` (which provides `ApplyUniversalProviderEnv`, `GetUniversalRequiredSecretNames`, etc.), making the shared logic a method on that struct is more idiomatic in Go and avoids a new interface. The method approach was chosen.

### Consequences

#### Positive
- Eliminates approximately 220 lines of duplicated code across CrushEngine and OpenCodeEngine; future engine additions can reuse the pattern with a single call.
- Bug fixes and behaviour changes to the standard execution path (env injection, firewall wrapping, step formatting) now apply to all engines simultaneously.
- `resolveBackendWithAliases` makes provider alias registration declarative and testable independently of any specific engine.
- `UniversalCLIEngineExecutionConfig` makes per-engine variation explicit and visible in one place rather than scattered across two large functions.

#### Negative
- `UniversalCLIEngineExecutionConfig` is a new public struct that must be kept in sync as the execution pattern evolves; adding a new field requires updating all callers.
- Engine-specific flags encoded as struct fields (e.g., `WriteTimestamp: true` for Crush, `false` for OpenCode) are less discoverable than inline conditionals; a reviewer must trace from the config struct back to `BuildCLIEngineExecutionSteps` to understand their effect.
- Engines that diverge significantly from the shared pattern in the future may need to partially bypass `BuildCLIEngineExecutionSteps`, creating awkward partial-use of the abstraction.

#### Neutral
- The `compilerenv` import moves from `crush_engine.go` and `opencode_engine.go` into `universal_llm_consumer_engine.go`; individual engine files become simpler.
- `GetUniversalRequiredSecretNames` is introduced as the method called inside `BuildCLIEngineExecutionSteps`, replacing the per-engine `GetRequiredSecretNames` calls; callers outside this path are unaffected.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
