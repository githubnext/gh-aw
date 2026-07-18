# ADR-46367: Consolidate Antigravity and Gemini Engines into a Shared `googleCLIEngine` Base

**Date**: 2026-07-18
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `GeminiEngine` and `AntigravityEngine` implementations were near-verbatim copies of each other across five file pairs, sharing ~90–95% identical code. Despite the obvious similarity, the two had already diverged in behavior: `computeGeminiToolsCore` used an inline two-pass bash-mapping loop, while `computeAntigravityToolsCore` had already been refactored to the canonical single-pass `appendBashTools` helper — a live behavioral inconsistency affecting tool allowlist generation. Both engines implement the `CodingAgentEngine` interface and share the same CLI invocation pattern (API key secret, CLI binary, config directory, log parser, MCP config rendering, secret filtering, etc.), differing only in constants and one installation method (npm vs. GCS binary).

### Decision

We will introduce a `googleCLIEngine` struct parameterized by a `googleCLIEngineConfig` config value that captures all per-engine constants (API key name, CLI binary, CLI flags, config directory, env var names, log parser identity, secret mirroring, etc.). Both `GeminiEngine` and `AntigravityEngine` will embed `googleCLIEngine` via Go struct embedding, inheriting the 13 shared methods. Each engine retains only `GetInstallationSteps`, which differs in mechanism (npm vs. GCS binary download). Four files (`antigravity_mcp.go`, `antigravity_logs.go`, `gemini_mcp.go`, `gemini_logs.go`) are deleted because their sole methods are now promoted from `googleCLIEngine`.

### Alternatives Considered

#### Alternative 1: Keep Separate Implementations, Establish a Sync Process

Maintain the two independent engine implementations and introduce a code-review policy requiring that changes to one engine be mirrored to the other within the same PR. This avoids any structural change to the codebase.

This was rejected because the behavioral drift (`computeGeminiToolsCore` two-pass vs. `computeAntigravityToolsCore` single-pass) demonstrates that manual sync policies fail in practice. Any future contributor adding a feature to one engine would need to remember to apply it to the other — a requirement that is not enforced by the compiler and will recur.

#### Alternative 2: Interface-Based Composition via Constructor Injection

Rather than struct embedding, extract the shared logic into standalone functions and pass them (or a helper object) into each engine's constructor. The engines would remain structurally independent but delegate shared operations to a common implementation.

This was rejected because it requires duplicating method signatures on both engine types, adds indirection without eliminating the risk of divergence (the function signatures must still be called identically in both engines), and is less idiomatic Go for this pattern. Struct embedding provides zero-overhead promotion of methods and makes interface satisfaction verifiable by the compiler (`var _ CodingAgentEngine = (*GeminiEngine)(nil)`).

### Consequences

#### Positive
- Behavioral drift between the two engines is eliminated by construction: all 13 shared methods have exactly one implementation, and any change automatically applies to both engines.
- Code volume is reduced significantly: four files are deleted (`antigravity_mcp.go`, `antigravity_logs.go`, `gemini_mcp.go`, `gemini_logs.go`), and each engine file is reduced to a constructor plus one method.
- Adding a future third Google CLI engine (or onboarding a new engine with the same CLI pattern) requires only populating a `googleCLIEngineConfig` struct and implementing `GetInstallationSteps`.
- The `mirrorAPIKeyAs` field in `googleCLIEngineConfig` provides a first-class, documented mechanism for the Antigravity → Gemini API key mirroring behavior, making it explicit rather than buried in `GetExecutionSteps`.

#### Negative
- Struct nesting is deeper: `AntigravityEngine` embeds `googleCLIEngine` which embeds `BaseEngine`, making field access paths longer (e.g., `e.cfg.log.Printf(...)` instead of `antigravityLog.Printf(...)`).
- `googleCLIEngineConfig` is a large struct with ~20 fields; future contributors must populate it correctly when adding a new engine, with no compile-time enforcement of required vs. optional fields.
- Go embedding promotes all `googleCLIEngine` methods onto both engine types silently — the promoted method set is not obvious from the engine type's own file, which may surprise contributors unfamiliar with Go embedding.

#### Neutral
- Existing test function signatures (`computeAntigravityToolsCore`, `computeGeminiToolsCore`) are preserved as thin wrappers delegating to `computeGoogleCLIToolsCore`, avoiding test churn while exposing the shared implementation.
- The `generateAntigravitySettingsStep` and `generateGeminiSettingsStep` methods are replaced by the unified `generateSettingsStep` on `googleCLIEngine`; callers in test files are updated to call the new name.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
