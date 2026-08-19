# ADR-53964: Prefer the Restrictive Safe Default When Codemods Encounter Ambiguous Security Configurations

**Date**: 2026-08-19
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`gh aw fix --write` applies a sequence of registered codemods to repair workflow files that fail strict-mode compilation. Two gaps caused the fix pass to leave files unrepaired:

1. The `sandbox-runtime-profiles` codemod hard-errored (and aborted the entire file's fix pass) when it encountered `sandbox.agent.runtime: gvisor` combined with `sudo: true` or `legacy-security: enable`. This combination is no longer supported, but gVisor and privileged options have conflicting intent — gVisor enforces strict network isolation while `sudo`/`legacy-security` request elevated host access.

2. No codemod existed for the strict-mode requirement that `tools.bash` must be explicitly specified when `tools.github.min-integrity: none`. Files with this configuration reported "No fixes needed" from `gh aw fix --write` yet still failed `--strict` compilation, silently blocking cross-repo audits.

Both gaps were reproduced across multiple independently-verified external repositories (e.g. `github/gh-aw-firewall`, `github/gh-aw-mcpg`, `chrizbo/agentics-beyond-code`) during the daily compilation audit.

### Decision

We will resolve ambiguous security configurations by **choosing the more restrictive safe default and auto-applying the fix** rather than aborting or requiring manual intervention:

- For `runtime: gvisor` combined with `sudo`/`legacy-security`: keep `runtime: gvisor` (the stricter isolation) and drop the incompatible privileged fields. This lets `gh aw fix --write` complete the file instead of aborting.
- For `min-integrity: none` without explicit `tools.bash`: insert `tools.bash: false`. This preserves the pre-existing behavior (bash was never configured) while satisfying the strict-mode requirement.

The guiding principle is that when a configuration is ambiguous, the codemod should not block the fix pass — it should apply the change that is safest and most likely correct, and log what it did so the author can review.

### Alternatives Considered

#### Alternative 1: Migrate gVisor + privileged to `docker-sudo-iptables`

Rewrite `runtime: gvisor` to `runtime: docker-sudo-iptables` when privileged options are present, on the grounds that the author's intent was privileged access and gVisor was incidental.

Not chosen because gVisor is an explicit runtime choice that signals a deliberate preference for strict network isolation. Silently downgrading isolation to satisfy a `sudo` flag would be a security regression and harder to review. Dropping the privileged fields is the smaller, more auditable change.

#### Alternative 2: Keep aborting with an actionable error (previous behavior for gVisor)

Continue returning an error that names the two choices and requires the author to resolve manually.

Not chosen because this leaves the file completely untouched by `gh aw fix --write` — every other codemod that would have applied to the same file is also skipped. The actionable error approach scales poorly when the same pattern appears across many external repos during automated audits.

#### Alternative 3: No codemod for `min-integrity: none` + missing `tools.bash`; require manual fix

Keep the existing behavior where `gh aw fix --write` reports "No fixes needed" and let authors add `tools.bash` themselves.

Not chosen because `tools.bash: false` is a safe, behavior-preserving default (bash was not configured before) and the strict-mode requirement is mechanical. Requiring manual action for a deterministic, zero-ambiguity fix creates unnecessary friction at scale.

#### Alternative 4: Insert `tools.bash: true` instead of `false` for the `min-integrity: none` codemod

Explicitly allow bash when min-integrity is none, arguing that the workflow might need shell access.

Not chosen because this changes behavior (enabling a tool that was previously absent) and could introduce unintended capabilities. `false` is the conservative, behavior-preserving choice.

### Consequences

#### Positive
- `gh aw fix --write` can now fully auto-repair all files affected by these two patterns without any manual intervention.
- gVisor's strict network isolation is preserved wherever it was already explicitly configured, avoiding unintended security downgrades.
- `tools.bash: false` satisfies the strict-mode compile requirement without changing runtime behavior for workflows that never relied on bash access.
- The fix pass no longer aborts an entire file when one codemod encounters an ambiguous case, allowing other codemods in the same file to run.

#### Negative
- Authors who had both `runtime: gvisor` and `sudo: true` with a genuine intent for privileged host access will have `sudo` silently dropped. The fix log records this, but the author must actively check it to notice.
- Auto-insertion of `tools.bash: false` is invisible to the author unless they diff the fixed file. Workflows that intended to add bash access later will need to update the field explicitly.

#### Neutral
- The `migrateSandboxAgentSecurityLines` function signature changed (added `oldRuntime` parameter, changed `hasRuntime bool` to a derived local variable) to support in-place rewriting of existing `runtime:` values. This is an internal refactor with no external API surface.
- Both codemods are registered in the standard codemod registry and covered by unit tests, following the existing extension pattern.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
