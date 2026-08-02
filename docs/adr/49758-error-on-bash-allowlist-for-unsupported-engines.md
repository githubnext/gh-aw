# ADR-49758: Compile-Time Error for Bash Allowlist on Unsupported Engines

**Date**: 2026-08-02
**Status**: Draft
**Deciders**: Unknown

---

### Context

The agentic workflow framework (AWF) allows users to configure a `tools.bash` allowlist in workflow frontmatter (e.g., `bash: [git, npm]`) to restrict which shell commands an AI agent may execute. For the Codex engine, this allowlist was silently accepted during compilation but never enforced at runtime: the Codex CLI has no mechanism to translate the list into restricted shell access, and workflows run with `--dangerously-bypass-approvals-and-sandbox`. This created a false sense of security — the config looked enforced but wasn't. Other engines (Claude, Copilot, Gemini, Antigravity) do map the allowlist to their CLI's tool-restriction flags, so the gap is Codex-specific.

### Decision

We will add a `BashCommandAllowlist` boolean field to `EngineCapabilities` and `EngineCapabilitiesDefinition` that each engine sets to reflect whether it can enforce a restricted bash allowlist. We will introduce `validateBashCommandAllowlistSupport()` — a compile-time validator that errors when a non-wildcard `tools.bash` list is used with an engine whose `BashCommandAllowlist` capability is `false`. Wildcard (`bash: ["*"]`), boolean (`bash: true`), and absent `bash` config remain valid for all engines. The validator is wired into the existing `validateEngineToolRequirements()` chain.

### Alternatives Considered

#### Alternative 1: Emit a Warning Instead of an Error

Surface the unsupported allowlist as a compiler warning rather than a hard error, allowing the workflow to compile and run. This lets existing Codex workflows continue without breakage.

Not chosen because a warning preserves the security illusion: users who set a restricted allowlist believe the restriction is enforced. A silent warning risks being ignored in CI output, leaving the misconfiguration in place indefinitely.

#### Alternative 2: Silently Coerce to Wildcard

When the engine is Codex (or any other engine that cannot enforce an allowlist), automatically replace the specific command list with `["*"]` (allow all) and log a debug message. The workflow compiles and runs, but the allowlist is not applied.

Not chosen because it changes user intent without acknowledgement. A user who configured `bash: [git, npm]` explicitly wanted restriction; silently widening to all commands is a correctness violation that is harder to discover than a compile error. It also does not help users migrate to an engine that actually enforces the restriction.

### Consequences

#### Positive
- Users receive an explicit, actionable compile-time error rather than discovering the security gap at runtime or through a security review.
- The engine capability system gains a structured, per-feature flag (`BashCommandAllowlist`) that generalises the pattern used for `BareMode`, `WebSearch`, and other per-engine features — future enforcement gaps can be expressed the same way.
- The error message names engines that do support the feature, guiding users toward a path forward without requiring documentation lookups.

#### Negative
- Existing Codex workflows that configured `tools.bash` with specific commands will fail to compile and require user action (change to `bash: ["*"]`, remove the entry, or switch engines).
- The `BashCommandAllowlist` capability flag must be kept accurate for each engine; if a future engine adds enforcement support, someone must update the flag or the compile error will persist for that engine.

#### Neutral
- The Codex engine's `BashCommandAllowlist` defaults to `false` (the zero value for bool), so newly registered engines that omit the field will correctly fail-safe and trigger the validator.
- The change is guarded by the existing `hasBashRestrictedAllowlist()` helper, so the validation path is exercised by the same condition that was already computing allowlist behavior.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
