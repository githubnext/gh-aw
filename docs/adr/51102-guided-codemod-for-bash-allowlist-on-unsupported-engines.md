# ADR-51102: Guided (Non-Auto-Rewriting) Codemod for Restricted `tools.bash` on Engines That Ignore Allow-Listing

**Date**: 2026-08-07
**Status**: Draft
**Deciders**: Unknown

---

### Context

`gh aw compile --strict` rejects workflows that pair an engine without bash allow-list support (e.g. `codex`) with a restricted `tools.bash` configuration (`bash: [cmd, ...]`, `bash: []`, or `bash: false`). However, `gh aw fix` reported "No fixes needed" for this exact configuration, leaving users with a broken workflow and no remediation path. The incompatibility arises because the engine silently ignores the allow-list at runtime, making the declared restriction meaningless. Two remediations exist — widen to `bash: ["*"]` or switch to a supporting engine — but both change workflow semantics and require human judgment.

### Decision

We will add a new guided codemod (`bash-allowlist-unsupported-engine-guided-error`) that detects the restriction pattern, checks the engine's `BashCommandAllowlist` capability from the global engine registry, and emits a descriptive guided error naming both fix options. The codemod never modifies the workflow file; it always returns `applied=false`. The check is capability-driven rather than engine-name-hardcoded so it stays correct as engines are added or changed. The existing `hasBashExplicitRestriction` function in `pkg/workflow/agent_validation.go` is exported as `HasBashExplicitRestriction` to be shared between the compiler and the new codemod, preventing logic drift.

### Alternatives Considered

#### Alternative 1: Auto-rewrite `tools.bash` to `bash: ["*"]`

Auto-correction would silently widen the effective bash permissions granted to the agent, changing what the workflow author explicitly declared. A workflow author who wrote `bash: ["git", "npm"]` intended to restrict bash access; overwriting this to unrestricted access without consent could introduce a security regression. This option was rejected because it changes semantics without user consent.

#### Alternative 2: Auto-switch the `engine` field to a supported engine

Automatically changing the engine (e.g. from `codex` to `copilot`) would change which AI agent executes the workflow, altering its behavior in ways unrelated to the bash restriction. The author may have chosen `codex` for reasons beyond bash tooling. This option was rejected because it modifies a high-impact field outside the scope of the bash restriction problem.

### Consequences

#### Positive
- Users get a clear, actionable guided error from `gh aw fix` that names both remediation paths, ending the "No fixes needed" false negative.
- The capability-driven check (via `engine.GetCapabilities().BashCommandAllowlist`) keeps the detection accurate without per-engine hardcoding, so new engines that lack allow-list support are caught automatically.
- Sharing `HasBashExplicitRestriction` between the compiler and the codemod eliminates the risk of the two checks diverging over time.

#### Negative
- Users still must perform the fix manually; no automatic remediation is provided, which may frustrate users expecting `gh aw fix` to resolve issues without human intervention.
- The guided error model requires the codemod framework to support `Guided: true` codemods that return errors without mutations — a subtler contract than standard codemods.

#### Neutral
- The new codemod is registered after the other bash codemods (`bash-anonymous-removal`, `bash-single-quoted-args-rewrite`) to maintain grouping by concern.
- Unknown or custom engines are treated as a no-op to avoid false positives in environments with private engine registries.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
